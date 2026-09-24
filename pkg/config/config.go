package config

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultOvirtConfigEnvVar = "OVIRT_CONFIG"

const (
	ovirtURLEnvVar      = "OVIRT_URL"
	ovirtUsernameEnvVar = "OVIRT_USERNAME"
	ovirtPasswordEnvVar = "OVIRT_PASSWORD"
	ovirtCAFileEnvVar   = "OVIRT_CAFILE"
	ovirtCABundleEnvVar = "OVIRT_CA_BUNDLE"
)

// PrepareOvirtConfigOptions controls the config file generated from environment variables.
type PrepareOvirtConfigOptions struct {
	ConfigPath string
	CAFilePath string
	Insecure   bool
	Logf       func(format string, args ...interface{})
}

// singleton config
var ovconfig *Config

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
	if ovconfig != nil {
		return ovconfig, nil
	}

	ovirtConfig, err := getOvirtConfigFromDisk()
	if err != nil {
		return nil, fmt.Errorf("Error getting ovirt config: %v", err)
	}

	ovirtConfig, err = ensureBase64PasswordInConfig(ovirtConfig)
	if err != nil {
		return nil, err
	}

	ovconfig = ovirtConfig
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

// PrepareOvirtConfigFromEnv writes an oVirt config file from environment variables.
func PrepareOvirtConfigFromEnv(options PrepareOvirtConfigOptions) error {
	logf := options.Logf
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}

	logf("Starting oVirt config preparation from environment variables")
	configPath := options.ConfigPath
	if configPath == "" {
		logf("No explicit config path provided; discovering path from %s or HOME", defaultOvirtConfigEnvVar)
		configPath = discoverConfigFilePath()
	}
	if configPath == "" {
		return fmt.Errorf("ovirt config path is empty")
	}
	logf("Resolved oVirt config output path: %s", configPath)

	url, err := requiredEnv(ovirtURLEnvVar, logf)
	if err != nil {
		return err
	}
	username, err := requiredEnv(ovirtUsernameEnvVar, logf)
	if err != nil {
		return err
	}
	password, err := requiredEnv(ovirtPasswordEnvVar, logf)
	if err != nil {
		return err
	}

	caBundle, caBundleProvided := os.LookupEnv(ovirtCABundleEnvVar)
	if caBundleProvided && caBundle != "" {
		logf("Detected non-empty %s; CA bundle file will be written", ovirtCABundleEnvVar)
	} else {
		logf("No non-empty %s detected; CA bundle file will not be written", ovirtCABundleEnvVar)
	}

	caFilePath := ""
	if caBundle != "" {
		caFilePath = options.CAFilePath
		if caFilePath == "" {
			logf("No explicit CA file path provided; checking %s", ovirtCAFileEnvVar)
			caFilePath, _ = os.LookupEnv(ovirtCAFileEnvVar)
		}
		if caFilePath == "" {
			return fmt.Errorf("%s must be set when %s is set", ovirtCAFileEnvVar, ovirtCABundleEnvVar)
		}
		logf("Resolved oVirt CA bundle output path: %s", caFilePath)
	}

	preparedConfig := Config{
		URL:      url,
		Username: username,
		Password: password,
		CAFile:   caFilePath,
		Insecure: options.Insecure,
	}

	logf("Serializing oVirt config content")
	out, err := yaml.Marshal(preparedConfigFile{
		URL:      preparedConfig.URL,
		Username: preparedConfig.Username,
		Password: preparedConfig.Password,
		CAFile:   preparedConfig.CAFile,
		Insecure: preparedConfig.Insecure,
	})
	if err != nil {
		return fmt.Errorf("error serializing ovirt config: %v", err)
	}

	logf("Ensuring oVirt config directory exists: %s", filepath.Dir(configPath))
	if err := os.MkdirAll(filepath.Dir(configPath), os.FileMode(0700)); err != nil {
		return fmt.Errorf("error creating ovirt config directory: %v", err)
	}

	logf("Writing oVirt config file: %s", configPath)
	if err := ioutil.WriteFile(configPath, out, os.FileMode(0600)); err != nil {
		return fmt.Errorf("error writing ovirt config file: %v", err)
	}

	if caBundle != "" {
		logf("Ensuring oVirt CA bundle directory exists: %s", filepath.Dir(caFilePath))
		if err := os.MkdirAll(filepath.Dir(caFilePath), os.FileMode(0700)); err != nil {
			return fmt.Errorf("error creating ovirt CA bundle directory: %v", err)
		}
		logf("Writing oVirt CA bundle file: %s", caFilePath)
		if err := ioutil.WriteFile(caFilePath, []byte(caBundle), os.FileMode(0600)); err != nil {
			return fmt.Errorf("error writing ovirt CA bundle file: %v", err)
		}
	}

	logf("Finished oVirt config preparation")
	return nil
}

type preparedConfigFile struct {
	URL      string `yaml:"ovirt_url"`
	Username string `yaml:"ovirt_username"`
	Password string `yaml:"ovirt_password"`
	CAFile   string `yaml:"ovirt_cafile"`
	Insecure bool   `yaml:"ovirt_insecure"`
}

func requiredEnv(name string, logf func(format string, args ...interface{})) (string, error) {
	logf("Reading required environment variable: %s", name)
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("%s must be set", name)
	}
	logf("Found required environment variable: %s", name)
	return value, nil
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
