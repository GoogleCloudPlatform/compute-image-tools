//  Copyright 2021 Google Inc. All Rights Reserved.
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

package ovfimporttestsuite

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	ovfimporter "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

// OvfImportTestProperties hold common OVF import properties shared between
// instance and machine image imports.
type OvfImportTestProperties struct {
	IsWindows                 bool
	ExpectedStartupOutput     string
	FailureMatches            []string
	VerificationStartupScript string
	AllowFallbackToPDStandard bool
	Zone                      string
	SourceURI                 string
	Os                        string
	MachineType               string
	Network                   string
	Subnet                    string
	Project                   string
	Tags                      []string
	ComputeServiceAccount     string
	InstanceServiceAccount    string
	NoInstanceServiceAccount  bool
	InstanceAccessScopes      string
	NoInstanceAccessScopes    bool
	InstanceMetadata          map[string]string
}

// BuildArgsMap builds CLI args map for all OVF import CLIs being tested
// (wrapper, gcloud). Specific args are provided as args to this func to allow
// customization for different tools (instance import, machine image import)
func BuildArgsMap(props *OvfImportTestProperties, testProjectConfig *testconfig.Project,
	gcloudBetaArgs, gcloudArgs, wrapperArgs []string) map[e2e.CLITestType][]string {

	project := GetProject(props, testProjectConfig)
	gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--project=%v", project))
	gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--source-uri=%v", props.SourceURI))
	gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--zone=%v", props.Zone))

	gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--project=%v", project))
	gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--source-uri=%v", props.SourceURI))
	gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--zone=%v", props.Zone))

	wrapperArgs = append(wrapperArgs, fmt.Sprintf("-project=%v", project))
	wrapperArgs = append(wrapperArgs, fmt.Sprintf("-ovf-gcs-path=%v", props.SourceURI))
	wrapperArgs = append(wrapperArgs, fmt.Sprintf("-zone=%v", props.Zone))
	wrapperArgs = append(wrapperArgs, fmt.Sprintf("-build-id=%v", path.RandString(10)))

	if len(props.Tags) > 0 {
		tags := strings.Join(props.Tags, ",")
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--tags=%v", tags))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--tags=%v", tags))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-tags=%v", tags))
	}
	if props.Os != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--os=%v", props.Os))
		gcloudArgs = append(gcloudBetaArgs, fmt.Sprintf("--os=%v", props.Os))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-os=%v", props.Os))
	}
	if props.MachineType != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--machine-type=%v", props.MachineType))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--machine-type=%v", props.MachineType))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-machine-type=%v", props.MachineType))
	}
	if props.Network != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--network=%v", props.Network))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--network=%v", props.Network))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-network=%v", props.Network))
	}
	if props.Subnet != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--subnet=%v", props.Subnet))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--subnet=%v", props.Subnet))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-subnet=%v", props.Subnet))
	}
	if props.ComputeServiceAccount != "" {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--compute-service-account=%v", props.ComputeServiceAccount))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--compute-service-account=%v", props.ComputeServiceAccount))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-compute-service-account=%v", props.ComputeServiceAccount))
	}
	if props.InstanceServiceAccount != "" && !props.NoInstanceServiceAccount {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--service-account=%v", props.InstanceServiceAccount))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--service-account=%v", props.InstanceServiceAccount))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-service-account=%v", props.InstanceServiceAccount))
	}
	if props.NoInstanceServiceAccount {
		gcloudBetaArgs = append(gcloudBetaArgs, "--service-account=")
		gcloudArgs = append(gcloudArgs, "--service-account=")
		wrapperArgs = append(wrapperArgs, "-service-account=")
	}
	if props.InstanceAccessScopes != "" && !props.NoInstanceAccessScopes {
		gcloudBetaArgs = append(gcloudBetaArgs, fmt.Sprintf("--scopes=%v", props.InstanceAccessScopes))
		gcloudArgs = append(gcloudArgs, fmt.Sprintf("--scopes=%v", props.InstanceAccessScopes))
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-scopes=%v", props.InstanceAccessScopes))
	}
	if props.NoInstanceAccessScopes {
		gcloudBetaArgs = append(gcloudBetaArgs, "--scopes=")
		gcloudArgs = append(gcloudArgs, "--scopes=")
		wrapperArgs = append(wrapperArgs, "-scopes=")
	}

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper:                       wrapperArgs,
		e2e.GcloudBetaProdWrapperLatest:   gcloudBetaArgs,
		e2e.GcloudBetaLatestWrapperLatest: gcloudBetaArgs,
		e2e.GcloudGaLatestWrapperRelease:  gcloudArgs,
	}
	return argsMap
}

