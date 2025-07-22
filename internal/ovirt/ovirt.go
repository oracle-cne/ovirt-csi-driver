package ovirt

import (
	"fmt"
	"github.com/ovirt/csi-driver/pkg/config"
	kloglogger "github.com/ovirt/go-ovirt-client-log-klog/v2"
	ovirtclient "github.com/ovirt/go-ovirt-client/v2"
)

func NewClient() (ovirtclient.Client, error) {
	ovirtConfig, err := config.GetOvirtConfig()
	if err != nil {
		return nil, fmt.Errorf("Error getting ovirt config: %v", err)
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
