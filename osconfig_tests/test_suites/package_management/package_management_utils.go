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

// Package utils contains helper utils for osconfig_tests.

package packagemanagement

import (
	"fmt"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"github.com/google/logger"
	api "google.golang.org/api/compute/v1"
)

var (
	yumStartupScripts = map[string]string{
		"rhel-6":   utils.InstallOSConfigYumEL6,
		"rhel-7":   utils.InstallOSConfigYumEL7,
		"rhel-8":   utils.InstallOSConfigYumEL7,
		"centos-6": utils.InstallOSConfigYumEL6,
		"centos-7": utils.InstallOSConfigYumEL7,
	}
)

func getPackageInstallStartupScript(image, pkgManager, packageName string) *api.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/dpkg-query -s %s`\n" +
			"if [[ $isinstalled =~ \"Status: install ok installed\" ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5;\n" +
			"done;\n"

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "yum":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^%s-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "googet":
		ss = "%s\n" +
			"c:\\programdata\\Googet\\googet.exe addrepo osconfig-agent-test https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository\n" +
			"Restart-Service google_osconfig_agent\n" +
			"while(1) {\n" +
			"$installed_packages = googet installed\n" +
			"Write-Host $installed_packages\n" +
			"if ($installed_packages -like \"*%s*\") {\n" +
			"Write-Host \"%s\"\n" +
			"} else {\n" +
			"Write-Host \"%s\"\n" +
			"}\n" +
			"sleep 5\n" +
			"}\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet, packageName, packageInstalledString, packageNotInstalledString)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &api.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}

func getPackageRemovalStartupScript(image, pkgManager, packageName string) *api.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = "%s\n" +
			"n=0\n" +
			"while ! apt-get -y install %s; do\n" +
			"if [[ n -gt 3 ]]; then\n" +
			"echo \"could not install package\"\n" +
			"exit 1\n" +
			"fi\n" +
			"n=$[$n+1]\n" +
			"sleep 5\n" +
			"done\n" +
			"systemctl restart google-osconfig-agent\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"Error restarting google-osconfig-agent\"\n" +
			"exit 1\n" +
			"fi\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=\"$(/usr/bin/dpkg-query -s %s 2>&1 )\"\n" +
			"if [[ $isinstalled =~ \"package '%s' is not installed\" ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5;\n" +
			"done;\n"

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageName, packageName, packageNotInstalledString, packageInstalledString)
		key = "startup-script"

	case "yum":
		restartAgent := "systemctl restart google-osconfig-agent"
		if strings.HasSuffix(image, "-6") {
			restartAgent = "restart -q -n google-osconfig-agent"
		}
		ss = "%s\n" +
			"yum -y install %s\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"could not install package\"\n" +
			"exit 1\n" +
			"fi\n" +
			restartAgent + "\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"Error restarting google-osconfig-agent\"\n" +
			"exit 1\n" +
			"fi\n" + "while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^%s-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, packageName, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "googet":
		ss = "%s\n" +
			"$n = 0\n" +
			"while (1) {\n" +
			"googet -noconfirm install -sources https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository %s\n" +
			"if ($?) {\n" +
			"break\n" +
			"} else {\n" +
			"$n = $n + 1\n" +
			"if ($n -eq 3) {\n" +
			"exit 1\n" +
			"}}}\n" +
			"sleep 10\n" +
			"Restart-Service google_osconfig_agent\n" +
			"while(1) {\n" +
			"$installed_packages = googet installed\n" +
			"Write-Host $installed_packages\n" +
			"if ($installed_packages -like \"*%s*\") {\n" +
			"Write-Host %s\n" +
			"} else {\n" +
			"Write-Host %s\n" +
			"}\n" +
			"sleep 5\n" +
			"}\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet, packageName, packageName, packageInstalledString, packageNotInstalledString)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &api.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}

func getPackageInstallRemovalStartupScript(image, pkgManager, packageName string) *api.MetadataItems {
	var ss, key string

	switch pkgManager {
	case "apt":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=\"$(/usr/bin/dpkg-query -s %s 2>&1 )\"\n" +
			"if [[ $isinstalled =~ \"package '%s' is not installed\" ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5;\n" +
			"done;\n"

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageName, packageNotInstalledString, packageInstalledString)
		key = "startup-script"

	case "yum":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^%s-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "googet":
		ss = "%s\n" +
			"Restart-Service google_osconfig_agent\n" +
			"while(1) {\n" +
			"$installed_packages = googet installed\n" +
			"Write-Host $installed_packages\n" +
			"if ($installed_packages -like \"*%s*\") {\n" +
			"Write-Host %s\n" +
			"} else {\n" +
			"Write-Host %s\n" +
			"}\n" +
			"sleep 5\n" +
			"}\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet, packageName, packageInstalledString, packageNotInstalledString)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &api.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}

func getPackageInstallFromNewRepoTestStartupScript(image, pkgManager, packageName string) *api.MetadataItems {
	var ss, key string

	switch pkgManager {

	case "apt":
		ss = "%s\n" +
			"systemctl restart google-osconfig-agent\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/dpkg-query -s %s`\n" +
			"if [[ $isinstalled =~ \"Status: install ok installed\" ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5;\n" +
			"done;\n"

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "yum":
		restartAgent := "systemctl restart google-osconfig-agent"
		if strings.HasSuffix(image, "-6") {
			restartAgent = "restart -q -n google-osconfig-agent"
		}
		ss = "%s\n" +
			restartAgent + "\n" +
			"while true\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^%s-* ]]; then\n" +
			"echo \"%s\"\n" +
			"break\n" +
			"else\n" +
			"sleep 5\n" +
			"fi\n" +
			"done\n" +
			"echo \"%s\"\n"
		ss = fmt.Sprintf(ss, yumStartupScripts[path.Base(image)], packageName, packageName, packageInstalledString, packageNotInstalledString)
		key = "startup-script"

	case "googet":
		ss = "%s\n" +
			"Restart-Service google_osconfig_agent\n" +
			"while(1) {\n" +
			"$installed_packages = googet installed\n" +
			"Write-Host $installed_packages\n" +
			"if ($installed_packages -like \"*%s*\") {\n" +
			"Write-Host %s\n" +
			"} else {\n" +
			"Write-Host %s\n" +
			"}\n" +
			"sleep 5\n" +
			"}\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigGooGet, packageName, packageInstalledString, packageNotInstalledString)
		key = "windows-startup-script-ps1"

	default:
		logger.Errorf(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return &api.MetadataItems{
		Key:   key,
		Value: &ss,
	}
}
