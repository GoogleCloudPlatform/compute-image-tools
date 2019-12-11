/*
Copyright 2019 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package packages

import "github.com/StackExchange/wmi"

type win32_QuickFixEngineering struct {
	Caption, Description, HotFixID, InstalledOn string
}

// QuickFixEngineering queries the wmi object win32_QuickFixEngineering for a list of installed updates.
func QuickFixEngineering() ([]QFEPackage, error) {
	var updts []win32_QuickFixEngineering
	DebugLogger.Print("Querying WMI for installed QuickFixEngineering updates.")
	if err := wmi.Query(wmi.CreateQuery(&updts, ""), &updts); err != nil {
		return nil, err
	}
	var qfe []QFEPackage
	for _, update := range updts {
		qfe = append(qfe, QFEPackage{
			Caption:     update.Caption,
			Description: update.Description,
			HotFixID:    update.HotFixID,
			InstalledOn: update.InstalledOn,
		})
	}
	return qfe, nil
}
