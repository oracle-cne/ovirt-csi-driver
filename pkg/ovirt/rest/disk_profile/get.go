// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk_profile

import (
	"fmt"
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
		err = fmt.Errorf("Error doing HTTP GET: %v", err)
		return nil, err
	}

	diskProfileList := &DiskProfileList{}
	err = json.Unmarshal(body, diskProfileList)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling StorageDomains: %v", err)
		log.Error(err)
		return nil, err
	}

	return diskProfileList, nil
}
