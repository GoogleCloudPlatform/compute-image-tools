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

// Package instanceovfexporttestsuite contains e2e tests for instance export cli tools
package instanceovfexporttestsuite

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

	computeBeta "google.golang.org/api/compute/v0.beta"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests/e2e"
	"github.com/GoogleCloudPlatform/compute-image-tools/common/gcp"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/test_config"
)

const (
	testSuiteName = "OVFInstanceExportTests"
)

var (
	cmds = map[e2e.CLITestType]string{
		e2e.Wrapper: "./gce_ovf_export",
	}
	// Apply this as instance metadata if the OS config agent is not
	// supported for the platform or version being exported.
	skipOSConfigMetadata = map[string]string{"osconfig_not_supported": "true"}
)

type instanceOvfExportTestProperties struct {
	instanceName              string
	isWindows                 bool
	expectedStartupOutput     string
	failureMatches            []string
	verificationStartupScript string
	zone                      string
	sourceGMI                 string
	os                        string
	network                   string
	subnet                    string
	instanceMetadata          map[string]string
	destinationURI            string
	buildID                   string
	exportBucket              string
	exportPath                string
}

// TestSuite is instance OVF export test suite.
func TestSuite(
	ctx context.Context, tswg *sync.WaitGroup, testSuites chan *junitxml.TestSuite,
	logger *log.Logger, testSuiteRegex, testCaseRegex *regexp.Regexp,
	testProjectConfig *testconfig.Project) {

	testsMap := map[e2e.CLITestType]map[*junitxml.TestCase]func(
		context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}

	testTypes := []e2e.CLITestType{
		e2e.Wrapper,
	}
	for _, testType := range testTypes {
		instanceExportUbuntu3DisksTestCase := junitxml.NewTestCase(
			testSuiteName, fmt.Sprintf("[%v][OVFInstanceExport] %v", testType, "Ubuntu 3 disks, one data disk larger than 10GB"))

		testsMap[testType] = map[*junitxml.TestCase]func(
			context.Context, *junitxml.TestCase, *log.Logger, *testconfig.Project, e2e.CLITestType){}
		testsMap[testType][instanceExportUbuntu3DisksTestCase] = runInstanceOVFExportUbuntu3Disks
	}

	e2e.CLITestSuite(ctx, tswg, testSuites, logger, testSuiteRegex, testCaseRegex,
		testProjectConfig, testSuiteName, testsMap)
}

func runInstanceOVFExportUbuntu3Disks(ctx context.Context, testCase *junitxml.TestCase, logger *log.Logger,
	testProjectConfig *testconfig.Project, testType e2e.CLITestType) {

	buildID := path.RandString(10)

	exportBucket := fmt.Sprintf("%v-test-image", testProjectConfig.TestProjectID)
	exportPath := fmt.Sprintf("ovf-export-%v", buildID)
	props := &instanceOvfExportTestProperties{
		instanceName: fmt.Sprintf("test-instance-ubuntu-3-disks-%v", buildID),
		verificationStartupScript: loadScriptContent(
			"scripts/ovf_import_test_ubuntu_3_disks.sh", logger),
		zone:                  testProjectConfig.TestZone,
		expectedStartupOutput: "All tests passed!",
		failureMatches:        []string{"FAILED:", "TestFailed:"},
		sourceGMI:             "projects/compute-image-test-pool-001/global/machineImages/ubuntu-1604-three-disks-do-not-delete",
		os:                    "ubuntu-1604",
		destinationURI:        fmt.Sprintf("gs://%v/%v/", exportBucket, exportPath),
		exportBucket:          exportBucket,
		exportPath:            exportPath,
		buildID:               buildID,
	}

	runOVFInstanceExportTest(ctx, buildTestArgs(props, testProjectConfig)[testType], testType, testProjectConfig, logger, testCase, props)
}

func buildTestArgs(props *instanceOvfExportTestProperties, testProjectConfig *testconfig.Project) map[e2e.CLITestType][]string {
	wrapperArgs := []string{"-client-id=e2e", fmt.Sprintf("-project=%v", testProjectConfig.TestProjectID),
		fmt.Sprintf("-instance-name=%s", props.instanceName),
		fmt.Sprintf("-destination-uri=%v", props.destinationURI),
		fmt.Sprintf("-zone=%v", testProjectConfig.TestZone),
		fmt.Sprintf("-build-id=%v", props.buildID),
	}
	if props.os != "" {
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-os=%v", props.os))
	}
	if props.network != "" {
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-network=%v", props.network))
	}
	if props.subnet != "" {
		wrapperArgs = append(wrapperArgs, fmt.Sprintf("-subnet=%v", props.subnet))
	}

	argsMap := map[e2e.CLITestType][]string{
		e2e.Wrapper: wrapperArgs,
	}
	return argsMap
}

