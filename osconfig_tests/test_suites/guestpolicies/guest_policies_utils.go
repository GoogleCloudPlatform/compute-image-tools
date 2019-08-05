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

package guestpolicies

import (
	"fmt"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/google/logger"
	computeApi "google.golang.org/api/compute/v1"
)

var (
	yumStartupScripts = map[string]string{
		"rhel-6":   utils.InstallOSConfigEL6(),
		"rhel-7":   utils.InstallOSConfigEL7(),
		"rhel-8":   utils.InstallOSConfigEL8(),
		"centos-6": utils.InstallOSConfigEL6(),
		"centos-7": utils.InstallOSConfigEL7(),
	}
)

func getStartupScript(image, pkgManager, packageName string) *computeApi.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = `%s
while true; do
  isinstalled=$(/usr/bin/dpkg-query -s %s)
  if [[ $isinstalled =~ "Status: install ok installed" ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5
done`

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb(), packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "yum":
		ss = `%s
while true; do
  isinstalled=$(/usr/bin/rpmquery -a %s)
  if [[ $isinstalled =~ ^%s-* ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5
done`
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "googet":
		ss = `%s
while(1) {
  $installed_packages = googet installed
  if ($installed_packages -like "*%s*") {
	$uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  } else {
	$uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  }
  sleep 5
}`
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet(), packageName, packageInstalled, packageNotInstalled)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &computeApi.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}

func getUpdateStartupScript(image, pkgManager, packageName string) *computeApi.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = `%s
sleep 30
while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
   sleep 5
done
apt-get -y remove %[2]s || exit 1
apt-get -y install %[2]s=3.03+dfsg1-10 || exit 1
systemctl restart google-osconfig-agent
sleep 30
while fuser /var/lib/dpkg/lock >/dev/null 2>&1; do
   sleep 5
done
while true; do
  isinstalled=$(/usr/bin/dpkg-query -s %[2]s)
  if [[ $isinstalled =~ "Version: 3.03+dfsg1-10" ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5;
done`

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb(), packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "yum":
		restartAgent := "systemctl restart google-osconfig-agent"
		if strings.HasSuffix(image, "-6") {
			restartAgent = "restart -q -n google-osconfig-agent"
		}
		ss = `%s
sleep 20
cat > /etc/yum.repos.d/google-osconfig-agent.repo <<EOM
[test-repo]
name=test repo
baseurl=https://packages.cloud.google.com/yum/repos/osconfig-agent-test-repository
enabled=1
gpgcheck=0
EOM
n=0
while ! yum -y remove %[2]s; do
  if [[ n -gt 5 ]]; then
    exit 1
  fi
  n=$[$n+1]
  sleep 10
done
yum -y install %[2]s-3.03-2.fc7 || exit 1
%[3]s
sleep 20
while true; do
  isinstalled=$(/usr/bin/rpmquery -a %[2]s)
  if [[ $isinstalled =~ 3.03-2.fc7 ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[4]s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[5]s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5
done`
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, restartAgent, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "googet":
		ss = `%s
sleep 60
googet -noconfirm remove %[2]s
googet -noconfirm install %[2]s.x86_64.0.1.0@1
Restart-Service google_osconfig_agent
sleep 60
while(1) {
  $installed_packages = googet installed %[2]s
  Write-Host $installed_packages
  if ($installed_packages -like "*0.1.0@1*") {
    $uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  } else {
    $uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  }
  sleep 5
}`
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet(), packageName, packageInstalled, packageNotInstalled)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &computeApi.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}
