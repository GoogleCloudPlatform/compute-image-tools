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
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
)

const (
	// InstanceMetadata is the instance metadata URL.
	InstanceMetadata = "http://metadata.google.internal/computeMetadata/v1/instance"
	// ReportURL is the guest attributes endpoint.
	ReportURL = InstanceMetadata + "/guest-attributes"

	googetRepoFilePath = "C:/ProgramData/GooGet/repos/google_osconfig.repo"
	zypperRepoFilePath = "/etc/zypp/repos.d/google_osconfig.repo"
	yumRepoFilePath    = "/etc/yum/repos.d/google_osconfig.repo"
	aptRepoFilePath    = "/etc/apt/sources.list.d/google_osconfig.list"

	prodEndpoint = "osconfig.googleapis.com:443"

	osInventoryEnabledDefault = false
	osPackageEnabledDefault   = false
	osPatchEnabledDefault     = false
	debugEnabledDefault       = false

	osConfigPollIntervalDefault = 10
)

var (
	resourceOverride = flag.String("resource_override", "", "The URI of the instance this agent is running on in the form of `projects/*/zones/*/instances/*`. If omitted, the name will be determined by querying the metadata service.")
	endpoint         = flag.String("endpoint", prodEndpoint, "osconfig endpoint override")
	oauth            = flag.String("oauth", "", "path to oauth json file")
	debug            = flag.Bool("debug", false, "set debug log verbosity")

	agentConfig   = &config{}
	agentConfigMx sync.RWMutex
	version       string
)

type config struct {
	osInventoryEnabled, osPackageEnabled, osPatchEnabled, debugEnabled                    bool
	svcEndpoint, googetRepoFilePath, zypperRepoFilePath, yumRepoFilePath, aptRepoFilePath string
	numericProjectID, osConfigPollInterval                                                int
	projectID, instanceZone, instanceName, instanceID                                     string
}

func (c *config) parsePreRelease(features string) {
	for _, f := range strings.Split(features, ",") {
		f = strings.TrimSpace(f)
		switch f {
		case "ospatch":
			c.osPatchEnabled = true
		case "ospackage":
			c.osPackageEnabled = true
		}
	}
}

func getAgentConfig() config {
	agentConfigMx.RLock()
	defer agentConfigMx.RUnlock()
	return *agentConfig
}

func parseBool(s string) bool {
	enabled, err := strconv.ParseBool(s)
	if err != nil {
		// Bad entry returns as not enabled.
		return false
	}
	return enabled
}

type metadataJSON struct {
	Instance instanceJSON
	Project  projectJSON
}

type instanceJSON struct {
	Attributes attributesJSON
	Zone       string
	Name       string
	ID         *json.Number
}

type projectJSON struct {
	Attributes       attributesJSON
	ProjectID        string
	NumericProjectID int
}

type attributesJSON struct {
	InventoryEnabled   string       `json:"os-inventory-enabled"`
	PreReleaseFeatures string       `json:"os-config-enabled-prerelease-features"`
	DebugEnabled       string       `json:"os-config-debug-enabled"`
	OSConfigEndpoint   string       `json:"os-config-endpoint"`
	PollInterval       *json.Number `json:"os-config-poll-interval"`
}

func createConfigFromMetadata(md metadataJSON) *config {
	old := getAgentConfig()
	c := &config{
		osInventoryEnabled:   osInventoryEnabledDefault,
		osPackageEnabled:     osPackageEnabledDefault,
		osPatchEnabled:       osPatchEnabledDefault,
		debugEnabled:         debugEnabledDefault,
		svcEndpoint:          prodEndpoint,
		osConfigPollInterval: osConfigPollIntervalDefault,

		googetRepoFilePath: googetRepoFilePath,
		zypperRepoFilePath: zypperRepoFilePath,
		yumRepoFilePath:    yumRepoFilePath,
		aptRepoFilePath:    aptRepoFilePath,

		projectID:        old.projectID,
		numericProjectID: old.numericProjectID,
		instanceZone:     old.instanceZone,
		instanceName:     old.instanceName,
		instanceID:       old.instanceID,
	}

	if md.Project.ProjectID != "" {
		c.projectID = md.Project.ProjectID
	}
	if md.Project.NumericProjectID != 0 {
		c.numericProjectID = md.Project.NumericProjectID
	}
	if md.Instance.Zone != "" {
		c.instanceZone = md.Instance.Zone
	}
	if md.Instance.Name != "" {
		c.instanceName = md.Instance.Name
	}
	if md.Instance.ID != nil {
		c.instanceID = md.Instance.ID.String()
	}

	// Check project first then instance as instance metadata overrides project.
	if md.Project.Attributes.InventoryEnabled != "" {
		c.osInventoryEnabled = parseBool(md.Project.Attributes.InventoryEnabled)
	}
	c.parsePreRelease(md.Project.Attributes.PreReleaseFeatures)

	if md.Instance.Attributes.InventoryEnabled != "" {
		c.osInventoryEnabled = parseBool(md.Instance.Attributes.InventoryEnabled)
	}
	c.parsePreRelease(md.Instance.Attributes.PreReleaseFeatures)

	if md.Instance.Attributes.PollInterval != nil {
		if val, err := md.Instance.Attributes.PollInterval.Int64(); err == nil {
			c.osConfigPollInterval = int(val)
		}
	}

	// Flags take precedence over metadata.
	if *debug {
		c.debugEnabled = true
	} else if md.Instance.Attributes.DebugEnabled != "" {
		c.debugEnabled = parseBool(md.Instance.Attributes.DebugEnabled)
	} else if md.Project.Attributes.DebugEnabled != "" {
		c.debugEnabled = parseBool(md.Project.Attributes.DebugEnabled)
	}

	if *endpoint != prodEndpoint {
		c.svcEndpoint = *endpoint
	} else if md.Instance.Attributes.OSConfigEndpoint != "" {
		c.svcEndpoint = md.Instance.Attributes.OSConfigEndpoint
	} else if md.Project.Attributes.OSConfigEndpoint != "" {
		c.svcEndpoint = md.Project.Attributes.OSConfigEndpoint
	}

	return c
}

