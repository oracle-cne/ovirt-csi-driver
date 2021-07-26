package service

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/ovirt/csi-driver/internal/ovirt"
	ovirtsdk "github.com/ovirt/go-ovirt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ParameterStorageDomainName = "storageDomainName"
	ParameterThinProvisioning  = "thinProvisioning"
	minimumDiskSize            = 1 * 1024 * 1024
)

//ControllerService implements the controller interface
type ControllerService struct {
	ovirtClient *ovirt.Client
	client      client.Client
}

var ControllerCaps = []csi.ControllerServiceCapability_RPC_Type{
	csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME, // attach/detach
}

//CreateVolume creates the disk for the request, unattached from any VM
func (c *ControllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("Creating disk %s", req.Name)
	storageDomain := req.Parameters[ParameterStorageDomainName]
	if len(storageDomain) == 0 {
		return nil, fmt.Errorf("error required storageClass paramater %s wasn't set",
			ParameterStorageDomainName)
	}
	diskName := req.Name
	if len(diskName) == 0 {
		return nil, fmt.Errorf("error required request parameter Name was not provided")
	}
	thinProvisioning, err := strconv.ParseBool(req.Parameters[ParameterThinProvisioning])
	if req.Parameters[ParameterThinProvisioning] == "" {
		// In case thin provisioning is not set, we default to true
		thinProvisioning = true
	}
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse storage class field %s, expected 'true' or 'false' but got %s",
			ParameterThinProvisioning, thinProvisioning)
	}
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	diskByName, err := conn.SystemService().DisksService().List().Search(diskName).Send()
	if err != nil {
		return nil, err
	}

	// if exists we're done
	if disks, ok := diskByName.Disks(); ok && len(disks.Slice()) == 1 {
		disk := disks.Slice()[0]
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes:      disk.MustProvisionedSize(),
				VolumeId:           disk.MustId(),
				VolumeContext:      nil,
				ContentSource:      nil,
				AccessibleTopology: nil,
			},
		}, nil
	}

	provisionedSize := req.CapacityRange.GetRequiredBytes()
	if provisionedSize < minimumDiskSize {
		provisionedSize = minimumDiskSize
	}

	imageFormat, err := handleCreateVolumeImageFormat(conn, storageDomain, thinProvisioning)
	if err != nil {
		msg := fmt.Errorf("error while choosing image format, error is %w", err)
		klog.Errorf(msg.Error())
		return nil, msg
	}

	// creating the disk
	disk, err := ovirtsdk.NewDiskBuilder().
		Name(diskName).
		StorageDomainsBuilderOfAny(*ovirtsdk.NewStorageDomainBuilder().Name(storageDomain)).
		ProvisionedSize(provisionedSize).
		ReadOnly(false).
		Format(imageFormat).
		Sparse(thinProvisioning).
		Build()

	if err != nil {
		// failed to construct the disk
		return nil, err
	}

	createDisk, err := conn.SystemService().DisksService().
		Add().
		Disk(disk).
		Send()
	if err != nil {
		// failed to create the disk
		klog.Errorf("Failed creating disk %s", diskName)
		return nil, err
	}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: createDisk.MustDisk().MustProvisionedSize(),
			VolumeId:      createDisk.MustDisk().MustId(),
		},
	}, nil
}

func handleCreateVolumeImageFormat(conn *ovirtsdk.Connection, storageDomainName string, thinProvisioning bool) (ovirtsdk.DiskFormat, error) {
	sd, err := getStorageDomainByName(conn, storageDomainName)
	if err != nil {
		return "", fmt.Errorf(
			"failed searching for storage domain with name %s, error: %w", storageDomainName, err)
	}
	if sd == nil {
		return "", fmt.Errorf(
			"storage domain with name %s wasn't found", storageDomainName)
	}
	storage, ok := sd.Storage()
	if !ok {
		return "", fmt.Errorf(
			"storage domain with name %s didn't have host storage, veify it is connected to a host",
			storageDomainName)
	}
	storageType, ok := storage.Type()
	if !ok {
		return "", fmt.Errorf(
			"storage domain with name %s didn't have a storage type, please check storage domain on ovirt engine",
			storageDomainName)
	}
	// Use COW diskformat only when thin provisioning is requested and storage domain
	// is a non file storage type (for example ISCSI)
	if !isFileDomain(storageType) && thinProvisioning {
		return ovirtsdk.DISKFORMAT_COW, nil
	} else {
		return ovirtsdk.DISKFORMAT_RAW, nil
	}
}

