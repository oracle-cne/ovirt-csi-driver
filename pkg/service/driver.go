package service

import (
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// set by ldflags
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
	var d *OvirtCSIDriver
	if string(nodeId) == "" {
		klog.Info("Creating driver for controller")

		// controller plugin
		d = &OvirtCSIDriver{
			IdentityService:   &IdentityService{ovirtClient},
			ControllerService: &ControllerService{ovirtClient: ovirtClient},
			ovirtClient:       ovirtClient,
		}
	} else {
		klog.Info("Creating driver for node")

		// node plugin
		d = &OvirtCSIDriver{
			NodeService: &NodeService{nodeId: nodeId, ovirtClient: ovirtClient},
			ovirtClient: ovirtClient,
		}
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
