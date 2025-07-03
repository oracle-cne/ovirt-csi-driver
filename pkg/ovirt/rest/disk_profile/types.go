// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk_profile

const (
	StatusOK = "ok"

	FormatCow = "cow"

	BackupNone = "none"
)

type DiskProfileList struct {
	DiskProfiles []DiskProfile `json:"disk_profiles"`
}

type DiskProfile struct {
	ActualSize      string `json:"actual_size"`
	Alias           string `json:"alias"`
	Backup          string `json:"backup"`
	ContentType     string `json:"content_type"`
	Format          string `json:"format"`
	ImageId         string `json:"image_id"`
	PropagateErrors string `json:"propagate_errors"`
	ProvisionedSize string `json:"provisioned_size"`
	Shareable       string `json:"shareable"`
	Sparse          string `json:"sparse"`
	Status          string `json:"status"`
	StorageType     string `json:"storage_type"`
	TotalSize       string `json:"total_size"`
	WipeAfterDelete string `json:"wipe_after_delete"`
	DiskProfile     struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"disk_profile"`
	Quota struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"quota"`
	StorageDomains struct {
		StorageDomain []struct {
			Href string `json:"href"`
			Id   string `json:"id"`
		} `json:"storage_domain"`
	} `json:"storage_domains"`
	Actions struct {
		Link []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"link"`
	} `json:"actions"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Link        []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"link"`
	Href string `json:"href"`
	Id   string `json:"id"`
}
