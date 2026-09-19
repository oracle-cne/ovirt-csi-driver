package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func TestLivenessProbeConfiguration(t *testing.T) {
	if healthProbeBindAddress != ":9808" {
		t.Fatalf("health probe bind address is %q, want %q", healthProbeBindAddress, ":9808")
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
