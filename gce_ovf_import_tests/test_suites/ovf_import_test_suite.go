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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_import_params"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_importer"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests/compute"
	computeUtils "github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
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
	importParams          *ovfimportparams.OVFImportParams
	name                  string
	description           string
	startup               *api.MetadataItems
	assertTimeout         time.Duration
	vf                    func(*compute.Instance, string, int64, time.Duration, time.Duration) error
	expectedMachineType   string
	expectedStartupOutput string
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
	suffix := path.RandString(5)
	ovaBucket := "compute-image-tools-test-resources"
	logger.Printf("Running TestSuite %q", testSuite.Name)

	startupScriptUbuntu3disks := loadScriptContent(
		"gce_ovf_import_tests/scripts/ovf_import_test_ubuntu_3_disks.sh", logger)
	startupScriptLinuxSingleDisk := loadScriptContent(
		"daisy_integration_tests/scripts/post_translate_test.sh", logger)
	startupScriptWindowsSingleDisk := loadScriptContent(
		"daisy_integration_tests/scripts/post_translate_test.ps1", logger)
	startupScriptWindowsTwoDisks := loadScriptContent(
		"gce_ovf_import_tests/scripts/ovf_import_test_windows_two_disks.ps1", logger)

	testSetup := []*ovfImportTestSetup{
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-ubuntu-3mounteddisks-%v", suffix),
				OvfOvaGcsPath: fmt.Sprintf("gs://%v/ova/Ubuntu_for_Horizon_three_disks_mounted.ova", ovaBucket),
				OsID:          "ubuntu-1404",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-1",
			},
			name:        fmt.Sprintf("ovf-import-test-ubuntu-3-disks-%s", suffix),
			description: "Ubuntu 3 disks mounted",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", startupScriptUbuntu3disks),
			assertTimeout:         7200 * time.Second,
			expectedMachineType:   "n1-standard-1",
			expectedStartupOutput: "All tests passed!",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-centos-6-%v", suffix),
				OvfOvaGcsPath: fmt.Sprintf("gs://%v/ova/centos-6.8", ovaBucket),
				OsID:          "centos-6",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-4",
			},
			name:        fmt.Sprintf("ovf-import-test-centos-6-%s", suffix),
			description: "Centos 6.8",
			startup: computeUtils.BuildInstanceMetadataItem(
				"startup-script", startupScriptLinuxSingleDisk),
			assertTimeout:         7200 * time.Second,
			expectedMachineType:   "n1-standard-4",
			expectedStartupOutput: "All tests passed!",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-w2k12-r2-%v", suffix),
				OvfOvaGcsPath: fmt.Sprintf("gs://%v/ova/w2k12-r2", ovaBucket),
				OsID:          "windows-2012r2",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-8",
			},
			name:        fmt.Sprintf("ovf-import-test-w2k12-r2-%s", suffix),
			description: "Windows 2012 R2 two disks",
			startup: computeUtils.BuildInstanceMetadataItem(
				"windows-startup-script-ps1", startupScriptWindowsTwoDisks),
			assertTimeout:         7200 * time.Second,
			expectedMachineType:   "n1-standard-8",
			expectedStartupOutput: "All Tests Passed",
		},
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-w2k16-%v", suffix),
				OvfOvaGcsPath: fmt.Sprintf("gs://%v/ova/w2k16/w2k16.ovf", ovaBucket),
				OsID:          "windows-2016",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
			},
			name:        fmt.Sprintf("ovf-import-test-w2k16-%s", suffix),
			description: "Windows 2016",
			startup: computeUtils.BuildInstanceMetadataItem(
				"windows-startup-script-ps1", startupScriptWindowsSingleDisk),
			assertTimeout:         7200 * time.Second,
			expectedMachineType:   "n2-standard-2",
			expectedStartupOutput: "All Tests Passed",
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

	logger.Printf("[%v] Creating OVF importer", testSetup.name)
	ovfImporter, err := ovfimporter.NewOVFImporter(testSetup.importParams)

	if err != nil {
		logger.Printf("[%v] error creating OVF importer: %v", testSetup.name, err)
		testCase.WriteFailure("error creating OVF importer: %v", err)
		return
	}

	logger.Printf("[%v] Starting OVF import", testSetup.name)
	err = ovfImporter.Import()
	if err != nil {
		logger.Printf("[%v] error while performing OVF import: %v", testSetup.name, err)
		testCase.WriteFailure("error while performing OVF import: %v", err)
		ovfImporter.CleanUp()
		return
	}

	logger.Printf("[%v] OVF import finished", testSetup.name)
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
	if err != nil {
		testCase.WriteFailure("Error retrieving instance `%v` in `%v` zone: %v", instanceName,
			testProjectConfig.TestZone, err)
		return
	}

	instanceWrapper := computeUtils.Instance{Instance: instance, Client: client,
		Project: testProjectConfig.TestProjectID, Zone: testProjectConfig.TestZone}

	defer func() {
		logger.Printf("[%v] Deleting instance `%v`", testSetup.name, instanceName)
		instanceWrapper.Cleanup()
	}()

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

	logger.Printf("[%v] Stopping instance before restarting with test startup script", testSetup.name)
	err = client.StopInstance(
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return
	}

	logger.Printf("[%v] Setting instance metadata with test startup script", testSetup.name)
	err = client.SetInstanceMetadata(testProjectConfig.TestProjectID, testProjectConfig.TestZone,
		instanceName, &api.Metadata{Items: []*api.MetadataItems{testSetup.startup},
			Fingerprint: instance.Metadata.Fingerprint})

	if err != nil {
		testCase.WriteFailure("Couldn't set instance metadata to verify OVF import: %v", err)
		return
	}

	logger.Printf("[%v] Starting instance with test startup script", testSetup.name)
	err = client.StartInstance(
		testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
	if err != nil {
		testCase.WriteFailure("Couldn't start instance to verify OVF import: %v", err)
		return
	}

	logger.Printf("[%v] Waiting for `%v` in instance serial console.", testSetup.name,
		testSetup.expectedStartupOutput)
	if err := instanceWrapper.WaitForSerialOutput(
		testSetup.expectedStartupOutput, 1, 5*time.Second, 7*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
		return
	}
}

func loadScriptContent(scriptPath string, logger *log.Logger) string {
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		logger.Printf("Error loading script `%v`: %v", scriptPath, err)
		os.Exit(1)
	}
	return string(scriptContent)
}
