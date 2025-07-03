package disk

import (
	"fmt"
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/disk_profile"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	log "k8s.io/klog"
)

func SelectStorageDomainFromDiskProfile(config *config.Config, profileName string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, profileName)
	if err != nil {
		return "", err
	}

	if domains == nil {
		return "", fmt.Errorf("no storage domains found for disk profile %s", profileName)
	}

	// For now use the first one.
	log.Infof("found %d storage domain(s) for disk profile %s", len(domains), profileName)
	return domains[0].Name, nil
}

func getStorageDomainsFromDiskProfile(config *config.Config, diskProfileName string) ([]*storagedomain.StorageDomain, error) {
	ovcli, err := ovclient.GetOVClient(config)
	if err != nil {
		return nil, fmt.Errorf("error getting ovirt client: %s", err.Error())
	}

	// get the disk profiles by name
	diskProfiles, err := disk_profile.GetDiskProfilesByName(ovcli, diskProfileName)
	if err != nil {
		return nil, fmt.Errorf("error getting disk profiles by disk profile name %s: %s", diskProfileName, err.Error())
	}

	// get all storage domains
	storageDomainList, err := storagedomain.GetStorageDomains(ovcli)
	if err != nil {
		return nil, fmt.Errorf("error getting storage domain list: %s", err.Error())
	}

	// create a map of storage domains by id
	sdMap := make(map[string]*storagedomain.StorageDomain)
	for i, sd := range storageDomainList.StorageDomains {
		sdMap[sd.Id] = &storageDomainList.StorageDomains[i]
	}

	// build a list of storage domains where the domain is being used by a profile
	sdList := []*storagedomain.StorageDomain{}
	for _, dp := range diskProfiles {
		sd, ok := sdMap[dp.StorageDomain.Id]
		if !ok {
			continue
		}
		sdList = append(sdList, sd)
	}

	// build the list of storage domains
	return sdList, nil
}
