package main

import (
	"io"
	"net/http"

	"google.golang.org/api/gensupport"
	"google.golang.org/api/googleapi"
)

func doRequest(client *http.Client, path, resource string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	var body io.Reader
	reqHeaders.Set("Content-Type", "application/json")

	url := path + resource + ":lookupConfigs"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header = reqHeaders
	return gensupport.SendRequest(nil, client, req)
}

func lookupConfigs(client *http.Client, path, resource string) (*LookupConfigsResponse, error) {
	res, err := doRequest(client, path, resource)
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &LookupConfigsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
}

// GooPackageConfig: A list of packages to install, remove, and their
// repos for a given
// package manager type.
type GooPackageConfig struct {
	// PackageInstalls: Packages to install.
	// googet -noconfirm install package1 package2 package3
	PackageInstalls []*Package `json:"packageInstalls,omitempty"`

	// PackageRemovals: Packages to remove.
	// googet -noconfirm remove package1 package2 package3
	PackageRemovals []*Package `json:"packageRemovals,omitempty"`

	// Repositories: Package repositories to configure in the package
	// manager. The instance
	// likely already has some defaults set and duplicates are acceptable
	// but
	// ignored.
	Repositories []*GooRepository `json:"repositories,omitempty"`
}

// GooRepository: Represents a Goo package repository. These will be
// added to a repo file that
// will be managed at C:/ProgramData/GooGet/repos/google_osconfig.repo
type GooRepository struct {
	// Name: The name of the repository.
	Name string `json:"name,omitempty"`

	// Url: The url of the repository.
	Url string `json:"url,omitempty"`
}

// YumPackageConfig: A list of packages to install, remove, and their
// repos for a given
// package manager type.
type YumPackageConfig struct {
	// PackageInstalls: Packages to install.
	// yum -y install package1 package2 package3
	PackageInstalls []*Package `json:"packageInstalls,omitempty"`

	// PackageRemovals: Packages to remove.
	// yum -y remove package1 package2 package3
	PackageRemovals []*Package `json:"packageRemovals,omitempty"`

	// Repositories: Package repositories to configure in the package
	// manager. The instance
	// likely already has some defaults set and duplicates are acceptable
	// but
	// ignored.
	Repositories []*YumRepository `json:"repositories,omitempty"`
}

// YumRepository: Represents a single yum package repository. These will
// be added to a repo
// file that will be managed a /etc/yum.repos.d/google_osconfig.repo
type YumRepository struct {
	// BaseUrl: Required. The location of the repository directory.
	BaseUrl string `json:"baseUrl,omitempty"`

	// DisplayName: Optional. If omitted, the id will be used for the name.
	DisplayName string `json:"displayName,omitempty"`

	// GpgKeys: Optional. URIs of GPG keys.
	GpgKeys []string `json:"gpgKeys,omitempty"`

	// Id: Required. A one word, unique name for this repository. This will
	// be
	// the `repo id` in the yum config file and also the `display_name`
	// if
	// `display_name` is omitted.
	Id string `json:"id,omitempty"`
}

// AptPackageConfig: A list of packages to install, remove, and their
// repos for a given
// package manager type.
type AptPackageConfig struct {
	// PackageInstalls: Packages to install.
	// apt-get update && apt-get -y install package1 package2 package3
	PackageInstalls []*Package `json:"packageInstalls,omitempty"`

	// PackageRemovals: Packages to remove.
	// apt-get -y remove package1 package2 package3
	PackageRemovals []*Package `json:"packageRemovals,omitempty"`

	// Repositories: Package repositories to configure in the package
	// manager. The instance
	// likely already has some defaults set and duplicates are acceptable
	// but
	// ignored.
	Repositories []*AptRepository `json:"repositories,omitempty"`
}

// AptRepository: Represents a single apt package repository. These will
// be added to a repo
// file that will be managed at
// /etc/apt/sources.list.d/google_osconfig.list
type AptRepository struct {
	// ArchiveType: Type of archive files in this repository. Unspecified
	// will default to DEB.
	//
	// Possible values:
	//   "ARCHIVE_TYPE_UNSPECIFIED" - Unspecified.
	//   "DEB" - Deb.
	//   "DEB_SRC" - Deb-src.
	ArchiveType string `json:"archiveType,omitempty"`

	// Components: List of components for this repository. Must contain at
	// least one item.
	Components []string `json:"components,omitempty"`

	// Distribution: Distribution of this repository.
	Distribution string `json:"distribution,omitempty"`

	// KeyUri: Optional. URI of the key file for this repository. The agent
	// will ensure
	// that this key has been downloaded.
	KeyUri string `json:"keyUri,omitempty"`

	// Uri: URI for this repository.
	Uri string `json:"uri,omitempty"`
}

// LookupConfigsRequest: A request message for getting the configs
// assigned to the instance.
type LookupConfigsRequest struct {
	// ConfigTypes: Types of configuration system the instance is using.
	// Only configs relevant
	// to these configuration types will be returned.
	//
	// Possible values:
	//   "CONFIG_TYPE_UNSPECIFIED" - Invalid. Config type must be specified.
	//   "APT" - This instance runs apt.
	//   "YUM" - This instance runs yum.
	//   "GOO" - This instance runs googet.
	//   "ZYPPER" - This instance runs zypper.
	//   "WUA" - This instance runs the Windows Update Agent.
	ConfigTypes []string `json:"configTypes,omitempty"`

	// OsInfo: Optional. OS info about the instance that can be used to
	// filter its
	// configs. If none is provided, the API will return the configs for
	// this
	// instance regardless of its OS.
	OsInfo *OsInfo `json:"osInfo,omitempty"`
}

