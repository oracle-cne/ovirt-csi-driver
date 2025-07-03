package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"encoding/base64"
	"gopkg.in/yaml.v2"
)

const defaultOvirtConfigEnvVar = "OVIRT_CONFIG"

// Config holds oVirt api access details.
type Config struct {
	URL      string `yaml:"ovirt_url"`
	Username string `yaml:"ovirt_username"`
	Password string `yaml:"ovirt_password,omitempty"`
	Base64   string `yaml:"ovirt_base64,omitempty"`
	CAFile   string `yaml:"ovirt_cafile,omitempty"`
	Insecure bool   `yaml:"ovirt_insecure,omitempty"`
}

// GetOvirtConfig will return a Config by loading
// it from disk and ensuring that the password on disk is base64 encoded.
func GetOvirtConfig() (*Config, error) {
	ovirtConfig, err := getOvirtConfigFromDisk()
	if err != nil {
		return nil, fmt.Errorf("Error getting ovirt config: %v", err)
	}

	ovirtConfig, err = ensureBase64PasswordInConfig(ovirtConfig)
	if err != nil {
		return nil, err
	}

	return ovirtConfig, nil
}

// getOvirtConfigFromFile will return a Config by loading
// the configuration from locations specified in @LoadOvirtConfig
// error is return if the configuration could not be retained.
func getOvirtConfigFromDisk() (*Config, error) {
	c := Config{}
	in, err := loadOvirtConfig()
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(in, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// loadOvirtConfig from the following location (first wins):
// 1. OVIRT_CONFIG env variable
// 2  $defaultOvirtConfigPath
func loadOvirtConfig() ([]byte, error) {
	data, err := ioutil.ReadFile(discoverConfigFilePath())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func discoverConfigFilePath() string {
	path, _ := os.LookupEnv(defaultOvirtConfigEnvVar)
	if path != "" {
		return path
	}

	return filepath.Join(os.Getenv("HOME"), ".ovirt", "ovirt-config.yaml")
}

// Save will serialize the config back into the locations
// specified in @LoadOvirtConfig, first location with a file, wins.
func (c *Config) Save() error {
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	path := discoverConfigFilePath()
	return ioutil.WriteFile(path, out, os.FileMode(0600))
}

// ensureBase64PasswordInConfig ensures that the password on disk is in base64
func ensureBase64PasswordInConfig(config *Config) (*Config, error) {
	pw := config.Password
	if pw != "" {
		// password is in clear text. Base64 encode it and remove the clear-text password.
		pw = config.Password
		config.Base64 = base64.StdEncoding.EncodeToString([]byte(pw))
		config.Password = ""
		if err := config.Save(); err != nil {
			return nil, err
		}
	}
	if config.Base64 == "" {
		return nil, fmt.Errorf("Config file is missing both Password and PasswordBase64")
	}

	decoded, err := base64.StdEncoding.DecodeString(config.Base64)
	if err != nil {
		return nil, fmt.Errorf("Error decoding base64 password: %v", err)
	}

	config.Password = string(decoded)
	return config, nil
}
