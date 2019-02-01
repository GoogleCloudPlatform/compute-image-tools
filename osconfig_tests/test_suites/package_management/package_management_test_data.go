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

const (
	packageInstallTestOsConfigString string = `{
	"name": "packageinstalltest",
	"description": "test osconfig to test package installation",
	"apt": {
		"package_installs": [{
			"name": "cowsay"
		}]
	}
}`

	packageInstallTestAssignmentString string = `{
	"name": "packageinstalltest",
	"description": "test assignment to test package installation",
	"os_configs": [
		"projects/281997379984/osConfigs/packageinstalltest"
	],
	"expression": "instance.name==\"osconfig-test-debian-9-packageinstalltest\""
}`

	packageRemovalTestOsConfigString string = `{
	"name": "packageremovaltest",
	"description": "test osconfig to test package removal",
	"apt": {
		"package_removals": [{
			"name": "wget"
		}]
	}
}`

	packageRemovalTestAssignmentString string = `{
	"name": "packageinstalltest",
	"description": "test assignment to test package installation",
	"os_configs": [
		"projects/281997379984/osConfigs/packageremovaltest"
	],
	"expression": "instance.name==\"osconfig-test-debian-9-packageremovaltest\""
}`

	packageInstallRemoveTestOsConfigString string = `{
	"name": "packageinstallremovetest",
	"description": "test osconfig to test package removal supersides installation",
	"apt": {
		"package_installs" : [{
			"name": "cowsay"
		}],
		"package_removals": [{
			"name": "cowsay"
		}]
	}
}`

	packageInstallRemoveTestAssignmentString string = `{
	"name": "packageinstallremovetest",
	"description": "test assignment to test package installation",
	"os_configs": [
		"projects/281997379984/osConfigs/packageinstallremovetest"
	],
	"expression": "instance.name==\"osconfig-test-debian-9-packageinstallremovetest\""
}`
)
