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

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
)

func getPackageInstallStartupScript(pkgManager, packageName string) string {
	var ss string

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

	case "yum":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^cowsay-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigYumEL7, packageName, packageInstalledString, packageNotInstalledString)

	default:
		panic(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return ss
}

func getPackageRemovalStartupScript(pkgManager, packageName string) string {
	var ss string

	switch pkgManager {
	case "apt":
		ss = "%s\n" +
			"sudo apt-get -y install %s\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"could not install package\"\n" +
			"exit 1\n" +
			"fi\n" +
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

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageNotInstalledString, packageInstalledString)

	case "yum":
		ss = "%s\n" +
			"sudo yum -y install %s\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"could not install package\"\n" +
			"exit 1\n" +
			"fi\n" +
			"systemctl restart google-osconfig-agent\n" +
			"if [[ $? != 0 ]]; then\n" +
			"echo \"Error restarting google-osconfig-agent\"\n" +
			"exit 1\n" +
			"fi\n" + "while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^cowsay-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigYumEL7, packageName, packageName, packageInstalledString, packageNotInstalledString)

	default:
		panic(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return ss
}

func getPackageInstallRemovalStartupScript(pkgManager, packageName string) string {
	var ss string

	switch pkgManager {
	case "apt":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/dpkg-query -s %s`\n" +
			"if [[ $isinstalled =~ \"package '%s' is not installed\" ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5;\n" +
			"done;\n"

		ss = fmt.Sprintf(ss, utils.InstallOSConfigDeb, packageName, packageName, packageNotInstalledString, packageInstalledString)

	case "yum":
		ss = "%s\n" +
			"while true;\n" +
			"do\n" +
			"isinstalled=`/usr/bin/rpmquery -a %s`\n" +
			"if [[ $isinstalled =~ ^cowsay-* ]]; then\n" +
			"echo \"%s\"\n" +
			"else\n" +
			"echo \"%s\"\n" +
			"fi\n" +
			"sleep 5\n" +
			"done\n"
		ss = fmt.Sprintf(ss, utils.InstallOSConfigYumEL7, packageName, packageName, packageInstalledString, packageNotInstalledString)

	default:
		panic(fmt.Sprintf("invalid package manager: %s", pkgManager))
	}

	return ss
}
