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
	resourceOverride   = flag.String("resource_override", "", "The URI of the instance this agent is running on in the form of `projects/*/zones/*/instances/*`. If omitted, the name will be determined by querying the metadata service.")
	endpoint           = flag.String("endpoint", "osconfig.googleapis.com:443", "osconfig endpoint override")
	oauth              = flag.String("oauth", "", "path to oauth json file")
	googetRepoFilePath = flag.String("googetRepoFile", "C:/ProgramData/GooGet/repos/google_osconfig.repo", "googet repo file location")
	zypperRepoFilePath = flag.String("zypperRepoFile", "/etc/zypp/repos.d/google_osconfig.repo", "zypper repo file location")
	debug              = flag.Bool("debug", false, "set debug log verbosity")
)

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

// GoogetRepoFilePath is the location where the googet repo file will be created.
func GoogetRepoFilePath() string {
	return *googetRepoFilePath
}

// Instance is the URI of the instance the agent is running on.
func Instance() (string, error) {
	if ResourceOverride() != "" {
		return ResourceOverride(), nil
	}

	zone, err := metadata.Zone()
	if err != nil {
		return "", err
	}

	name, err := metadata.InstanceName()
	if err != nil {
		return "", err
	}
	// Zone returns projects/<project>/zones/<zone>
	return fmt.Sprintf("%s/instances/%s", zone, name), nil
}

// Project is the URI of the instance the agent is running on.
func Project() (string, error) {
	return metadata.ProjectID()
}
