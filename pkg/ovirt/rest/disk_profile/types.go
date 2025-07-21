// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk_profile

const (
	StatusOK = "ok"

	FormatCow = "cow"

	BackupNone = "none"
)

type DiskProfileList struct {
	DiskProfiles []DiskProfile `json:"disk_profile"`
}

type DiskProfile struct {
	StorageDomain struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"storage_domain"`
	Name string `json:"name"`
	Link []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"link"`
	Href string `json:"href"`
	Id   string `json:"id"`
}