func formatError(err error) error {
	if urlErr, ok := err.(*url.Error); ok {
		if _, ok := urlErr.Err.(*net.DNSError); ok {
			return fmt.Errorf("DNS error when requesting metadata, check DNS settings and ensure metadata.internal.google is setup in your hosts file")
		}
		if _, ok := urlErr.Err.(*net.OpError); ok {
			return fmt.Errorf("network error when requesting metadata, make sure your instance has an active network and can reach the metadata server")
		}
	}
	return err
}

// SetConfig sets the agent config.
func SetConfig() error {
	var md string
	var webError error
	webErrorCount := 0
	for {
		md, webError = metadata.Get("?recursive=true&alt=json")
		if webError == nil {
			break
		}
		// Try up to 3 times to wait for slow network initialization, after
		// that resort to using defaults and returning the error.
		if webErrorCount == 2 {
			webError = formatError(webError)
			break
		}
		webErrorCount++
		time.Sleep(5 * time.Second)
	}

	var metadata metadataJSON
	if err := json.Unmarshal([]byte(md), &metadata); err != nil {
		return err
	}

	new := createConfigFromMetadata(metadata)
	agentConfigMx.Lock()
	agentConfig = new
	agentConfigMx.Unlock()

	return webError
}

// SvcPollInterval returns the frequency to poll the service.
func SvcPollInterval() time.Duration {
	return time.Duration(getAgentConfig().osConfigPollInterval) * time.Minute
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
	// Don't write directly to the serial port on Linux as syslog already writes there.
	return ""
}

// ResourceOverride is the URI of the resource.
func ResourceOverride() string {
	return *resourceOverride
}

// Debug sets the debug log verbosity.
func Debug() bool {
	return (*debug || getAgentConfig().debugEnabled)
}

// OAuthPath is the local location of the OAuth credentials file.
func OAuthPath() string {
	return *oauth
}

// SvcEndpoint is the OS Config service endpoint.
func SvcEndpoint() string {
	return getAgentConfig().svcEndpoint
}

// ZypperRepoFilePath is the location where the zypper repo file will be created.
func ZypperRepoFilePath() string {
	return getAgentConfig().zypperRepoFilePath
}

// YumRepoFilePath is the location where the zypper repo file will be created.
func YumRepoFilePath() string {
	return getAgentConfig().yumRepoFilePath
}

// AptRepoFilePath is the location where the zypper repo file will be created.
func AptRepoFilePath() string {
	return getAgentConfig().aptRepoFilePath
}

// GooGetRepoFilePath is the location where the googet repo file will be created.
func GooGetRepoFilePath() string {
	return getAgentConfig().googetRepoFilePath
}

// OSInventoryEnabled indicates whether OSInventory should be enabled.
func OSInventoryEnabled() bool {
	return getAgentConfig().osInventoryEnabled
}

// OSPackageEnabled indicates whether OSPackage should be enabled.
func OSPackageEnabled() bool {
	return getAgentConfig().osPackageEnabled
}

// OSPatchEnabled indicates whether OSPatch should be enabled.
func OSPatchEnabled() bool {
	return getAgentConfig().osPatchEnabled
}

// Instance is the URI of the instance the agent is running on.
func Instance() string {
	if ResourceOverride() != "" {
		return ResourceOverride()
	}

	// Zone contains 'projects/project-id/zones' as a prefix.
	return fmt.Sprintf("%s/instances/%s", Zone(), Name())
}

// NumericProjectID is the numeric project ID of the instance.
func NumericProjectID() int {
	return getAgentConfig().numericProjectID
}

// ProjectID is the project ID of the instance.
func ProjectID() string {
	return getAgentConfig().projectID
}

// Zone is the zone the instance is running in.
func Zone() string {
	return getAgentConfig().instanceZone
}

// Name is the instance name.
func Name() string {
	return getAgentConfig().instanceName
}

// ID is the instance id.
func ID() string {
	return getAgentConfig().instanceID
}

// Version is the agent version.
func Version() string {
	return version
}

// SetVersion sets the agent version.
func SetVersion(v string) {
	version = v
}
