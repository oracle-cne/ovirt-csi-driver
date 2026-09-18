package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/ovirt/csi-driver/pkg/service"
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type probeClient struct {
	ovirtclient.Client
	err   error
	calls int
}

func (c *probeClient) Test(...ovirtclient.RetryStrategy) error {
	c.calls++
	return c.err
}

func TestProbeReady(t *testing.T) {
	client := &probeClient{}
	driver := service.NewOvirtCSIDriver(client, "")

	response, err := driver.Probe(context.Background(), &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe returned an unexpected error: %v", err)
	}
	if response.GetReady() == nil || !response.GetReady().Value {
		t.Fatal("Probe did not report the driver ready")
	}
	if client.calls != 1 {
		t.Fatalf("connection test called %d times, want 1", client.calls)
	}
}

func TestProbeConnectionFailure(t *testing.T) {
	client := &probeClient{err: errors.New("backend unavailable")}
	driver := service.NewOvirtCSIDriver(client, "")

	response, err := driver.Probe(context.Background(), &csi.ProbeRequest{})
	if response != nil {
		t.Fatalf("Probe returned an unexpected response: %#v", response)
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Probe error code was %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
	if client.calls != 1 {
		t.Fatalf("connection test called %d times, want 1", client.calls)
	}
}

func TestProbeInitializationFailure(t *testing.T) {
	for _, nodeID := range []ovirtclient.VMID{"", "node-id"} {
		driver := service.NewOvirtCSIDriver(nil, nodeID)

		response, err := driver.Probe(context.Background(), &csi.ProbeRequest{})
		if response != nil {
			t.Fatalf("Probe returned an unexpected response: %#v", response)
		}
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("Probe error code was %s, want %s", status.Code(err), codes.FailedPrecondition)
		}
		if driver.ControllerService != nil || driver.NodeService != nil {
			t.Fatal("driver exposed a controller or node service without an oVirt client")
		}
	}
}
