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

// Package ovftestsuite contains e2e tests for OVF import
package ovftestsuite

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_import_params"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_importer"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeUtils "github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"
)

const (
	testSuiteName = "OVFImportTests"
)

var (
	dump = &pretty.Config{IncludeUnexported: true}
)

var vf = func(
		inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

type ovfImportTestSetup struct {
	importParams        *ovfimportparams.OVFImportParams
	name                string
	description         string
	startup             *api.MetadataItems
	assertTimeout       time.Duration
	vf                  func(*compute.Instance, string, int64, time.Duration, time.Duration) error
	expectedMachineType string
}

// TestSuite is OVF import test suite.
func TestSuite(
		ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
		logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
		testProjectConfig *testconfig.Project) {

	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)
	suffix := utils.RandString(5)
	logger.Printf("Running TestSuite %q", testSuite.Name)

	startupScriptUbuntu3disks, err := ioutil.ReadFile(
		"gce_ovf_import_tests/scripts/ovf_import_test_ubuntu_3_disks.sh")
	if err != nil {
		os.Exit(1)
	}
	startupScriptLinuxSingleDisk, err := ioutil.ReadFile(
		"daisy_integration_tests/scripts/post_translate_test.sh")
	if err != nil {
		os.Exit(1)
	}
	startupScriptWindowsSingleDisk, err := ioutil.ReadFile(
		"daisy_integration_tests/scripts/post_translate_test.ps1")
	if err != nil {
		os.Exit(1)
	}
	startupScriptWindowsTwoDisks, err := ioutil.ReadFile(
		"gce_ovf_import_tests/scripts/ovf_import_test_windows_2k12_r2_two_disks.ps1")
	if err != nil {
		os.Exit(1)
	}

	testSetup := []*ovfImportTestSetup{
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-ubuntu-3mounteddisks-%v", suffix),
				OvfOvaGcsPath: "gs://compute-image-tools-test-resources/ova/Ubuntu_for_Horizon_three_disks_mounted.ova",
				OsID:          "ubuntu-1404",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-1",
			},
			name:        fmt.Sprintf("ovf-import-test-ubuntu-3-disks-%s", suffix),
			description: "Ubuntu 3 disks mounted",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", string(startupScriptUbuntu3disks)),
			assertTimeout: 7200 * time.Second,
			expectedMachineType: "n1-standard-1",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-centos-6-%v", suffix),
				OvfOvaGcsPath: "gs://compute-image-tools-test-resources/ova/centos-6.8",
				OsID:          "centos-6",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-4",
			},
			name:        fmt.Sprintf("ovf-import-test-centos-6-%s", suffix),
			description: "Centos 6.8",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", string(startupScriptLinuxSingleDisk)),
			assertTimeout: 7200 * time.Second,
			expectedMachineType: "n1-standard-4",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-w2k12-r2-%v", suffix),
				OvfOvaGcsPath: "gs://compute-image-tools-test-resources/ova/w2k12-r2",
				OsID:          "windows-2012r2",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-8",
			},
			name:        fmt.Sprintf("ovf-import-test-w2k12-r2-%s", suffix),
			description: "Windows 2012 R2 two disks",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", string(startupScriptWindowsTwoDisks)),
			assertTimeout: 7200 * time.Second,
			expectedMachineType: "n1-standard-8",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-w2k16-%v", suffix),
				OvfOvaGcsPath: "gs://compute-image-tools-test-resources/ova/w2k16/w2k16.ovf",
				OsID:          "windows-2016",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
			},
			name:        fmt.Sprintf("ovf-import-test-w2k16-%s", suffix),
			description: "Windows 2016",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", string(startupScriptWindowsSingleDisk)),
			assertTimeout: 7200 * time.Second,
			expectedMachineType: "n1-standard-2",
		},
	}

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for _, setup := range testSetup {
		wg.Add(1)
		go ovfImportTestCase(ctx, setup, tests, &wg, logger, testCaseRegex, testProjectConfig)
	}

	go func() {
		wg.Wait()
		close(tests)
	}()

	for ret := range tests {
		testSuite.TestCase = append(testSuite.TestCase, ret)
	}

	logger.Printf("Finished TestSuite %q", testSuite.Name)
}

func ovfImportTestCase(
		ctx context.Context, testSetup *ovfImportTestSetup, tests chan *junitxml.TestCase,
		wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp,
		testProjectConfig *testconfig.Project) {

	defer wg.Done()

	ovfImportTestCase := junitxml.NewTestCase(
		testSuiteName, fmt.Sprintf("[OVFImport] %v", testSetup.description))

	for tc, f := range map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *ovfImportTestSetup, *log.Logger, *testconfig.Project){
		ovfImportTestCase: runOvfImportTest,
	} {
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
			f(ctx, tc, testSetup, logger, testProjectConfig)
			tc.Finish(tests)
			logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
		}
	}
}

func runOvfImportTest(
		ctx context.Context, testCase *junitxml.TestCase, testSetup *ovfImportTestSetup,
		logger *log.Logger, testProjectConfig *testconfig.Project) {

	logger.Printf("Creating OVF importer")
	ovfImporter, err := ovfimporter.NewOVFImporter(testSetup.importParams)

	if err != nil {
		testCase.WriteFailure("error creating OVF importer: %v", err)
		return
	}

	logger.Printf("Starting OVF import")
	err = ovfImporter.Import()
	if err != nil {
		testCase.WriteFailure("error while performing OVF import: %v", err)
		ovfImporter.CleanUp()
		return
	}

	logger.Printf("OVF import finished")
	ovfImporter.CleanUp()

	// assertion
	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("Error creating client: %v", err)
		return
	}

	instanceName := testSetup.importParams.InstanceNames

	instance, err := client.GetInstance(
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
	if !strings.HasSuffix(instance.MachineType, testSetup.expectedMachineType) {
		testCase.WriteFailure(
			"Instance machine type `%v` doesn't match the expected machine type `%v`",
			instance.MachineType, testSetup.expectedMachineType)
		return
	}

	if !strings.HasSuffix(instance.Zone, testSetup.importParams.Zone) {
		testCase.WriteFailure("Instance zone `%v` doesn't match requested zone `%v`",
			instance.Zone, testSetup.importParams.Zone)
		return
	}

	logger.Printf("Stopping instance before restarting with test startup script")
	err = client.StopInstance(
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return
	}

	logger.Printf("Setting instance metadata with test startup script")
	err = client.SetInstanceMetadata(testProjectConfig.TestProjectID, testProjectConfig.TestZone,
		instanceName, &api.Metadata{Items: []*api.MetadataItems{testSetup.startup},
			Fingerprint: instance.Metadata.Fingerprint})

	if err != nil {
		testCase.WriteFailure("Couldn't set instance metadata to verify OVF import: %v", err)
		return
	}

	logger.Printf("Starting instance with test startup script")
	err = client.StartInstance(
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
	if err != nil {
		testCase.WriteFailure("Couldn't start instance to verify OVF import: %v", err)
		return
	}

	inst := computeUtils.Instance{Instance: instance, Client: client,
		Project: testProjectConfig.TestProjectID, Zone: testProjectConfig.TestZone}

	if err := inst.WaitForSerialOutput(
		"All tests passed", 1, 5*time.Second, 7*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
		return
	}

	logger.Printf("Deleting instance `%v`", instanceName)
	inst.Cleanup()
}