//DeleteVolume removed the disk from oVirt
func (c *ControllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.Infof("Removing disk %s", req.VolumeId)
	// idempotence first - see if disk already exists, ovirt creates disk by name(alias in ovirt as well)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	diskService := conn.SystemService().DisksService().DiskService(req.VolumeId)

	_, err = diskService.Get().Send()
	// if doesn't exists we're done
	if err != nil {
		return &csi.DeleteVolumeResponse{}, nil
	}
	_, err = diskService.Remove().Send()
	if err != nil {
		return nil, err
	}

	klog.Infof("Finished removing disk %s", req.VolumeId)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume takes a volume, which is an oVirt disk, and attaches it to a node, which is an oVirt VM.
func (c *ControllerService) ControllerPublishVolume(
	ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {

	klog.Infof("Attaching Disk %s to VM %s", req.VolumeId, req.NodeId)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	_, err = diskAttachmentByVmAndDisk(conn, req.NodeId, req.VolumeId)
	if err != nil {
		// If attachment was not found we are good to
		attachmentNotFoundErr := &AttachmentNotFoundError{}
		if !errors.As(err, &attachmentNotFoundErr) {
			klog.Error("Failed to list attachments", err)
			return nil, err
		}
	} else {
		klog.Infof("Disk %s is already attached to VM %s, returning OK", req.VolumeId, req.NodeId)
		return &csi.ControllerPublishVolumeResponse{}, nil
	}

	vmService := conn.SystemService().VmsService().VmService(req.NodeId)

	attachmentBuilder := ovirtsdk.NewDiskAttachmentBuilder().
		DiskBuilder(ovirtsdk.NewDiskBuilder().Id(req.VolumeId)).
		Interface(ovirtsdk.DISKINTERFACE_VIRTIO_SCSI).
		Bootable(false).
		Active(true)

	_, err = vmService.
		DiskAttachmentsService().
		Add().
		Attachment(attachmentBuilder.MustBuild()).
		Send()
	if err != nil {
		return nil, err
	}
	klog.Infof("Attached Disk %v to VM %s", req.VolumeId, req.NodeId)
	return &csi.ControllerPublishVolumeResponse{}, nil
}

//ControllerUnpublishVolume detaches the disk from the VM.
func (c *ControllerService) ControllerUnpublishVolume(_ context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.Infof("Detaching Disk %s from VM %s", req.VolumeId, req.NodeId)
	conn, err := c.ovirtClient.GetConnection()
	if err != nil {
		klog.Errorf("Failed to get ovirt client connection")
		return nil, err
	}

	attachment, err := diskAttachmentByVmAndDisk(conn, req.NodeId, req.VolumeId)
	if err != nil {
		klog.Errorf("Failed to get disk attachment %s for VM %s, returning OK", req.VolumeId, req.NodeId)
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}
	_, err = conn.SystemService().VmsService().VmService(req.NodeId).
		DiskAttachmentsService().
		AttachmentService(attachment.MustId()).
		Remove().
		Send()

	if err != nil {
		return nil, err
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

//ValidateVolumeCapabilities
func (c *ControllerService) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ListVolumes
func (c *ControllerService) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//GetCapacity
func (c *ControllerService) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//CreateSnapshot
func (c *ControllerService) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//DeleteSnapshot
func (c *ControllerService) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ListSnapshots
func (c *ControllerService) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ControllerExpandVolume
func (c *ControllerService) ControllerExpandVolume(context.Context, *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//ControllerGetCapabilities
func (c *ControllerService) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	caps := make([]*csi.ControllerServiceCapability, 0, len(ControllerCaps))
	for _, capability := range ControllerCaps {
		caps = append(
			caps,
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: capability,
					},
				},
			},
		)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}
