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

var waitForRestartLinux = `
echo 'Waiting for signal to restart agent'
while [[ -z $restarted ]]; do
  sleep 1
  restart=$(curl -f "http://metadata.google.internal/computeMetadata/v1/instance/attributes/restart-agent" -H "Metadata-Flavor: Google")
  if [[ -n $restart ]]; then
    systemctl restart google-osconfig-agent
    restart -q -n google-osconfig-agent  # required for EL6
    restarted=true
    sleep 30
  fi
done
`

var waitForRestartWin = `
echo 'Waiting for signal to restart agent'
while (! $restarted) {
  sleep 1
  $restart = Invoke-WebRequest -UseBasicParsing http://metadata.google.internal/computeMetadata/v1/instance/attributes/restart-agent -Headers @{"Metadata-Flavor"="Google"}
  if ($restart) {
    Restart-Service google_osconfig_agent
    $restarted = $true
    sleep 30
  }
}
`

func getStartupScript(image, pkgManager, packageName string) *computeApi.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = `systemctl stop google-osconfig-agent
%s
%s
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

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb(), waitForRestartLinux, packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "yum":
		ss = `systemctl stop google-osconfig-agent
stop -q -n google-osconfig-agent  # required for EL6
%s
%s
while true; do
  isinstalled=$(/usr/bin/rpmquery -a %[3]s)
  if [[ $isinstalled =~ ^%[3]s-* ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5
done`
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], waitForRestartLinux, packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "googet":
		ss = `Stop-Service google_osconfig_agent
googet addrepo test https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository
%s
%s
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
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet(), waitForRestartWin, packageName, packageInstalled, packageNotInstalled)
		key = "windows-startup-script-ps1"

	case "zypper":
		ss = `systemctl stop google-osconfig-agent
%s
%s
while true; do
  isinstalled=$(/usr/bin/rpmquery -a %[3]s)
  if [[ $isinstalled =~ ^%[3]s-* ]]; then
	uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
	curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
	uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%s
	curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5
done`
		ss = fmt.Sprintf(ss, utils.InstallOSConfigSUSE(), waitForRestartLinux, packageName, packageInstalled, packageNotInstalled)
		key = "startup-script"

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
		ss = `systemctl stop google-osconfig-agent
echo 'Adding test repo'
echo 'deb http://packages.cloud.google.com/apt osconfig-agent-test-repository main' >> /etc/apt/sources.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
   sleep 5
done
apt-get update
apt-get -y remove %[2]s || exit 1
apt-get -y install %[2]s=3.03+dfsg1-10 || exit 1
%[1]s
%[3]s
while true; do
  isinstalled=$(/usr/bin/dpkg-query -s %[2]s)
  if [[ $isinstalled =~ "Version: 3.03+dfsg1-10" ]]; then
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[4]s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  else
    uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[5]s
    curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
  fi
  sleep 5;
done`

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb(), packageName, waitForRestartLinux, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "yum":
		ss = `systemctl stop google-osconfig-agent
stop -q -n google-osconfig-agent  # required for EL6
echo 'Adding test repo'
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
%[1]s
%[3]s
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
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, waitForRestartLinux, packageInstalled, packageNotInstalled)
		key = "startup-script"

	case "googet":
		ss = `Stop-Service google_osconfig_agent
echo 'Adding test repo'
googet addrepo test https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository
googet -noconfirm remove %[2]s
googet -noconfirm install %[2]s.x86_64.0.1.0@1
%[1]s
%[3]s
while(1) {
  $installed_packages = googet installed %[2]s
  Write-Host $installed_packages
  if ($installed_packages -like "*0.1.0@1*") {
    $uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[4]s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  } else {
    $uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[5]s'
    Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body 1
  }
  sleep 5
}`
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet(), packageName, waitForRestartWin, packageInstalled, packageNotInstalled)
		key = "windows-startup-script-ps1"

	case "zypper":
		ss = `systemctl stop google-osconfig-agent
echo 'Adding test repo'
cat > /etc/zypp/repos.d/google-osconfig-agent.repo <<EOM
[test-repo]
name=test repo
baseurl=https://packages.cloud.google.com/yum/repos/osconfig-agent-test-repository
enabled=1
gpgcheck=0
EOM
zypper -n remove %[2]s
zypper -n --no-gpg-checks install %[2]s-3.03-2.fc7
%[1]s
%[3]s
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
		ss = fmt.Sprintf(ss, utils.InstallOSConfigSUSE(), packageName, waitForRestartLinux, packageInstalled, packageNotInstalled)
		key = "startup-script"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &computeApi.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}

