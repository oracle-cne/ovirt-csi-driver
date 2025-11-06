// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diskprofile

import (
	"fmt"
	"strconv"

	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
)

const PolicyLeastUsed = "leastUsed"

// Select the best domain using the policy
// The current policy picks the storage domain that is least used (most space).
func selectDomainUsingPolicy(domains []*storagedomain.StorageDomain, policy string) (*storagedomain.StorageDomain, error) {
	switch policy {
	case PolicyLeastUsed:
		return selectLeastUsed(domains)
	default:
		return selectLeastUsed(domains)
	}
}

func selectLeastUsed(domains []*storagedomain.StorageDomain) (*storagedomain.StorageDomain, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("no storage domains provided")
	}
	var selected *storagedomain.StorageDomain
	maxsize := int64(-1)
	for _, d := range domains {
		size, err := strconv.ParseInt(d.Available, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse disk size '%v' for domain '%v': %w", d.Available, d, err)
		}
		if size > maxsize {
			selected = d
			maxsize = size
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no valid domain found")
	}
	return selected, nil
}
