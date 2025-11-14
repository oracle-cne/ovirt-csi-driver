// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk_profile

import (
	"fmt"
	"strings"

	"github.com/ovirt/csi-driver/pkg/ovirt/ovclient"
	"k8s.io/apimachinery/pkg/util/json"
	log "k8s.io/klog"
)

// GetDiskProfiles gets all disk profiles.
func GetDiskProfiles(ovcli *ovclient.Client) (*DiskProfileList, error) {
	const path = "/api/diskprofiles"

	// call the server
	body, err := ovcli.REST.Get(ovcli.AccessToken, path)
	if err != nil {
		err = fmt.Errorf("error doing HTTP GET: %v", err)
		return nil, err
	}

	diskProfileList := &DiskProfileList{}
	err = json.Unmarshal(body, diskProfileList)
	if err != nil {
		err = fmt.Errorf("error unmarshaling StorageDomains: %v", err)
		log.Error(err)
		return nil, err
	}

	return diskProfileList, nil
}

// GetDiskProfilesByName gets a list disk profile by name
func GetDiskProfilesByName(ovcli *ovclient.Client, profileName string) ([]*DiskProfile, error) {
	var matchList []*DiskProfile
	profileList, err := GetDiskProfiles(ovcli)
	if err != nil {
		return nil, err
	}

	for i, sd := range profileList.DiskProfiles {
		if strings.EqualFold(sd.Name, profileName) {
			matchList = append(matchList, &profileList.DiskProfiles[i])
		}
	}

	return matchList, nil
}
