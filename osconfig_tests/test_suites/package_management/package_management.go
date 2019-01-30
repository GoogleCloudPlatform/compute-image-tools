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
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/junitxml"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/kylelemons/godebug/pretty"
	api "google.golang.org/api/compute/v1"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
)

const (
	testSuiteName = "PackageManagementTests"
	testProject   = "compute-image-test-pool-001"
	testZone      = "us-central1-c"

	// dpkg-query
	dpkgListCmd = "/usr/bin/dpkg-query -W"

	serviceAccountEmail  = "281997379984-compute@developer.gserviceaccount.com"
	serviceAccountScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}

	writeToSerialConsole = " | sudo tee /dev/ttyS0"

	dump = &pretty.Config{IncludeUnexported: true}
)

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
				Value: &utils.InstallOSConfigDeb,
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

func runCreateOsConfigTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger) {

	oc := &osconfigserver.OsConfig{
		&osconfigpb.OsConfig{
			Name: "createosconfig-test-osconfig",
		},
	}

	logger.Printf("create osconfig request:\n%s\n\n", dump.Sprint(oc))

	parent := fmt.Sprintf("projects/%s", testProject)
	res, err := osconfigserver.CreateOsConfig(ctx, logger, oc, parent)

	defer cleanupOsConfig(ctx, testCase, logger, oc)

	if err != nil {
		testCase.WriteFailure("error while creating osconfig:\n%s\n\n", err)
	}

	logger.Printf("CreateOsConfig response:\n%s\n\n", dump.Sprint(res))
}

func runPackageInstallTest(ctx context.Context, testCase *junitxml.TestCase, testSetup *packageManagementTestSetup, logger *log.Logger) {

	osConfig, err := osconfigserver.JsonToOsConfig(packageInstallTestOsConfigString, logger)

	if err != nil {
		testCase.WriteFailure("error while converting json to osconfig: \n%s\n", err)
		return
	}

	oc := &osconfigserver.OsConfig{OsConfig: osConfig}

	parent := fmt.Sprintf("projects/%s", testProject)
	osconfigserver.CreateOsConfig(ctx, logger, oc, parent)

	if err != nil {
		testCase.WriteFailure("error while creating osconfig: \n%s\n", err)
		return
	}

	defer cleanupOsConfig(ctx, testCase, logger, oc)

	assignment, err := osconfigserver.JsonToAssignment(packageInstallTestAssignmentString, logger)
	if err != nil {
		testCase.WriteFailure("error while converting json to assignment: \n%s\n", err)
		return
	}

	assign := &osconfigserver.Assignment{Assignment: assignment}

	_, err := osconfigserver.CreateAssignment(ctx, logger, assign, parent)
	if err != nil {
		testCase.WriteFailure("error while creating assignment: \n%s\n", err)
	}

	defer cleanupAssignment(ctx, testCase, logger, assign)

	client, err := daisyCompute.NewClient(ctx)
	if err != nil {
		testCase.WriteFailure("error creating client: %v", err)
		return
	}

	testCase.Logf("Creating instance with image %q", testSetup.image)
	testSetup.name = getTestSetupName(testSetup.image, "packageInstallTest")
	i := &api.Instance{
		Name:        testSetup.name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", testProject, testZone),
		NetworkInterfaces: []*api.NetworkInterface{
			&api.NetworkInterface{
				Network: "global/networks/default",
				AccessConfigs: []*api.AccessConfig{
					&api.AccessConfig{
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Metadata: &api.Metadata{
			Items: []*api.MetadataItems{
				testSetup.startup,
			},
		},
		Disks: []*api.AttachedDisk{
			&api.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				InitializeParams: &api.AttachedDiskInitializeParams{
					SourceImage: testSetup.image,
				},
			},
		},
		ServiceAccounts: []*api.ServiceAccount{
			&api.ServiceAccount{
				Email:  serviceAccountEmail,
				Scopes: serviceAccountScopes,
			},
		},
	}

	inst, err := compute.CreateInstance(client, testProject, testZone, i)
	if err != nil {
		testCase.WriteFailure("Error creating instance: %v", err)
		return
	}
	defer inst.Cleanup()

	testCase.Logf("Waiting for agent install to complete")
	if err := inst.WaitForSerialOutput("osconfig install done", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for osconfig agent install: %v", err)
		return
	}

	testCase.Logf("Agent installed successfully")

	// allow agent to make the lookupconfig call and install the package
	time.Sleep(1 * time.Minute)

	sshCmd := getGcloudSshCmd(testZone, testSetup.name, dpkgListCmd, logger)
	out, err := run(sshCmd, logger)

	if err != nil {
		testCase.WriteFailure("Error running verification command: %v", err)
		return
	}
	_ = out

	// TODO refactor to remove hardcoding of package name
	if err = inst.WaitForSerialOutput("cowsay", 1, 5*time.Second, 5*time.Minute); err != nil {
		testCase.WriteFailure("Error waiting for assertion: %v", err)
		return
	}
}

func getGcloudSshCmd(zone string, instance string, pkgManagerCommand string, logger *log.Logger) *exec.Cmd {
	return exec.Command("gcloud", []string{"compute", "ssh", "--zone",
		fmt.Sprintf("%s", zone),
		instance, "--command", fmt.Sprintf("%s %s\n",
			pkgManagerCommand, writeToSerialConsole)}...)
}

//TODO move this to common library
func run(cmd *exec.Cmd, logger *log.Logger) ([]byte, error) {
	logger.Printf("Running %q with args %q\n", cmd.Path, cmd.Args[1:])
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running %q with args %q: %v, stdout: %s", cmd.Path, cmd.Args, err, out)
	}
	return out, nil
}

func packageManagementTestCase(ctx context.Context, testSetup *packageManagementTestSetup, tests chan *junitxml.TestCase, wg *sync.WaitGroup, logger *log.Logger, regex *regexp.Regexp) {
	defer wg.Done()

	createOsConfigTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[CreateOsConfig] Create OsConfig"))
	packageInstallTest := junitxml.NewTestCase(testSuiteName, fmt.Sprintf("[PackageInstall] Pacakge installation"))

	for tc, f := range map[*junitxml.TestCase]func(context.Context, *junitxml.TestCase, *packageManagementTestSetup, *log.Logger){
		createOsConfigTest: runCreateOsConfigTest,
		packageInstallTest: runPackageInstallTest,
	} {
		if tc.FilterTestCase(regex) {
			tc.Finish(tests)
		} else {
			logger.Printf("Running TestCase %s.%q", tc.Classname, tc.Name)
			f(ctx, tc, testSetup, logger)
			tc.Finish(tests)
			logger.Printf("TestCase %s.%q finished in %fs", tc.Classname, tc.Name, tc.Time)
		}
	}
}

func cleanupOsConfig(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger, oc *osconfigserver.OsConfig) {
	err := oc.Cleanup(ctx, logger)
	if err != nil {
		testCase.WriteFailure(fmt.Sprintf("error while deleting osconfig: %s", err))
	}
}

func cleanupAssignment(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger, assignment *osconfigserver.Assignment) {
	err := assignment.Cleanup(ctx, logger)
	if err != nil {
		testCase.WriteFailure(fmt.Sprintf("error while deleting assignment: %s", err))
	}
}

func getTestSetupName(imageName string, testName string) {
	return fmt.Sprintf("osconfig-test-%s-%s", imageName, testName)
}
