package disk

import (
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/disk_profile"
)

func SelectStorageDomainsFromDiskProfile(config *config.Config, diskProfile string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, diskProfile)
	if err != nil {
		return "", err
	}

	return domains[0], nil
}

func getStorageDomainsFromDiskProfile(config *config.Config, diskProfile string) ([]string, error) {
	ovcli, err := ovclient.GetOVClient(config)
	if err != nil {
		return nil, err
	}

	disk_profile.GetDiskProfiles(ovcli)

	return nil, nil
}
