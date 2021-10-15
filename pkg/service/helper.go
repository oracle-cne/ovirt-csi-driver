package service

import (
	"fmt"

	ovirtsdk "github.com/ovirt/go-ovirt"
)

type AttachmentNotFoundError struct {
	vmId   string
	diskId string
}

func (e *AttachmentNotFoundError) Error() string {
	return fmt.Sprintf("failed to find attachment by disk %s for VM %s", e.diskId, e.vmId)
}

func diskAttachmentByVmAndDisk(connection *ovirtsdk.Connection, vmId string, diskId string) (*ovirtsdk.DiskAttachment, error) {
	vmService := connection.SystemService().VmsService().VmService(vmId)
	attachments, err := vmService.DiskAttachmentsService().List().Send()
	if err != nil {
		return nil, err
	}

	for _, attachment := range attachments.MustAttachments().Slice() {
		if diskId == attachment.MustDisk().MustId() {
			return attachment, nil
		}
	}
	return nil, &AttachmentNotFoundError{
		vmId:   vmId,
		diskId: diskId,
	}
}

func getStorageDomainByName(conn *ovirtsdk.Connection, storageDomainName string) (*ovirtsdk.StorageDomain, error) {
	searchString := fmt.Sprintf("name=%s", storageDomainName)
	sdByName, err := conn.SystemService().StorageDomainsService().List().Search(searchString).Send()
	if err != nil {
		return nil, err
	}
	sd, ok := sdByName.StorageDomains()
	if !ok {
		return nil, fmt.Errorf(
			"error, failed searching for storage domain with name %s", storageDomainName)
	}
	if len(sd.Slice()) > 1 {
		return nil, fmt.Errorf(
			"error, found more then one storage domain with the name %s, please use ID instead", storageDomainName)
	}
	if len(sd.Slice()) == 0 {
		return nil, nil
	}
	return sd.Slice()[0], nil
}

func isFileDomain(storageType ovirtsdk.StorageType) bool {
	switch storageType {
	case ovirtsdk.STORAGETYPE_NFS, ovirtsdk.STORAGETYPE_GLUSTERFS, ovirtsdk.STORAGETYPE_POSIXFS, ovirtsdk.STORAGETYPE_LOCALFS:
		return true
	default:
		return false
	}
}
