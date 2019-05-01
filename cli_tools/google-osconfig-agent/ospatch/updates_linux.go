//  Copyright 2018 Google Inc. All Rights Reserved.
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

//+build !test

package ospatch

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

const (
	systemctl = "/bin/systemctl"
	reboot    = "/bin/reboot"
	shutdown  = "/bin/shutdown"
)

func exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func systemRebootRequired() (bool, error) {
	switch {
	case packages.AptExists:
		logger.Debugf("Checking if reboot required by looking at /var/run/reboot-required.")
		rr, err := exists("/var/run/reboot-required")
		if err != nil {
			return false, err
		}
		if rr {
			logger.Debugf("/var/run/reboot-required exists indicating a reboot is required.")
			return true, nil
		}
		logger.Debugf("/var/run/reboot-required does not exist, indicating no reboot is required.")
		return false, nil
	case packages.YumExists:
		logger.Debugf("Checking if reboot required by querying /usr/bin/needs-restarting.")
		if e, _ := exists("/usr/bin/needs-restarting"); e {
			cmd := exec.Command("/usr/bin/needs-restarting", "-r")
			err := cmd.Run()
			if err == nil {
				logger.Debugf("'/usr/bin/needs-restarting -r' exit code 0 indicating no reboot is required.")
				return false, nil
			}
			if eerr, ok := err.(*exec.ExitError); ok {
				if eerr.ExitCode() == 1 {
					logger.Debugf("'/usr/bin/needs-restarting -r' exit code 1 indicating a reboot is required")
					return true, nil
				}
			}
			return false, err
		}
		logger.Infof("/usr/bin/needs-restarting does not exist, can't check if reboot is required. Try installing the 'yum-utils' package.")
		return false, nil
	case packages.ZypperExists:
		// TODO: implement
		logger.Errorf("systemRebootRequired not implemented for zypper")
		return false, nil
	}

	return false, fmt.Errorf("no recognized package manager installed, can't if reboot is required")
}

func runUpdates(r *patchRun) error {
	if r.reportState(osconfigpb.Instance_APPLYING_PATCHES) {
		return nil
	}
	return packages.UpdatePackages()
}

func rebootSystem() error {
	// Start with systemctl and work down a list of reboot methods.
	if e, _ := exists(systemctl); e {
		logger.Debugf("Rebooting using systemctl.")
		return exec.Command(systemctl, "reboot").Run()
	}
	if e, _ := exists(reboot); e {
		logger.Debugf("Rebooting using reboot command.")
		return exec.Command(reboot).Run()
	}
	if e, _ := exists(shutdown); e {
		logger.Debugf("Rebooting using shutdown command.")
		return exec.Command(shutdown, "-r", "-t", "0").Run()
	}

	// Fall back to reboot(2) system call
	logger.Debugf("No suitable reboot command found, rebooting using reboot(2).")
	syscall.Sync()
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}