func getRecipeInstallStartupScript(image, recipeName, pkgManager string) *computeApi.MetadataItems {
	scriptLinux := fmt.Sprintf(`
    # loop and check for recipedb entry
    while true; do
    isInstalled=$(grep '{"Name":"%[1]s","Version":\[0],"InstallTime":[0-9]*,"Success":true}' /var/lib/google/osconfig_recipedb)
    if [[ -n $isInstalled ]]; then
      uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[2]s
      curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
    else
      uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/%[3]s
      curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
    fi
    sleep 5
  done
    `, recipeName, packageInstalled, packageNotInstalled)

	var script string
	key := "startup-script"
	switch pkgManager {
	case "apt":
		script = fmt.Sprintf("%s\n%s\n%s", utils.InstallOSConfigDeb(), waitForRestartLinux, scriptLinux)
	case "yum":
		script = fmt.Sprintf("%s\n%s\n%s", yumStartupScripts[path.Base(image)], waitForRestartLinux, scriptLinux)
	case "zypper":
		script = fmt.Sprintf("%s\n%s\n%s", utils.InstallOSConfigSUSE(), waitForRestartLinux, scriptLinux)

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &computeApi.MetadataItems{
		Key:   key,
		Value: &script,
	}
}

func getRecipeStepsStartupScript(image, recipeName, pkgManager string) *computeApi.MetadataItems {
	scriptLinux := fmt.Sprintf(`
    # loop and check for recipedb entry
    function set_metadata {
      uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/$1
      curl -X PUT --data "1" $uri -H "Metadata-Flavor: Google"
    }
    while true; do
    while [[ ! -f /tmp/osconfig-script-test ]]; do
      set_metadata %[3]s
      sleep 5
    done
    echo "script step verified"
    while [[ ! -f /tmp/osconfig-exec-test ]]; do
      set_metadata %[3]s
      sleep 5
    done
    echo "exec step verified"
    while [[ ! -f /tmp/osconfig-copy-test ]]; do
      set_metadata %[3]s
      sleep 5
    done
    echo "copy step verified"
    while [[ ! -f /tmp/tar-test/tar/test.txt ]]; do
      set_metadata %[3]s
      sleep 5
    done
    echo "tar step verified"
    while [[ ! -f /tmp/zip-test/zip/test.txt ]]; do
      set_metadata %[3]s
      sleep 5
    done
    echo "zip step verified"

    isinstalled=$(grep '{"Name":"%[1]s","Version":\[0],"InstallTime":[0-9]*,"Success":true}' /tmp/var/lib/google/osconfig_recipedb)
    if [[ -n $isinstalled ]]
      set_metadata %[2]s
    else
    set_metadata %[3]s
    fi
    sleep 5
  done
    `, recipeName, packageInstalled, packageNotInstalled)

	var script string
	key := "startup-script"
	switch pkgManager {
	case "apt":
		script = fmt.Sprintf("%s\n\n%s", utils.InstallOSConfigDeb(), scriptLinux)
	case "yum":
		script = fmt.Sprintf("%s\n\n%s", yumStartupScripts[path.Base(image)], scriptLinux)
	case "zypper":
		script = fmt.Sprintf("%s\n\n%s", utils.InstallOSConfigSUSE(), scriptLinux)

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &computeApi.MetadataItems{
		Key:   key,
		Value: &script,
	}
}
