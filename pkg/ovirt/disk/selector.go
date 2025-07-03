package disk

import (
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/disk_profile"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	log "k8s.io/klog"
)

func SelectStorageDomainFromDiskProfile(config *config.Config, diskProfile string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, diskProfile)
	if err != nil {
		return "", err
	}

	if domains == nil {
		return "", nil
	}

	// For now use the first one.
	log.Infof("Found %d storage domain(s)", len(domains))
	return domains[0].Name, nil
}

func getStorageDomainsFromDiskProfile(config *config.Config, diskProfileName string) ([]*storagedomain.StorageDomain, error) {
	ovcli, err := ovclient.GetOVClient(config)
	if err != nil {
		return nil, err
	}

	// get the disk profiles by name
	diskProfiles, err := disk_profile.GetDiskProfilesByName(ovcli, diskProfileName)
	if err != nil {
		return nil, err
	}

	// get all storage domains
	storageDomainList, err := storagedomain.GetStorageDomains(ovcli)
	if err != nil {
		return nil, err
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
