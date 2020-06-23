//  Copyright 2020 Google Inc. All Rights Reserved.
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

package upgrader

import (
	"context"
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

// Parameter key shared with external packages
const (
	ClientIDFlagKey = "client-id"
	DefaultTimeout  = "90m"
)

const (
	logPrefix = "[windows-upgrade]"

	metadataKeyWindowsStartupScriptURL       = "windows-startup-script-url"
	metadataKeyWindowsStartupScriptURLBackup = "windows-startup-script-url-backup"

	versionWindows2008r2 = "windows-2008r2"
	versionWindows2012r2 = "windows-2012r2"
)

var (
	upgradeScriptName = map[string]string{versionWindows2008r2: "upgrade_script_2008r2_to_2012r2.ps1"}

	expectedCurrentLicense = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2008-r2-dc"}
	licenseToAdd           = map[string]string{versionWindows2008r2: "projects/windows-cloud/global/licenses/windows-server-2012-r2-dc-in-place-upgrade"}
)

type derivedVars struct {
	instanceProject string
	instanceZone    string
	instanceURI     string

	osDiskURI        string
	osDiskType       string
	osDiskDeviceName string
	osDiskAutoDelete bool

	instanceName           string
	machineImageBackupName string
	osDiskSnapshotName     string
	newOSDiskName          string
	installMediaDiskName   string

	originalWindowsStartupScriptURL *string
}

// InputParams contains input params for the upgrade.
type InputParams struct {
	ClientID               string
	ProjectPtr             *string
	Zone                   string
	Instance               string
	CreateMachineBackup    bool
	AutoRollback           bool
	SourceOS               string
	TargetOS               string
	Timeout                string
	UseStagingInstallMedia bool
	ScratchBucketGcsPath   string
	Oauth                  string
	Ce                     string
	GcsLogsDisabled        bool
	CloudLogsDisabled      bool
	StdoutLogsDisabled     bool
}

type upgrader struct {
	*InputParams
	*derivedVars

	ctx context.Context

	initFn                    func() error
	printIntroHelpTextFn      func() error
	validateAndDeriveParamsFn func() error
	prepareFn                 func() (*daisy.Workflow, error)
	upgradeFn                 func() (*daisy.Workflow, error)
	retryUpgradeFn            func() (*daisy.Workflow, error)
	rebootFn                  func() (*daisy.Workflow, error)
	cleanupFn                 func() (*daisy.Workflow, error)
	rollbackFn                func() (*daisy.Workflow, error)
}

// Run runs upgrader.
func Run(p *InputParams) (service.Loggable, error) {
	u := upgrader{InputParams: p}
	return u.run()
}

func (u *upgrader) run() (service.Loggable, error) {
	if err := u.init(); err != nil {
		return nil, err
	}
	if err := u.validateAndDeriveParams(); err != nil {
		return nil, err
	}
	if err := u.printIntroHelpText(); err != nil {
		return nil, err
	}
	w, err := u.runUpgradeWorkflow()
	return service.NewLoggableFromWorkflow(w), err
}

func (u *upgrader) init() error {
	if u.initFn != nil {
		return u.initFn()
	}

	log.SetPrefix(logPrefix + " ")

	var err error
	u.ctx = context.Background()
	computeClient, err = daisyCompute.NewClient(u.ctx, option.WithCredentialsFile(u.Oauth))
	mgce = &compute.MetadataGCE{}
	if err != nil {
		return daisy.Errf("Failed to create GCE client: %v", err)
	}
	return nil
}

func (u *upgrader) printIntroHelpText() error {
	if u.printIntroHelpTextFn != nil {
		return u.printIntroHelpTextFn()
	}

	guide, err := getIntroHelpText(u)
	if err != nil {
		return err
	}
	fmt.Print(guide, "\n\n")
	return nil
}

func (u *upgrader) runUpgradeWorkflow() (*daisy.Workflow, error) {
	var err error

	// If upgrade failed, run cleanup or rollback before exiting.
	defer func() {
		u.handleResult(err)
	}()

	// step 1: preparation - take snapshot, attach install media, backup/set startup script
	fmt.Print("\nPreparing for upgrade...\n\n")
	prepareWf, err := u.prepare()
	if err != nil {
		return prepareWf, err
	}

	// step 2: run upgrade.
	fmt.Print("\nRunning upgrade...\n\n")
	upgradeWf, err := u.upgrade()
	if err == nil {
		return upgradeWf, nil
	}

	// step 3: reboot if necessary.
	if !needReboot(err) {
		return upgradeWf, err
	}
	fmt.Print("\nRebooting...\n\n")
	rebootWf, err := u.reboot()
	if err != nil {
		return rebootWf, err
	}

	// step 4: retry upgrade.
	fmt.Print("\nRetrying upgrade...\n\n")
	retryUpgradeWf, err := u.retryUpgrade()
	return retryUpgradeWf, err
}

func (u *upgrader) handleResult(err error) {
	if err == nil {
		fmt.Printf("\nSuccessfully upgraded instance '%v' to '%v'.\n", u.instanceURI, u.TargetOS)
		// TODO: update the help guide link. b/154838004
		fmt.Printf("\nPlease verify your application's functionality on the " +
			"instance, and if you run into any issues, please manually rollback following " +
			"the instructions in the guide.\n\n")
		if cleanupIntro, err := getCleanupIntroduction(u); err == nil {
			fmt.Printf(cleanupIntro)
		}
		return
	}

	isNewOSDiskAttached := isNewOSDiskAttached(u.instanceProject, u.instanceZone, u.instanceName, u.newOSDiskName)
	if u.AutoRollback {
		if isNewOSDiskAttached {
			fmt.Printf("\nUpgrade failed to finish. Rolling back to the "+
				"original state from the original boot disk '%v'...\n\n", u.osDiskURI)
			_, err := u.rollback()
			if err != nil {
				fmt.Printf("\nRollback failed. Error: %v\n"+
					"Please rollback the image manually following the instructions in the guide.\n\n", err)
			} else {
				fmt.Printf("\nCompleted rollback to the original boot disk. Please " +
					"verify the rollback. If the rollback does not function as expected, " +
					"consider restoring the instance from the machine image.\n\n")
			}
			return
		}
		fmt.Printf("\nNo boot disk attached during the failure. No need to rollback. "+
			"If the instance doesn't work as expected, please verify that the original "+
			"boot disk (%v) is attached and whether the instance has started. If necessary, "+
			"please manually rollback by using the instructions in the guide..\n\n", u.osDiskURI)
	}

	fmt.Printf("\nUpgrade failed. Please manually rollback following the " +
		"instructions in the guide.\n\n")

	fmt.Print("\nCleaning up temporary resources...\n\n")
	if _, err := u.cleanup(); err != nil {
		fmt.Printf("\nFailed to cleanup temporary resources: %v\n"+
			"Please cleanup the resources manually by following the instructions in the guide.\n\n", err)
	}
}
