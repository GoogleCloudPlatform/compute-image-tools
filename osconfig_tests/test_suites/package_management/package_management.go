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

package packagemanagement

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
)

const testSuiteName = "PackageManagementTests"

// TODO: Should these be configurable via flags?
const testProject = "compute-image-test-pool-001"
const testZone = "us-central1-c"

var dump = &pretty.Config{IncludeUnexported: true}

var installOsConfigAgentDeb = `echo 'deb http://packages.cloud.google.com/apt google-osconfig-agent-stretch-unstable main' >> /etc/apt/sources.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update
apt-get install -y google-osconfig-agent
echo 'osconfig agent install done'`

var installOsConfigAgentGooGet = `c:\programdata\googet\googet.exe -noconfirm install -sources https://packages.cloud.google.com/yuck/repos/google-osconfig-agent-unstable google-osconfig-agent
echo 'osconfig agent install done'`

var installOsConfigAgentYumEL7 = `cat > /etc/yum.repos.d/google-osconfig-agent.repo <<EOM
[google-osconfig-agent]
name=Google OSConfig Agent Repository
baseurl=https://packages.cloud.google.com/yum/repos/google-osconfig-agent-el7-unstable
enabled=1
gpgcheck=0
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM
yum -y install google-osconfig-agent
echo 'osconfig agent install done'`

type packageManagementTestSetup struct {
	image       string
	name        string
	packageType []string
	shortName   string
	startup     *api.MetadataItems
}

// TestSuite is a PackageManagementTests test suite.
func TestSuite(ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite, logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp) {
	defer tswg.Done()

	if testSuiteRegex != nil && !testSuiteRegex.MatchString(testSuiteName) {
		return
	}

	testSuite := junitxml.NewTestSuite(testSuiteName)
	defer testSuite.Finish(testSuites)

	logger.Printf("Running TestSuite %q", testSuite.Name)

	testSetup := []*packageManagementTestSetup{
		// Debian images.
		&packageManagementTestSetup{
			image:       "projects/debian-cloud/global/images/family/debian-9",
			packageType: []string{"deb"},
			shortName:   "debian",
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &installOsConfigAgentDeb,
			},
		},
	}

	var wg sync.WaitGroup
	tests := make(chan *junitxml.TestCase)
	for _, setup := range testSetup {
		wg.Add(1)
		go packageManagementTestCase(ctx, setup, tests, &wg, logger, testCaseRegex)
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

func runCreateOsConfigTest(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger) {

	osConfig := &osconfigpb.OsConfig{
		Name: "create-osconfig-test",
	}

	defer osconfigserver.CleanupOsConfig(ctx, testCase, logger, osConfig)

	req := &osconfigpb.CreateOsConfigRequest{
		Parent:   "projects/compute-image-test-pool-001",
		OsConfig: osConfig,
	}

	logger.Printf("create osconfig request:\n%s\n\n", dump.Sprint(req))

	res, err := osconfigserver.CreateOsConfig(ctx, logger, req)
	if err != nil {
		testCase.WriteFailure("error while creating osconfig:\n%s\n\n", err)
	}

	logger.Printf("CreateOsConfig response:\n%s\n\n", dump.Sprint(res))
}

func packageManagementTestCase(ctx context.Context, testSetup *packageManagementTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp) {
	defer wg.Done()

	createOsConfigTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[CreateOsConfig] Create OsConfig"))

	for tc, f := range map[*junitxml.TestCase]func(context.Context, *junitxml.TestCase, *log.Logger){
		createOsConfigTest: runCreateOsConfigTest,
	} {
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
			f(ctx, tc, logger)
			tc.Finish(tests)
			logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
		}
	}
}
