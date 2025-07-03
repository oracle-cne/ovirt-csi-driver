package disk

import (
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
)

func SelectStorageDomainsFromDiskProfile(config *config.Config, diskProfile string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, diskProfile)
	if err != nil {

	}

}

func getStorageDomainsFromDiskProfile(config *config.Config, diskProfile string) ([]string, error) {

	// func GetOVClient(cli kubernetes.Interface, caMap map[string]string, apiServerURL string, insecureSkipTLSVerify bool) (*Client, error) {

	ovcli, err := ovclient.GetOVClient(config)

}
