// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diskprofile

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	log "k8s.io/klog"
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

	maxsize := int64(-1)
	var sameChoices []*storagedomain.StorageDomain

	for _, d := range domains {
		size, err := strconv.ParseInt(d.Available, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse disk size '%v' for domain '%v': %w", d.Available, d, err)
		}
		log.Infof("The available size of storage domain '%s' is '%s', used: '%s'", d.Name, d.Available, d.Used)
		if size > maxsize {
			maxsize = size
			sameChoices = []*storagedomain.StorageDomain{d}
		} else if size == maxsize {
			sameChoices = append(sameChoices, d)
		}
	}

	if len(sameChoices) == 0 {
		return nil, fmt.Errorf("no valid domain found")
	}
	if len(sameChoices) == 1 {
		return sameChoices[0], nil
	}

	// Randomly pick which storage domain to use when more than one choice exists
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	return sameChoices[rnd.Intn(len(sameChoices))], nil
}
