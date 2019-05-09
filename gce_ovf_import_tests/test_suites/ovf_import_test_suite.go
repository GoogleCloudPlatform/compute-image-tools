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

// Package main contains e2e tests for OVF import
package ovf_test_suites

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

var vf = func(inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

type ovfImportTestSetup struct {
	importParams  *ovfimportparams.OVFImportParams
	name          string
	startup       *api.MetadataItems
	assertTimeout time.Duration
	vf            func(*compute.Instance, string, int64, time.Duration, time.Duration) error
}

// TestSuite is OVF import test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)
	suffix := utils.RandString(5)
	logger.Printf("Running TestSuite %q", testSuite.Name)

	//startupScript := "echo 'SUCCESS wVnWw3a41CVe3mBVvTMn'"
	startupScript, err := ioutil.ReadFile("gce_ovf_import_tests/scripts/ovf_import_test_ubuntu_3_disks.sh")
	if err != nil {
		os.Exit(1)
	}
	testSetup := []*ovfImportTestSetup{
		{
			importParams: &ovfimportparams.OVFImportParams{
				ClientID:      "test",
				InstanceNames: fmt.Sprintf("test-instance-ubuntu-3mounteddisks-%v", suffix),
				OvfOvaGcsPath: "gs://zoran-playground/ova/Ubuntu_for_Horizon_three_disks_mounted.ova",
				OsID:          "ubuntu-1404",
				Labels:        "lk1=lv1,lk2=kv2",
				Project:       testProjectConfig.TestProjectID,
				Zone:          testProjectConfig.TestZone,
				MachineType:   "n1-standard-2",
			},
			name:          fmt.Sprintf("ovf-import-test-%s", suffix),
			startup:       computeUtils.BuildInstanceMetadataItem("startup-script", string(startupScript)),
			assertTimeout: 7200 * time.Second,
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

func ovfImportTestCase(ctx context.Context, testSetup *ovfImportTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer wg.Done()

	ovfImportTestCase := junitxml.NewTestCase(testSuiteName, "[OVFImport] Import OVF")

	for tc, f := range map[*junitxml.TestCase]func(context.Context, *junitxml.TestCase, *ovfImportTestSetup, *log.Logger, *testconfig.Project){
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

func runOvfImportTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *ovfImportTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {
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
	//instanceName := "test-instance-ubuntu-3mounteddisks-v0352"

	instance, err := client.GetInstance(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
	if !strings.HasSuffix(instance.MachineType, testSetup.importParams.MachineType) {
		testCase.WriteFailure("Instance machine type `%v` doesn't match requested machine type `%v`", instance.MachineType, testSetup.importParams.MachineType)
		return
	}

	if !strings.HasSuffix(instance.Zone, testSetup.importParams.Zone) {
		testCase.WriteFailure("Instance zone `%v` doesn't match requested zone `%v`", instance.Zone, testSetup.importParams.Zone)
		return
	}

	logger.Printf("Stopping instance before restarting with test startup script")
	err = client.StopInstance(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return
	}

	logger.Printf("Setting instance metadata with test startup script")
	err = client.SetInstanceMetadata(testProjectConfig.TestProjectID, testProjectConfig.TestZone,
		instanceName, &api.Metadata{Items: []*api.MetadataItems{testSetup.startup}, Fingerprint: instance.Metadata.Fingerprint})
	if err != nil {
		testCase.WriteFailure("Couldn't set instance metadata to verify OVF import: %v", err)
		return
	}

	logger.Printf("Starting instance with test startup script")
	err = client.StartInstance(testProjectConfig.TestProjectID, testProjectConfig.TestZone, instanceName)
	if err != nil {
		testCase.WriteFailure("Couldn't start instance to verify OVF import: %v", err)
		return
	}

	inst := computeUtils.Instance{Instance: instance, Client: client,
		Project: testProjectConfig.TestProjectID, Zone: testProjectConfig.TestZone}

	if err := inst.WaitForSerialOutput("PASSED: All tests passed!", 1, 5*time.Second, 7*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
		return
	}

	inst.Cleanup()
}