// OsInfo: Guest information provided to service by agent when
// requesting
// configurations.
type OsInfo struct {
	// OsArchitecture: Architecture of the OS. Optional.
	OsArchitecture string `json:"osArchitecture,omitempty"`

	// OsKernel: OS kernel name. Optional.
	OsKernel string `json:"osKernel,omitempty"`

	// OsLongName: OS long name. Optional.
	OsLongName string `json:"osLongName,omitempty"`

	// OsShortName: OS short name. Optional.
	OsShortName string `json:"osShortName,omitempty"`

	// OsVersion: OS version. Optional.
	OsVersion string `json:"osVersion,omitempty"`
}

// Package: Package is a reference to the actual package to be installed
// or removed.
type Package struct {
	// Name: The name of the package.
	Name string `json:"name,omitempty"`
}

// LookupConfigsResponse: Response with assigned configs for the
// instance.
type LookupConfigsResponse struct {
	// Apt: Configs for apt.
	Apt *AptPackageConfig `json:"apt,omitempty"`

	// Goo: Configs for windows.
	Goo *GooPackageConfig `json:"goo,omitempty"`

	// Yum: Configs for yum.
	Yum *YumPackageConfig `json:"yum,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`
}

// PatchWindow: Specifies a recurring window of time. This window can
// recur daily, weekly, or
// monthly and must last for at least one hour.
type PatchWindow struct {
	// Daily: The patch window should reoccur daily.
	Daily *Daily `json:"daily,omitempty"`

	// Duration: Duration of the patch window. Must be at least 60 minutes
	// and an
	// interval of 30 minutes.
	Duration string `json:"duration,omitempty"`

	// Monthly: The patch window should reoccur monthly.
	Monthly *Monthly `json:"monthly,omitempty"`

	// StartTime: Time to start the maintenance operations.
	// It must include hours and minutes but not seconds or nanos.
	StartTime *TimeOfDay `json:"startTime,omitempty"`

	// Weekly: The patch window should reoccur weekly.
	Weekly *Weekly `json:"weekly,omitempty"`
}

// Daily: The patch window should run daily.
type Daily struct {
}

// Weekly: The patch window should run weekly.
type Weekly struct {
	// Day: Day of the week to run the patch. An unspecified days are
	// invalid.
	//
	// Possible values:
	//   "DAY_OF_WEEK_UNSPECIFIED" - The unspecified day-of-week.
	//   "MONDAY" - The day-of-week of Monday.
	//   "TUESDAY" - The day-of-week of Tuesday.
	//   "WEDNESDAY" - The day-of-week of Wednesday.
	//   "THURSDAY" - The day-of-week of Thursday.
	//   "FRIDAY" - The day-of-week of Friday.
	//   "SATURDAY" - The day-of-week of Saturday.
	//   "SUNDAY" - The day-of-week of Sunday.
	Day string `json:"day,omitempty"`
}

// Monthly: The patch window should reoccur monthly.
type Monthly struct {
	// DayOfMonth: Day of the month to patch. Eg. `12` means to patch every
	// month on the
	// twelfth. Valid values are from 1 to 31 inclusive. Note that if
	// the
	// month doesn't happen to have an occurrence of this day (for
	// example,
	// not all months have a 31st) the patch will not run that month.
	DayOfMonth int64 `json:"dayOfMonth,omitempty"`

	// OccurrenceOfDay: Occurrence of day in the month to patch such as the
	// third tuesday of
	// the month.
	OccurrenceOfDay *OccurrenceOfDay `json:"occurrenceOfDay,omitempty"`
}

// OccurrenceOfDay: Represents a day of the week in a month such as
// the
// third monday of the month.
type OccurrenceOfDay struct {
	// Day: Day of the week to run the patch.
	//
	// Possible values:
	//   "DAY_OF_WEEK_UNSPECIFIED" - The unspecified day-of-week.
	//   "MONDAY" - The day-of-week of Monday.
	//   "TUESDAY" - The day-of-week of Tuesday.
	//   "WEDNESDAY" - The day-of-week of Wednesday.
	//   "THURSDAY" - The day-of-week of Thursday.
	//   "FRIDAY" - The day-of-week of Friday.
	//   "SATURDAY" - The day-of-week of Saturday.
	//   "SUNDAY" - The day-of-week of Sunday.
	Day string `json:"day,omitempty"`

	// Occurrence: Occurrence within the month. Note that if the month
	// doesn't happen to
	// have an occurrence of this day (for example not all months have a
	// 5th
	// Friday) the patch will not run that month.
	Occurrence int64 `json:"occurrence,omitempty"`
}

// TimeOfDay: Represents a time of day. The date and time zone are
// either not significant
// or are specified elsewhere. An API may choose to allow leap seconds.
// Related
// types are google.type.Date and `google.protobuf.Timestamp`.
type TimeOfDay struct {
	// Hours: Hours of day in 24 hour format. Should be from 0 to 23. An API
	// may choose
	// to allow the value "24:00:00" for scenarios like business closing
	// time.
	Hours int64 `json:"hours,omitempty"`

	// Minutes: Minutes of hour of day. Must be from 0 to 59.
	Minutes int64 `json:"minutes,omitempty"`

	// Nanos: Fractions of seconds in nanoseconds. Must be from 0 to
	// 999,999,999.
	Nanos int64 `json:"nanos,omitempty"`
}
