package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

// The production client implements reconnect in addition to the base API. Keep
// this fake separate so the existing tests also cover clients without reconnect.
type scheduledReconnectProbeClient struct {
	probeClient
	reconnected    bool
	reconnectErr   error
	reconnectCalls int
	events         []string
	policy         ovirtclient.ReconnectPolicy
}

func (c *scheduledReconnectProbeClient) ReconnectIfDue(policies ...ovirtclient.ReconnectPolicy) (bool, error) {
	c.policy = policies[0]
	c.reconnectCalls++
	c.events = append(c.events, "reconnect")
	return c.reconnected, c.reconnectErr
}

func (c *scheduledReconnectProbeClient) Test(retries ...ovirtclient.RetryStrategy) error {
	c.events = append(c.events, "test")
	return c.probeClient.Test(retries...)
}

func TestProbeScheduledReconnect(t *testing.T) {
	for _, tc := range []struct {
		name          string
		reconnected   bool
		reconnectErr  error
		connectionErr error
		wantReady     bool
	}{
		{name: "not due", wantReady: true},
		{name: "reconnected", reconnected: true, wantReady: true},
		{name: "reconnect failed with usable old token", reconnectErr: errors.New("SSO unavailable"), wantReady: true},
		{name: "reconnect failed with expired old token", reconnectErr: errors.New("SSO unavailable"), connectionErr: errors.New("unauthorized")},
		{name: "replacement rejected by API", reconnected: true, connectionErr: errors.New("unauthorized")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Run consecutive probes to check reconnect, authentication and retry ordering")
			client := &scheduledReconnectProbeClient{
				probeClient:  probeClient{err: tc.connectionErr},
				reconnected:  tc.reconnected,
				reconnectErr: tc.reconnectErr,
			}
			driver := service.NewOvirtCSIDriver(client, "")
			for n := 1; n <= 2; n++ {
				response, err := driver.Probe(context.Background(), &csi.ProbeRequest{})
				if tc.wantReady {
					if err != nil || response.GetReady() == nil || !response.GetReady().Value {
						t.Fatalf("probe should be ready: response=%v error=%v", response, err)
					}
				} else if status.Code(err) != codes.FailedPrecondition || response != nil {
					t.Fatalf("probe should fail readiness: response=%v error=%v", response, err)
				}
				if client.reconnectCalls != n || client.calls != n {
					t.Fatalf("unexpected reconnect/test behavior: reconnects=%d tests=%d", client.reconnectCalls, client.calls)
				}
				if client.events[2*n-2] != "reconnect" || client.events[2*n-1] != "test" {
					t.Fatalf("reconnect must precede the connection test: %v", client.events)
				}
			}
		})
	}
}

func TestProbePassesConfiguredRenewalPolicy(t *testing.T) {
	for _, policy := range []ovirtclient.ReconnectPolicy{
		{Interval: 5 * time.Minute}, {LifetimePercentage: 75},
	} {
		t.Logf("Verify probes pass the configured policy to unified reconnect: %+v", policy)
		client := &scheduledReconnectProbeClient{}
		driver := service.NewOvirtCSIDriver(client, "")
		if driver.TokenRenewalPolicy != (ovirtclient.ReconnectPolicy{LifetimePercentage: 90}) {
			t.Fatal("default renewal policy changed")
		}
		driver.TokenRenewalPolicy = policy
		if _, err := driver.Probe(context.Background(), &csi.ProbeRequest{}); err != nil {
			t.Fatal(err)
		}
		if client.policy != policy {
			t.Fatalf("received policy %+v, want %+v", client.policy, policy)
		}
	}
}