func runOVFInstanceExportTest(ctx context.Context, args []string, testType e2e.CLITestType,
	testProjectConfig *testconfig.Project, logger *log.Logger, testCase *junitxml.TestCase,
	props *instanceOvfExportTestProperties) {

	computeClient, err := daisyCompute.NewClient(ctx)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating computeClient: %v", err))
		return
	}
	logger.Printf("Creating a temporary virtual machine instance `%v` to export to OVF...", props.instanceName)
	err = computeClient.CreateInstanceBeta(testProjectConfig.TestProjectID, props.zone, &computeBeta.Instance{
		Name:               props.instanceName,
		SourceMachineImage: props.sourceGMI,
	})

	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Error creating instance to export: %v", err))
		return
	}

	instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, props.zone, props.instanceName, props.isWindows)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance to export'%v' doesn't exist: %v", props.instanceName, err))
		return
	}

	defer func() {
		logger.Printf("Deleting temporary virtual machine instance `%v`", props.instanceName)
		if err := instance.Cleanup(); err != nil {
			logger.Printf("Temporary virtual machine instance '%v' failed to clean up: %v", props.instanceName, err)
		} else {
			logger.Printf("Temporary virtual machine instance '%v' cleaned up.", props.instanceName)
		}
	}()

	logger.Printf("Exporting a temporary virtual machine instance `%v` to OVF at: %v", props.instanceName, props.destinationURI)
	if e2e.RunTestForTestType(cmds[testType], args, testType, logger, testCase) {
		if verifyExportedInstanceIsUnmodifiedAfterExport(ctx, testCase, testProjectConfig, logger, props) {
			if ok, filesToClean := verifyExportedOVFContent(ctx, testCase, logger, props, instance); ok {
				defer cleanupGCSFiles(filesToClean, logger)
				verifyExportedOVFUsingOVFImport(ctx, testCase, testProjectConfig, logger, props, computeClient)
			}
		}
	}
}

func verifyExportedInstanceIsUnmodifiedAfterExport(
	ctx context.Context, testCase *junitxml.TestCase, testProjectConfig *testconfig.Project,
	logger *log.Logger, props *instanceOvfExportTestProperties) bool {

	logger.Printf("Verifying exported instance is unmodified after export...")
	instance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, props.zone, props.instanceName, props.isWindows)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance '%v' doesn't exist after export: %v", props.instanceName, err))
		return false
	}

	attachedDisks := instance.Instance.Disks
	if attachedDisks == nil || len(attachedDisks) != 3 {
		e2e.Failure(testCase, logger, "Exported instance should have 3 disks attached after export")
		return false
	}

	if instance.Instance.Status != "RUNNING" {
		e2e.Failure(testCase, logger,
			fmt.Sprintf(
				"Exported instance should be in `RUNNING` state, but the state is `%v`", instance.Instance.Status))
		return false
	}
	logger.Printf("Exported instance verified to be in the original sate...")
	return true
}

func verifyExportedOVFContent(ctx context.Context, testCase *junitxml.TestCase,
	logger *log.Logger, props *instanceOvfExportTestProperties, instance *gcp.Instance) (bool, []*gcp.File) {

	filePaths := []string{
		fmt.Sprintf("%v/%v.mf", props.exportPath, props.instanceName),
		fmt.Sprintf("%v/%v.ovf", props.exportPath, props.instanceName),
	}
	for _, attachedDisk := range instance.Instance.Disks {
		filePaths = append(filePaths, fmt.Sprintf("%v/%v-%v.vmdk", props.exportPath, props.instanceName, attachedDisk.DeviceName))
	}

	var fileErrors []error
	var filePathsForError []string
	var filesToClean []*gcp.File
	for _, filePath := range filePaths {
		file, fileError := verifyFileInGCSExists(ctx, testCase, props.exportBucket, filePath, logger)
		filesToClean = append(filesToClean, file)
		if fileError != nil {
			fileErrors = append(fileErrors, fileError)
			filePathsForError = append(filePathsForError, filePath)
		}
	}
	if len(fileErrors) != 0 {
		var stringBuilder strings.Builder
		for i, fileError := range fileErrors {
			fmt.Fprintf(&stringBuilder, "File '%v' don't exist after export: %v\n", filePathsForError[i], fileError)
		}
		e2e.Failure(testCase, logger, stringBuilder.String())
		return false, filesToClean
	}
	return true, filesToClean
}