// GetProject returns project ID by prioritizing the value from OvfImportTestProperties.
// If not set, the value from testconfig.Project is used
func GetProject(props *OvfImportTestProperties, testProjectConfig *testconfig.Project) string {
	if props.Project != "" {
		return props.Project
	}
	return testProjectConfig.TestProjectID
}

// LoadScriptContent loads script content from local file system. If it fails,
// the whole program will exit with an error.
func LoadScriptContent(scriptPath string, logger *log.Logger) string {
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		logger.Printf("Error loading script `%v`: %v", scriptPath, err)
		os.Exit(1)
	}
	return string(scriptContent)
}

// VerifyInstance verifies that imported instance is matching OvfImportTestProperties.
func VerifyInstance(instance *gcp.InstanceBeta, client daisyCompute.Client,
	testCase *junitxml.TestCase, project string, logger *log.Logger, props *OvfImportTestProperties) {

	for attachedDiskIndex, attachedDisk := range instance.Disks {
		//Try to use device name to retrieve a PD. This works well for instance import
		disk, diskErr := client.GetDisk(project, props.Zone, attachedDisk.DeviceName)
		if diskErr != nil {
			// If device name didn't return a disk, try to use instance name as the
			// base for disk name. This should work for GMI import.

			diskNameSuffix := ""
			if attachedDiskIndex > 0 {
				// first/boot disk has the same name as instance. Data disks have suffix
				// based on their position
				diskNameSuffix = fmt.Sprintf("-%v", attachedDiskIndex)
			}
			disk, diskErr = client.GetDisk(project, props.Zone, instance.Name+diskNameSuffix)
			if diskErr != nil {
				e2e.Failure(testCase, logger,
					fmt.Sprintf("Can't retrieve disk `%v` for instance `%v`: %v",
						attachedDisk.DeviceName, instance.Name, diskErr))
				return
			}
		}

		// Verify disk type (SSD vs Standard)
		expectedDiskType := "pd-ssd"
		if !strings.Contains(disk.Type, expectedDiskType) && !props.AllowFallbackToPDStandard {
			e2e.Failure(testCase, logger, fmt.Sprintf(
				"Disk should be of `%v` type, but is `%v`", expectedDiskType, disk.Type))
			return
		}

		if attachedDisk.Boot {
			// The boot disk for a Windows instance must have the WINDOWS GuestOSFeature,
			// while the boot disk for other operating systems shouldn't have it.
			hasWindowsFeature := false
			for _, feature := range attachedDisk.GuestOsFeatures {
				if "WINDOWS" == feature.Type {
					hasWindowsFeature = true
					break
				}
			}
			if props.IsWindows && !hasWindowsFeature {
				testCase.WriteFailure(
					"Windows boot disk missing WINDOWS GuestOsFeature. Features found=%v",
					attachedDisk.GuestOsFeatures)
			} else if !props.IsWindows && hasWindowsFeature {
				testCase.WriteFailure(
					"Non-Windows boot disk includes WINDOWS GuestOsFeature. Features found=%v",
					attachedDisk.GuestOsFeatures)
			}
		}
	}

	if props.MachineType != "" && !strings.HasSuffix(instance.MachineType, props.MachineType) {
		testCase.WriteFailure(
			"Instance machine type `%v` doesn't match the expected machine type `%v`",
			instance.MachineType, props.MachineType)
		return
	}

	if !strings.HasSuffix(instance.Zone, props.Zone) {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance zone `%v` doesn't match requested zone `%v`",
			instance.Zone, props.Zone))
		return
	}

	//Verify networks
	if (props.Network != "" || props.Subnet != "") && len(instance.NetworkInterfaces) == 0 {
		e2e.Failure(testCase, logger,
			fmt.Sprintf("Instance does not have network interface when `%v` network flag is provided", props.Network))
		return
	}
	for _, networkInterface := range instance.NetworkInterfaces {
		if props.Network != "" && !strings.Contains(networkInterface.Network, props.Network) {
			e2e.Failure(testCase, logger,
				fmt.Sprintf("Instance network (%v) does not match network flag value: %v", networkInterface.Network, props.Network))
			return
		}
		if props.Subnet != "" && !strings.Contains(networkInterface.Subnetwork, props.Subnet) {
			e2e.Failure(testCase, logger,
				fmt.Sprintf("Instance subnetwork (%v) does not match subnetwork flag value: %v", networkInterface.Subnetwork, props.Subnet))
			return
		}
	}

	// Verify that the instance's tags match the user's request (regardless of order).
	expectedTags := map[string]struct{}{}
	for _, tag := range props.Tags {
		expectedTags[tag] = struct{}{}
	}
	actualTags := map[string]struct{}{}
	for _, tag := range instance.Tags.Items {
		actualTags[tag] = struct{}{}
	}
	if !reflect.DeepEqual(expectedTags, actualTags) {
		e2e.Failure(testCase, logger,
			fmt.Sprintf("Instance's tags don't match import request. Expected=`%v`, Actual=`%v`",
				props.Tags, instance.Tags.Items))
		return
	}

	// Verify instance service account
	if props.NoInstanceServiceAccount {
		if len(instance.ServiceAccounts) > 0 {
			e2e.Failure(testCase, logger,
				fmt.Sprintf("Instance service accounts should be empty but is `%v`", instance.ServiceAccounts[0].Email))
			return
		}
	} else {
		if len(instance.ServiceAccounts) == 0 {
			e2e.Failure(testCase, logger, fmt.Sprintf("No service account on the instance when one should be set"))
			return
		}
		if props.InstanceServiceAccount != "" {
			serviceAccountMatch := false
			var instanceServiceAccountEmails []string
			for _, instanceServiceAccount := range instance.ServiceAccounts {
				instanceServiceAccountEmails = append(instanceServiceAccountEmails, instanceServiceAccount.Email)
				if instanceServiceAccount.Email == props.InstanceServiceAccount {
					serviceAccountMatch = true
				}
			}
			if !serviceAccountMatch {
				e2e.Failure(testCase, logger, fmt.Sprintf("Instance service accounts (`%v`) don't contain custom service account `%v`",
					strings.Join(instanceServiceAccountEmails, ","), props.InstanceServiceAccount))
				return
			}
		}
	}

	// Verify instance access scopes
	expectedScopes := []string{}
	if props.InstanceAccessScopes != "" {
		expectedScopes = strings.Split(props.InstanceAccessScopes, ",")
	} else if !props.NoInstanceAccessScopes {
		expectedScopes = ovfimporter.DefaultInstanceAccessScopes
	}
	sort.Strings(expectedScopes)

	for _, instanceServiceAccount := range instance.ServiceAccounts {
		if props.NoInstanceAccessScopes && len(instanceServiceAccount.Scopes) > 0 {
			e2e.Failure(testCase, logger, fmt.Sprintf(
				"Instance access scopes for service account `%v` should be empty but are: %v",
				instanceServiceAccount.Email, strings.Join(instanceServiceAccount.Scopes, ",")))
			return
		}
		sort.Strings(instanceServiceAccount.Scopes)
		if len(instanceServiceAccount.Scopes) == 0 && len(expectedScopes) == 0 {
			continue
		}
		if !reflect.DeepEqual(expectedScopes, instanceServiceAccount.Scopes) {
			e2e.Failure(testCase, logger, fmt.Sprintf(
				"Instance access scopes (%v) for service account `%v` do not match expected scopes: `%v`. NoInstanceAccessScopes %v, len %v",
				strings.Join(instanceServiceAccount.Scopes, ","), instanceServiceAccount.Email, strings.Join(expectedScopes, ","), props.NoInstanceAccessScopes, len(instanceServiceAccount.Scopes)))
			return
		}
	}

	logger.Printf("[%v] Stopping instance before restarting with test startup script", instance.Name)
	err := client.StopInstance(project, props.Zone, instance.Name)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return
	}

	if props.VerificationStartupScript == "" {
		logger.Printf("[%v] Will not set test startup script to instance metadata as it's not defined", instance.Name)
		return
	}

	err = instance.StartWithScriptCode(props.VerificationStartupScript, props.InstanceMetadata)
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: %v", instance.Name, err)
		return
	}
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", instance.Name,
		props.ExpectedStartupOutput)
	if err := instance.WaitForSerialOutput(
		props.ExpectedStartupOutput, props.FailureMatches, 1, 5*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
	}
}
