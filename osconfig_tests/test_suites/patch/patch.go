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

// Package patch contains end to end tests for patch management
package patch

import (
	"context"
	"fmt"
	"log"
	"path"
	"regexp"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	gcpclients "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/gcp_clients"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

const (
	testSuiteName = "PatchTests"
)

var (
	dump   = &pretty.Config{IncludeUnexported: true}
	suffix = utils.RandString(5)
)

type patchTestSetup struct {
	image         string
	startup       *api.MetadataItems
	assertTimeout time.Duration
}

// TestSuite is a PatchTests test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)

	logger.Printf("Running TestSuite %q", testSuite.Name)

	testSetup := []*patchTestSetup{
		// Windows images.
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2008-r2-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2012-r2-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2012-r2-core-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2016-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2016-core-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-1803-core-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-1809-core-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2019-core-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},
		&patchTestSetup{
			image:         "projects/compute-image-osconfig-agent/global/images/windows-2019-v20190515",
			assertTimeout: 10 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "windows-startup-script-ps1",
				Value: &utils.InstallOSConfigGooGet,
			},
		},

		// Debian images.
		&patchTestSetup{
			image:         "projects/debian-cloud/global/images/family/debian-9",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},

		// Centos images.
		&patchTestSetup{
			image:         "projects/centos-cloud/global/images/family/centos-6",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL6,
			},
		},
		&patchTestSetup{
			image:         "projects/centos-cloud/global/images/family/centos-7",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL7,
			},
		},

		// RHEL images.
		&patchTestSetup{
			image:         "projects/rhel-cloud/global/images/family/rhel-6",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL6,
			},
		},
		&patchTestSetup{
			image:         "projects/rhel-cloud/global/images/family/rhel-7",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigYumEL7,
			},
		},

		// Ubuntu images
		&patchTestSetup{
			image:         "projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},
		&patchTestSetup{
			image:         "projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts",
			assertTimeout: 5 * time.Minute,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &utils.InstallOSConfigDeb,
			},
		},
	}

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for _, setup := range testSetup {
		wg.Add(1)
		go patchTestCase(ctx, setup, tests, &wg, logger, testCaseRegex, testProjectConfig)
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

func runExecutePatchTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *patchTestSetup, logger *log.Logger, testProjectConfig *testconfig.Project) {

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	var metadataItems []*api.MetadataItems
	metadataItems = append(metadataItems, testSetup.startup)
	metadataItems = append(metadataItems, compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospatch"))
	testSetupName := fmt.Sprintf("patch-test-%s-%s", path.Base(testSetup.image), suffix)
	inst, err := utils.CreateComputeInstance(metadataItems, client, "n1-standard-2", testSetup.image, testSetupName, testProjectConfig.TestProjectID, testProjectConfig.GetZone(), testProjectConfig.ServiceAccountEmail, testProjectConfig.ServiceAccountScopes)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", utils.GetStatusFromError(err))
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// create patch job
	parent := fmt.Sprintf("projects/%s", testProjectConfig.TestProjectID)
	osconfigClient, err := gcpclients.GetOsConfigClient(ctx)

	assertTimeout := testSetup.assertTimeout

	req := &osconfigpb.ExecutePatchJobRequest{
		Parent:      parent,
		Description: "testing default patch job run",
		Filter:      fmt.Sprintf("name=\"%s\"", testSetupName),
		Duration:    &duration.Duration{Seconds: int64(assertTimeout / time.Second)},
	}
	job, err := osconfigClient.ExecutePatchJob(ctx, req)

	if err != nil {
		testCase.WriteFailure("error while executing patch job: \n%s\n", utils.GetStatusFromError(err))
		return
	}

	testCase.Logf("Started patch job '%s'", job.GetName())

	logger.Printf("%v\n", job)

	// assertion
	tick := time.Tick(5 * time.Second)
	timedout := time.Tick(testSetup.assertTimeout)
	for {
		select {
		case <-timedout:
			testCase.WriteFailure("Patch job '%s' timed out", job.GetName())
			return
		case <-tick:
			req := &osconfigpb.GetPatchJobRequest{
				Name: job.GetName(),
			}
			res, err := osconfigClient.GetPatchJob(ctx, req)
			if err != nil {
				testCase.WriteFailure("Error while fetching patch job: \n%s\n", utils.GetStatusFromError(err))
				return
			}

			if isPatchJobFailureState(res.State) {
				testCase.WriteFailure("Patch job '%s' completed with status %v and message '%s'", job.GetName(), res.State, job.GetErrorMessage())
				return
			}

			if res.State == osconfigpb.PatchJob_SUCCEEDED {
				if res.InstanceDetailsSummary.GetInstancesSucceeded() < 1 {
					testCase.WriteFailure("Patch job '%s' completed with no instances patched", job.GetName())
				}
				return
			}
		}
	}
}

func isPatchJobFailureState(state osconfigpb.PatchJob_State) bool {
	return state == osconfigpb.PatchJob_COMPLETED_WITH_ERRORS ||
		state == osconfigpb.PatchJob_TIMED_OUT ||
		state == osconfigpb.PatchJob_CANCELED
}

func patchTestCase(ctx context.Context, testSetup *patchTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp, testProjectConfig *testconfig.Project) {
	defer wg.Done()

	executePatchTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[executePatchTest] [%s] Execute PatchJob", testSetup.image))

	for tc, f := range map[*junitxml.TestCase]func(context.Context, *junitxml.TestCase, *patchTestSetup, *log.Logger, *testconfig.Project){
		executePatchTest: runExecutePatchTest,
	} {
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase %q", tc.Name)
			f(ctx, tc, testSetup, logger, testProjectConfig)
			tc.Finish(tests)
			logger.Printf("TestCase c%q finished in %fs", tc.Name, tc.Time)
		}
	}
}