func verifyExportedOVFUsingOVFImport(
	ctx context.Context, testCase *junitxml.TestCase, testProjectConfig *testconfig.Project,
	logger *log.Logger, props *instanceOvfExportTestProperties, computeClient daisyCompute.Client) bool {

	verificationInstanceName := fmt.Sprintf("ovf-export-verification-instance--%v", props.buildID)
	logger.Printf("Verifying exported OVF by importing it via OVF import as `%v`", verificationInstanceName)

	ovfImportError := e2e.RunCliTool(logger, testCase, "gcloud", []string{
		"compute",
		"instances",
		"import",
		verificationInstanceName,
		fmt.Sprintf("--source-uri=%v", props.destinationURI),
		fmt.Sprintf("--os=%v", props.os),
		fmt.Sprintf("--zone=%v", props.zone),
		fmt.Sprintf("--project=%v", testProjectConfig.TestProjectID),
	})
	if ovfImportError != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Failed importing exported OVF as instance: %v", ovfImportError))
		return false
	}

	logger.Printf("Verifying imported instance...")
	verificationInstance, err := gcp.CreateInstanceObject(ctx, testProjectConfig.TestProjectID, props.zone, verificationInstanceName, props.isWindows)
	if err != nil {
		e2e.Failure(testCase, logger, fmt.Sprintf("Verification instance '%v' doesn't exist after import: %v", verificationInstanceName, err))
		return false
	}

	defer func() {
		logger.Printf("Deleting verification instance `%v`", verificationInstanceName)
		if err := verificationInstance.Cleanup(); err != nil {
			logger.Printf("Verification instance '%v' failed to clean up: %v", verificationInstanceName, err)
		} else {
			logger.Printf("Verification instance '%v' cleaned up.", verificationInstanceName)
		}
	}()

	// The boot disk for a Windows instance must have the WINDOWS GuestOSFeature,
	// while the boot disk for other operating systems shouldn't have it.
	for _, disk := range verificationInstance.Disks {
		if !disk.Boot {
			continue
		}

		hasWindowsFeature := false
		for _, feature := range disk.GuestOsFeatures {
			if "WINDOWS" == feature.Type {
				hasWindowsFeature = true
				break
			}
		}

		if props.isWindows && !hasWindowsFeature {
			testCase.WriteFailure(
				"Windows boot disk missing WINDOWS GuestOsFeature. Features found=%v",
				disk.GuestOsFeatures)
			return false
		} else if !props.isWindows && hasWindowsFeature {
			testCase.WriteFailure(
				"Non-Windows boot disk includes WINDOWS GuestOsFeature. Features found=%v",
				disk.GuestOsFeatures)
			return false
		}
	}
	if !strings.HasSuffix(verificationInstance.Zone, props.zone) {
		e2e.Failure(testCase, logger, fmt.Sprintf("Instance zone `%v` doesn't match requested zone `%v`",
			verificationInstance.Zone, props.zone))
		return false
	}

	logger.Printf("[%v] Stopping instance before restarting with test startup script", verificationInstanceName)
	err = computeClient.StopInstance(
		testProjectConfig.TestProjectID, props.zone, verificationInstanceName)

	if err != nil {
		testCase.WriteFailure("Error stopping imported instance: %v", err)
		return false
	}

	if props.verificationStartupScript == "" {
		logger.Printf("[%v] Will not set test startup script to instance metadata as it's not defined", verificationInstanceName)
		return false
	}

	logger.Printf("[%v] Starting instance with test startup script", verificationInstanceName)
	err = verificationInstance.StartWithScriptCode(props.verificationStartupScript, props.instanceMetadata)
	if err != nil {
		testCase.WriteFailure("Error starting instance `%v` with script: %v", verificationInstanceName, err)
		return false
	}
	logger.Printf("[%v] Waiting for `%v` in instance serial console.", verificationInstanceName,
		props.expectedStartupOutput)
	if err := verificationInstance.WaitForSerialOutput(
		props.expectedStartupOutput, props.failureMatches, 1, 5*time.Second, 15*time.Minute); err != nil {
		testCase.WriteFailure("Error during VM validation: %v", err)
		return false
	}
	return true
}

func loadScriptContent(scriptPath string, logger *log.Logger) string {
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		logger.Printf("Error loading script `%v`: %v", scriptPath, err)
		os.Exit(1)
	}
	return string(scriptContent)
}

func verifyFileInGCSExists(ctx context.Context, testCase *junitxml.TestCase, bucketName string,
	objectName string, logger *log.Logger) (*gcp.File, error) {

	logger.Printf("Verifying exported file gs://%v/%v", bucketName, objectName)
	file, err := gcp.CreateFileObject(ctx, bucketName, objectName)
	if err != nil {
		testCase.WriteFailure("File '%v' doesn't exist after export: %v", objectName, err)
		logger.Printf("File '%v' doesn't exist after export: %v", objectName, err)
		return file, err
	}
	logger.Printf("File '%v' exists!", objectName)

	return file, nil
}

func cleanupGCSFiles(filesToClean []*gcp.File, logger *log.Logger) {
	for _, fileToClean := range filesToClean {
		if err := fileToClean.Cleanup(); err != nil {
			logger.Printf("File '%v' failed to clean up.", fileToClean.FileObject.ObjectName())
		} else {
			logger.Printf("File '%v' cleaned up.", fileToClean.FileObject.ObjectName())
		}
	}
}
