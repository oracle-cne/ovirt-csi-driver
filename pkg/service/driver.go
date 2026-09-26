package service

import (
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// VendorVersion - set by ldflags
	VendorVersion = "0.1.1"
	VendorName    = "csi.ovirt.org"
)

type OvirtCSIDriver struct {
	*IdentityService
	*ControllerService
	*NodeService
	nodeId      string
	ovirtClient ovirtclient.Client
	Client      client.Client
}

// NewOvirtCSIDriver creates a driver instance
func NewOvirtCSIDriver(ovirtClient ovirtclient.Client, nodeId ovirtclient.VMID) *OvirtCSIDriver {
	d := &OvirtCSIDriver{
		IdentityService: &IdentityService{ovirtClient},
		ovirtClient:     ovirtClient,
	}
	if ovirtClient == nil {
		return d
	}

	if string(nodeId) == "" {
		klog.Info("Creating driver for controller")
		d.ControllerService = &ControllerService{ovirtClient: ovirtClient}
	} else {
		klog.Info("Creating driver for node")
		d.NodeService = &NodeService{nodeId: nodeId, ovirtClient: ovirtClient}
	}

	return d
}

// Run will initiate the grpc services Identity, Controller, and Node.
func (driver *OvirtCSIDriver) Run(endpoint string) {
	// run the gRPC server
	klog.Info("Setting the rpc server")

	s := NewNonBlockingGRPCServer()
	s.Start(endpoint, driver.IdentityService, driver.ControllerService, driver.NodeService)
	s.Wait()
}
