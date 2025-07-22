// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diskprofile

import (
	"fmt"
	"github.com/ovirt/csi-driver/pkg/config"
	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/disk_profile"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	log "k8s.io/klog"
)

// SelectStorageDomainFromDiskProfile selects a storage domain from the list of storage domains
// associated with the specified disk profile name.
func SelectStorageDomainFromDiskProfile(config *config.Config, profileName string, policy string) (string, error) {
	domains, err := getStorageDomainsFromDiskProfile(config, profileName)
	if err != nil {
		return "", err
	}
	if domains == nil {
		return "", fmt.Errorf("No storage domains found for disk profile %s", profileName)
	}

	// filter out the domains that cannot be used
	domains, err = getUseableDomains(domains)
	if domains == nil {
		return "", fmt.Errorf("No storage domains found with the acceptable external status")
	}

	// pick the best domain given the policy
	domain, err := selectDomainUsingPolicy(domains, policy)
	if err != nil {
		return "", fmt.Errorf("Error selecting domain using policy %s: %s", policy, err.Error())
	}
	if domain == nil {
		return "", fmt.Errorf("No storage domain selected for disk profile %s", profileName)
	}

	log.Infof("Selected storage domain %s for disk profile %s", domain.Name, profileName)
	return domain.Name, nil
}

// Get all the storage domains associated with the disk profile name
func getStorageDomainsFromDiskProfile(config *config.Config, diskProfileName string) ([]*storagedomain.StorageDomain, error) {
	ovcli, err := ovclient.GetOVClient(config)
	if err != nil {
		return nil, fmt.Errorf("Error getting ovirt client: %s", err.Error())
	}

	// get the disk profiles by name
	diskProfiles, err := disk_profile.GetDiskProfilesByName(ovcli, diskProfileName)
	if err != nil {
		return nil, fmt.Errorf("Error getting disk profiles by disk profile name %s: %s", diskProfileName, err.Error())
	}

	// get all storage domains
	storageDomainList, err := storagedomain.GetStorageDomains(ovcli)
	if err != nil {
		return nil, fmt.Errorf("Error getting storage domain list: %s", err.Error())
	}

	// create a map of storage domains by storage domain id
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

	// return the list of storage domains that can potentially be used
	return sdList, nil
}

// return domains that have external status == ok/info
func getUseableDomains(domains []*storagedomain.StorageDomain) ([]*storagedomain.StorageDomain, error) {
	results := []*storagedomain.StorageDomain{}
	for i, d := range domains {
		if d.ExternalStatus == storagedomain.ExternalStatusOK ||
			d.ExternalStatus == storagedomain.ExternalStatusInfo {
			results = append(results, domains[i])
			log.Infof("Including storage domain %s in selection list", d.Name)
		} else {
			log.Infof("Ignoring storage domain %s. ExternalStatus is %s",
				d.Name, d.ExternalStatus)
		}
	}
	return results, nil
}
