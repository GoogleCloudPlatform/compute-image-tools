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

package ospatch

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
)

// disableAutoUpdates disables system auto updates.
func disableAutoUpdates() {
	// yum-cron on el systems
	if _, err := os.Stat("/usr/sbin/yum-cron"); err == nil {
		out, err := exec.Command("chkconfig", "yum-cron").CombinedOutput()
		if err != nil {
			logger.Errorf("error checking status of yum-cron, error: %v, out: %s", err, out)
		}
		if bytes.Contains(out, []byte("disabled")) {
			return
		}

		logger.Debugf("Disabling yum-cron")
		out, err = exec.Command("chkconfig", "yum-cron", "off").CombinedOutput()
		if err != nil {
			logger.Errorf("error disabling yum-cron, error: %v, out: %s", err, out)
		}
	}

	// apt unattended-upgrades
	// TODO: Removing the package is a bit overkill, look into just managing
	// the configs, this is probably best done by looking through
	// /etc/apt/apt.conf.d/ and setting APT::Periodic::Unattended-Upgrade to 0.
	if _, err := os.Stat("/usr/bin/unattended-upgrades"); err == nil {
		logger.Debugf("Removing unattended-upgrades package")
		out, err := exec.Command("apt-get", "remove", "-y", "unattended-upgrades").CombinedOutput()
		if err != nil {
			logger.Errorf("error disabling unattended-upgrades, error: %v, out: %s", err, out)
		}
	}
}
