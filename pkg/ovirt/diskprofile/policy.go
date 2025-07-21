package diskprofile

import (
	"fmt"
	"github.com/ovirt/csi-driver/pkg/ovirt/rest/storagedomain"
	"strconv"
)

const PolicyLeastUsed = "leastUsed"

// selectDomainUsingPolicy selects the best domain using the policy
func selectDomainUsingPolicy(domains []*storagedomain.StorageDomain, policy string) (*storagedomain.StorageDomain, error) {
	switch policy {
	case PolicyLeastUsed:
		return selectLeastUsed(domains)
	default:
		return selectLeastUsed(domains)
	}
}

func selectLeastUsed(domains []*storagedomain.StorageDomain) (*storagedomain.StorageDomain, error) {
	var selected *storagedomain.StorageDomain
	var maxsize int64
	for i, d := range domains {
		size, err := strconv.ParseInt(d.Available, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse disk size: %w", err)
		}
		if size > maxsize {
			selected = domains[i]
			maxsize = size
		}
	}
	return selected, nil
}
