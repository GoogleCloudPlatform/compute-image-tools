//  Copyright 2018 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Package config stores and retrieves configuration settings for the OS Config agent.
package config

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"cloud.google.com/go/compute/metadata"
)

const (
	// InstanceMetadata is the instance metadata URL.
	InstanceMetadata = "http://metadata.google.internal/computeMetadata/v1/instance"
	// ReportURL is where OS configurations are written in guest attributes.
	ReportURL = InstanceMetadata + "/guest-attributes"
)

var (
	resourceOverride = flag.String("resource_override", "", "The URI of the instance this agent is running on in the form of `projects/*/zones/*/instances/*`. If omitted, the name will be determined by querying the metadata service.")
	endpoint         = flag.String("endpoint", "osconfig.googleapis.com:443", "osconfig endpoint override")
	oauth            = flag.String("oauth", "", "path to oauth json file")
	debug            = flag.Bool("debug", false, "set debug log verbosity")

	googetRepoFilePath = flag.String("googet_repo_file", "C:/ProgramData/GooGet/repos/google_osconfig.repo", "googet repo file location")
	zypperRepoFilePath = flag.String("zypper_repo_file", "/etc/zypp/repos.d/google_osconfig.repo", "zypper repo file location")
	yumRepoFilePath    = flag.String("yum_repo_file", "/etc/yum/repos.d/google_osconfig.repo", "yum repo file location")
	aptRepoFilePath    = flag.String("apt_repo_file", "/etc/apt/sources.list.d/google_osconfig.list", "apt repo file location")

	osPackageEnabled bool
	osPatchEnabled   bool

	version string
)

// ReadConfig reads and parses the osconfig config file.
func ReadConfig() error {
	return nil
}

// SvcPollInterval returns the frequency to poll the service.
func SvcPollInterval() time.Duration {
	return 10 * time.Minute
}

// MaxMetadataRetryDelay is the maximum retry delay when getting data from the metadata server.
func MaxMetadataRetryDelay() time.Duration {
	return 30 * time.Second
}

// MaxMetadataRetries is the maximum retry delay when getting data from the metadata server.
func MaxMetadataRetries() int {
	return 30
}

// SerialLogPort is the serial port to log to.
func SerialLogPort() string {
	if runtime.GOOS == "windows" {
		return "COM1"
	}
	return "/dev/ttyS0"
}

// ResourceOverride is the URI of the resource.
func ResourceOverride() string {
	return *resourceOverride
}

// Debug sets the debug log verbosity.
func Debug() bool {
	return *debug
}

// OAuthPath is the local location of the OAuth credentials file.
func OAuthPath() string {
	return *oauth
}

// SvcEndpoint is the OS Config service endpoint.
func SvcEndpoint() string {
	return *endpoint
}

// ZypperRepoFilePath is the location where the zypper repo file will be created.
func ZypperRepoFilePath() string {
	return *zypperRepoFilePath
}

// YumRepoFilePath is the location where the zypper repo file will be created.
func YumRepoFilePath() string {
	return *yumRepoFilePath
}

// AptRepoFilePath is the location where the zypper repo file will be created.
func AptRepoFilePath() string {
	return *aptRepoFilePath
}

// GoogetRepoFilePath is the location where the googet repo file will be created.
func GoogetRepoFilePath() string {
	return *googetRepoFilePath
}

// OSPackageEnabled indicates whether OSPackage should be enabled.
func OSPackageEnabled() bool {
	return osPackageEnabled
}

// OSPatchEnabled indicates whether OSPatch should be enabled.
func OSPatchEnabled() bool {
	return osPatchEnabled
}

// Instance is the URI of the instance the agent is running on.
func Instance() (string, error) {
	if ResourceOverride() != "" {
		return ResourceOverride(), nil
	}

	project, err := metadata.ProjectID()
	if err != nil {
		return "", err
	}

	zone, err := metadata.Zone()
	if err != nil {
		return "", err
	}

	name, err := metadata.InstanceName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, name), nil
}

// Project is the URI of the instance the agent is running on.
func Project() (string, error) {
	proj, err := metadata.ProjectID()
	if err != nil {
		return "", fmt.Errorf("unable to resolve project, are you running in GCE? error: %v", err)
	}
	return proj, nil
}

// Version is the agent version.
func Version() string {
	return version
}

// SetVersion sets the agent version.
func SetVersion(v string) {
	version = v
}
