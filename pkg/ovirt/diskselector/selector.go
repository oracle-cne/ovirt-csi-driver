package diskselector

import (
	"fmt"
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/disk_profile"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	log "k8s.io/klog"
)

func SelectStorageDomainFromDiskProfile(config *config.Config, profileName string, policy string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, profileName)
	if err != nil {
		return "", err
	}
	if domains == nil {
		return "", fmt.Errorf("no storage domains found for disk profile %s", profileName)
	}

	// filter out the domains that cannot be used
	domains, err = filterDomains(domains)
	if domains == nil {
		return "", fmt.Errorf("no storage domains found with the acceptable status or external status")
	}

	domain, err := selectDomainUsingPolicy(domains, policy)
	if err != nil {
		return "", fmt.Errorf("error selecting domain using policy %s: %v", policy, err)
	}
	if domain == nil {
		return "", fmt.Errorf("no storage domain selected for disk profile %s", profileName)
	}

	// For now use the first one.
	log.Infof("using storage domain %s for disk profile %s", domain.Name, profileName)
	return domain.Name, nil
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

// return domains that have status == active and external status == ok/info
func filterDomains(domains []*storagedomain.StorageDomain) ([]*storagedomain.StorageDomain, error) {
	results := []*storagedomain.StorageDomain{}
	for i, d := range domains {
		if d.Status == storagedomain.StatusActive &&
			(d.ExternalStatus == storagedomain.ExternalStatusOK || d.ExternalStatus == storagedomain.ExternalStatusInfo) {
			results = append(results, domains[i])
			log.Infof("Including storage domain %s in selection list", d.Name)
		} else {
			log.Infof("Ignoring storage domain %s. Status is %s, ExternalStatus is %s",
				d.Name, d.Status, d.ExternalStatus)
		}
	}
	return results, nil
}
