package ovirt

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	kloglogger "github.com/ovirt/go-ovirt-client-log-klog/v2"
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
	"gopkg.in/yaml.v3"
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

func NewClient() (ovirtclient.Client, error) {
	ovirtConfig, err := GetOvirtConfig()
	if err != nil {
		return nil, fmt.Errorf("Error getting ovirt config: %v", err)
	}

	ovirtConfig, err = ensureBase64PasswordInConfig(ovirtConfig)
	if err != nil {
		return nil, err
	}

	tls := ovirtclient.TLS()
	if ovirtConfig.Insecure {
		tls.Insecure()
	}
	if ovirtConfig.CAFile != "" {
		tls.CACertsFromFile(ovirtConfig.CAFile)
	}
	logger := kloglogger.New()
	//TODO: HANDLE VERBOSE
	client, err := ovirtclient.New(
		ovirtConfig.URL,
		ovirtConfig.Username,
		ovirtConfig.Password,
		tls,
		logger,
		nil,
	)

	return client, nil
}

// LoadOvirtConfig from the following location (first wins):
// 1. OVIRT_CONFIG env variable
// 2  $defaultOvirtConfigPath
func LoadOvirtConfig() ([]byte, error) {
	data, err := ioutil.ReadFile(DiscoverConfigFilePath())
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetOvirtConfig will return an Config by loading
// the configuration from locations specified in @LoadOvirtConfig
// error is return if the configuration could not be retained.
func GetOvirtConfig() (*Config, error) {
	c := Config{}
	in, err := LoadOvirtConfig()
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(in, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func DiscoverConfigFilePath() string {
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

	path := DiscoverConfigFilePath()
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
