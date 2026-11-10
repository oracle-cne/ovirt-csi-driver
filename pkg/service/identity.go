package service

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

type tokenReconnector interface {
	ReconnectIfDue(...ovirtclient.ReconnectPolicy) (bool, error)
}

// IdentityService of ovirt-csi-driver
type IdentityService struct {
	ovirtClient ovirtclient.Client
	// TokenRenewalPolicy is configured before the driver starts serving probes.
	TokenRenewalPolicy ovirtclient.ReconnectPolicy
}

// GetPluginInfo returns the vendor name and version - set in build time
func (i *IdentityService) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          VendorName,
		VendorVersion: VendorVersion,
	}, nil
}

// GetPluginCapabilities declares the plugins capabilities
func (i *IdentityService) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_VolumeExpansion_{
					VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
						Type: csi.PluginCapability_VolumeExpansion_ONLINE,
					},
				},
			},
		},
	}, nil
}

// Probe checks the state of the connection to ovirt-engine
func (i *IdentityService) Probe(_ context.Context, _ *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	if i.ovirtClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "Could not initialize connection to ovirt-engine")
	}

	if reconnector, ok := i.ovirtClient.(tokenReconnector); ok {
		klog.V(4).Infof("Checking oVirt reconnect: interval=%s lifetime-percentage=%d", i.TokenRenewalPolicy.Interval, i.TokenRenewalPolicy.LifetimePercentage)
		if reconnected, err := reconnector.ReconnectIfDue(i.TokenRenewalPolicy); err != nil {
			// Keep serving with the old token if it still works. A failed renewal
			// does not reset its deadline, so the next probe will retry.
			klog.Warning("oVirt scheduled reconnect failed; testing the current token and retrying on the next probe")
		} else if reconnected {
			klog.Info("oVirt reconnected; testing the active connection")
		} else {
			klog.V(4).Info("oVirt reconnect is not due; testing the current connection")
		}
	}
	err := i.ovirtClient.Test()
	if err != nil {
		klog.Errorf("Could not get connection %v", err)
		return nil, status.Error(codes.FailedPrecondition, "Could not get connection to ovirt-engine")
	}
	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}, nil
}
