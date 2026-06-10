package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPrepareOvirtConfigFromEnvWritesConfigAndCAFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config", "ovirt-config.yaml")
	caFilePath := filepath.Join(dir, "config", "ovirt-engine-ca.pem")
	caBundle := "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n"

	t.Setenv(ovirtURLEnvVar, "https://engine.example.test/ovirt-engine/api")
	t.Setenv(ovirtUsernameEnvVar, "admin@internal")
	t.Setenv(ovirtPasswordEnvVar, "secret: password")
	t.Setenv(ovirtCABundleEnvVar, caBundle)

	var logs []string
	err := PrepareOvirtConfigFromEnv(PrepareOvirtConfigOptions{
		ConfigPath: configPath,
		CAFilePath: caFilePath,
		Insecure:   true,
		Logf: func(format string, args ...interface{}) {
			logs = append(logs, format)
		},
	})
	if err != nil {
		t.Fatalf("PrepareOvirtConfigFromEnv returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read generated config file: %v", err)
	}
	generatedConfig := Config{}
	if err := yaml.Unmarshal(configData, &generatedConfig); err != nil {
		t.Fatalf("failed to unmarshal generated config file: %v", err)
	}
	if generatedConfig.URL != "https://engine.example.test/ovirt-engine/api" {
		t.Fatalf("unexpected generated URL: %q", generatedConfig.URL)
	}
	if generatedConfig.Username != "admin@internal" {
		t.Fatalf("unexpected generated username: %q", generatedConfig.Username)
	}
	if generatedConfig.Password != "secret: password" {
		t.Fatalf("unexpected generated password: %q", generatedConfig.Password)
	}
	if generatedConfig.CAFile != caFilePath {
		t.Fatalf("unexpected generated CA file path: %q", generatedConfig.CAFile)
	}
	if !generatedConfig.Insecure {
		t.Fatalf("expected generated config to set insecure mode")
	}

	caData, err := os.ReadFile(caFilePath)
	if err != nil {
		t.Fatalf("failed to read generated CA file: %v", err)
	}
	if string(caData) != caBundle {
		t.Fatalf("unexpected generated CA file content: %q", string(caData))
	}
	assertFileMode(t, configPath, 0600)
	assertFileMode(t, caFilePath, 0600)

	if len(logs) == 0 {
		t.Fatalf("expected verbose flow logging")
	}
}

func TestPrepareOvirtConfigFromEnvOmitsCAFileWhenCABundleMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "ovirt-config.yaml")

	t.Setenv(ovirtURLEnvVar, "https://engine.example.test/ovirt-engine/api")
	t.Setenv(ovirtUsernameEnvVar, "admin@internal")
	t.Setenv(ovirtPasswordEnvVar, "secret")
	t.Setenv(ovirtCAFileEnvVar, "/tmp/unused-ca-file.pem")

	if err := PrepareOvirtConfigFromEnv(PrepareOvirtConfigOptions{ConfigPath: configPath}); err != nil {
		t.Fatalf("PrepareOvirtConfigFromEnv returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read generated config file: %v", err)
	}
	generatedConfig := Config{}
	if err := yaml.Unmarshal(configData, &generatedConfig); err != nil {
		t.Fatalf("failed to unmarshal generated config file: %v", err)
	}
	if generatedConfig.CAFile != "" {
		t.Fatalf("expected generated CA file path to be empty, got %q", generatedConfig.CAFile)
	}
	if generatedConfig.Insecure {
		t.Fatalf("expected generated config to default insecure mode to false")
	}
}

func TestPrepareOvirtConfigFromEnvUsesDefaultPathsFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ovirt-config.yaml")
	caFilePath := filepath.Join(dir, "ovirt-engine-ca.pem")

	t.Setenv(defaultOvirtConfigEnvVar, configPath)
	t.Setenv(ovirtURLEnvVar, "https://engine.example.test/ovirt-engine/api")
	t.Setenv(ovirtUsernameEnvVar, "admin@internal")
	t.Setenv(ovirtPasswordEnvVar, "secret")
	t.Setenv(ovirtCABundleEnvVar, "ca")
	t.Setenv(ovirtCAFileEnvVar, caFilePath)

	if err := PrepareOvirtConfigFromEnv(PrepareOvirtConfigOptions{}); err != nil {
		t.Fatalf("PrepareOvirtConfigFromEnv returned error: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected generated config file at OVIRT_CONFIG path: %v", err)
	}
	if _, err := os.Stat(caFilePath); err != nil {
		t.Fatalf("expected generated CA file at OVIRT_CAFILE path: %v", err)
	}
}

func TestPrepareOvirtConfigFromEnvRequiresInputs(t *testing.T) {
	tests := []struct {
		name       string
		missingEnv string
	}{
		{name: "url", missingEnv: ovirtURLEnvVar},
		{name: "username", missingEnv: ovirtUsernameEnvVar},
		{name: "password", missingEnv: ovirtPasswordEnvVar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(ovirtURLEnvVar, "https://engine.example.test/ovirt-engine/api")
			t.Setenv(ovirtUsernameEnvVar, "admin@internal")
			t.Setenv(ovirtPasswordEnvVar, "secret")
			t.Setenv(tt.missingEnv, "")

			err := PrepareOvirtConfigFromEnv(PrepareOvirtConfigOptions{
				ConfigPath: filepath.Join(t.TempDir(), "ovirt-config.yaml"),
			})
			if err == nil {
				t.Fatalf("expected error when %s is empty", tt.missingEnv)
			}
			if !strings.Contains(err.Error(), tt.missingEnv) {
				t.Fatalf("expected error to reference %s, got %v", tt.missingEnv, err)
			}
		})
	}
}

func TestPrepareOvirtConfigFromEnvRequiresCAFileWhenCABundleProvided(t *testing.T) {
	t.Setenv(ovirtURLEnvVar, "https://engine.example.test/ovirt-engine/api")
	t.Setenv(ovirtUsernameEnvVar, "admin@internal")
	t.Setenv(ovirtPasswordEnvVar, "secret")
	t.Setenv(ovirtCABundleEnvVar, "ca")
	t.Setenv(ovirtCAFileEnvVar, "")

	err := PrepareOvirtConfigFromEnv(PrepareOvirtConfigOptions{
		ConfigPath: filepath.Join(t.TempDir(), "ovirt-config.yaml"),
	})
	if err == nil {
		t.Fatalf("expected error when CA bundle is set without a CA file path")
	}
	if !strings.Contains(err.Error(), ovirtCAFileEnvVar) {
		t.Fatalf("expected error to reference %s, got %v", ovirtCAFileEnvVar, err)
	}
}

func assertFileMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}
	if info.Mode().Perm() != expected {
		t.Fatalf("unexpected mode for %s: got %v, want %v", path, info.Mode().Perm(), expected)
	}
}
