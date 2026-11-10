package main

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func TestLivenessProbeConfiguration(t *testing.T) {
	if *livenessPort != defaultLivenessPort {
		t.Fatalf("liveness port is %d, want %d", *livenessPort, defaultLivenessPort)
	}
	if address := livenessBindAddress(defaultLivenessPort); address != ":9808" {
		t.Fatalf("default liveness bind address is %q, want %q", address, ":9808")
	}
	if address := livenessBindAddress(12345); address != ":12345" {
		t.Fatalf("configured liveness bind address is %q, want %q", address, ":12345")
	}
	if livenessEndpointName != "/livez" {
		t.Fatalf("liveness endpoint name is %q, want %q", livenessEndpointName, "/livez")
	}

	handler := &healthz.Handler{
		Checks: map[string]healthz.Checker{"ping": healthz.Ping},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, livenessEndpointName, nil)

	http.StripPrefix(livenessEndpointName, handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("liveness probe returned status %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "ok") {
		t.Fatalf("liveness probe returned body %q, want it to contain %q", body, "ok")
	}
}

func TestTokenRenewalArguments(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		interval   time.Duration
		percentage int
		invalid    bool
	}{
		{name: "default", percentage: 90},
		{name: "duration", args: []string{"--token-renewal-interval=5m"}, interval: 5 * time.Minute},
		{name: "percentage", args: []string{"--token-renewal-percentage=75"}, percentage: 75},
		{name: "both", args: []string{"--token-renewal-interval=5m", "--token-renewal-percentage=90"}, invalid: true},
		{name: "zero duration", args: []string{"--token-renewal-interval=0"}, invalid: true},
		{name: "negative duration", args: []string{"--token-renewal-interval=-5m"}, invalid: true},
		{name: "bad duration", args: []string{"--token-renewal-interval=five"}, invalid: true},
		{name: "zero percentage", args: []string{"--token-renewal-percentage=0"}, invalid: true},
		{name: "negative percentage", args: []string{"--token-renewal-percentage=-1"}, invalid: true},
		{name: "expiry percentage", args: []string{"--token-renewal-percentage=100"}, invalid: true},
		{name: "large percentage", args: []string{"--token-renewal-percentage=101"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Validate CLI policy selection before connecting to either API")
			flags := flag.NewFlagSet(tc.name, flag.ContinueOnError)
			flags.SetOutput(io.Discard)
			flags.Duration("token-renewal-interval", 0, "")
			flags.Int("token-renewal-percentage", 90, "")
			err := flags.Parse(tc.args)
			var policy ovirtclient.ReconnectPolicy
			if err == nil {
				policy, err = tokenRenewalPolicy(flags)
			}
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid renewal arguments accepted")
				}
				return
			}
			if err != nil || policy.Interval != tc.interval || policy.LifetimePercentage != tc.percentage {
				t.Fatalf("policy=%+v error=%v", policy, err)
			}
		})
	}
}
