//  Copyright 2019 Google Inc. All Rights Reserved.
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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/flags"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	ovfimporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_importer"
)

func TestBuildParams(t *testing.T) {
	cliArgs := getAllCliArgs()
	defer test.BackupOsArgs()()
	test.BuildOsArgs(cliArgs)

	params := buildOVFImportParams()

	assert.Equal(t, cliArgs[ovfimporter.InstanceNameFlagKey], params.InstanceNames)
	assert.Equal(t, cliArgs[ovfimporter.MachineImageNameFlagKey], params.MachineImageName)
	assert.Equal(t, cliArgs[ovfimporter.ClientIDFlagKey], params.ClientID)
	assert.Equal(t, cliArgs[ovfimporter.OvfGcsPathFlagKey], params.OvfOvaGcsPath)
	assert.Equal(t, cliArgs["no-guest-environment"], params.NoGuestEnvironment)
	assert.Equal(t, cliArgs["can-ip-forward"], params.CanIPForward)
	assert.Equal(t, cliArgs["deletion-protection"], params.DeletionProtection)
	assert.Equal(t, cliArgs["description"], params.Description)
	assert.Equal(t, cliArgs["labels"], params.Labels)
	assert.Equal(t, cliArgs["machine-type"], params.MachineType)
	assert.Equal(t, cliArgs["network"], params.Network)
	assert.Equal(t, cliArgs["subnet"], params.Subnet)
	assert.Equal(t, cliArgs["network-tier"], params.NetworkTier)
	assert.Equal(t, cliArgs["private-network-ip"], params.PrivateNetworkIP)
	assert.Equal(t, cliArgs["no-external-ip"], params.NoExternalIP)
	assert.Equal(t, cliArgs["no-restart-on-failure"], params.NoRestartOnFailure)
	assert.Equal(t, cliArgs["os"], params.OsID)
	assert.Equal(t, cliArgs["shielded-integrity-monitoring"], params.ShieldedIntegrityMonitoring)
	assert.Equal(t, cliArgs["shielded-secure-boot"], params.ShieldedSecureBoot)
	assert.Equal(t, cliArgs["shielded-vtpm"], params.ShieldedVtpm)
	assert.Equal(t, cliArgs["tags"], params.Tags)
	assert.Equal(t, cliArgs["zone"], params.Zone)
	assert.Equal(t, cliArgs["boot-disk-kms-key"], params.BootDiskKmskey)
	assert.Equal(t, cliArgs["boot-disk-kms-keyring"], params.BootDiskKmsKeyring)
	assert.Equal(t, cliArgs["boot-disk-kms-location"], params.BootDiskKmsLocation)
	assert.Equal(t, cliArgs["boot-disk-kms-project"], params.BootDiskKmsProject)
	assert.Equal(t, cliArgs["timeout"], params.Timeout)
	assert.Equal(t, cliArgs["project"], *params.Project)
	assert.Equal(t, cliArgs["scratch-bucket-gcs-path"], params.ScratchBucketGcsPath)
	assert.Equal(t, cliArgs["oauth"], params.Oauth)
	assert.Equal(t, cliArgs["compute-endpoint-override"], params.Ce)
	assert.Equal(t, cliArgs["disable-gcs-logging"], params.GcsLogsDisabled)
	assert.Equal(t, cliArgs["disable-cloud-logging"], params.CloudLogsDisabled)
	assert.Equal(t, cliArgs["disable-stdout-logging"], params.StdoutLogsDisabled)
	assert.Equal(t, flags.StringArrayFlag{"env,IN,prod,test"}, params.NodeAffinityLabelsFlag)
	assert.Equal(t, cliArgs[ovfimporter.HostnameFlagKey], params.Hostname)
	assert.Equal(t, cliArgs[ovfimporter.MachineImageStorageLocationFlagKey], params.MachineImageStorageLocation)
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{}{
		ovfimporter.InstanceNameFlagKey:     "instance1",
		ovfimporter.MachineImageNameFlagKey: "machineimage1",
		ovfimporter.ClientIDFlagKey:         "aClient",
		ovfimporter.OvfGcsPathFlagKey:       "gs://ovfbucket/ovfpath/vmware.ova",
		"no-guest-environment":              true,
		"can-ip-forward":                    true,
		"deletion-protection":               true,
		"description":                       "aDescription",
		"labels":                            "userkey1=uservalue1,userkey2=uservalue2",
		"machine-type":                      "n1-standard-2",
		"network":                           "aNetwork",
		"subnet":                            "aSubnet",
		"network-tier":                      "PREMIUM",
		"private-network-ip":                "10.0.0.1",
		"no-external-ip":                    true,
		"no-restart-on-failure":             true,
		"os":                                "ubuntu-1404",
		"shielded-integrity-monitoring":     true,
		"shielded-secure-boot":              true,
		"shielded-vtpm":                     true,
		"tags":                              "tag1=val1",
		"zone":                              "us-central1-c",
		"boot-disk-kms-key":                 "aKey",
		"boot-disk-kms-keyring":             "aKeyring",
		"boot-disk-kms-location":            "aKmsLocation",
		"boot-disk-kms-project":             "aKmsProject",
		"timeout":                           "3h",
		"project":                           "aProject",
		"scratch-bucket-gcs-path":           "gs://bucket/folder",
		"oauth":                             "oAuthFilePath",
		"compute-endpoint-override":         "us-east1-c",
		"disable-gcs-logging":               true,
		"disable-cloud-logging":             true,
		"disable-stdout-logging":            true,
		"node-affinity-label":               "env,IN,prod,test",
		ovfimporter.HostnameFlagKey:         "hostname1",
		ovfimporter.MachineImageStorageLocationFlagKey: "us-west2",
	}
}
