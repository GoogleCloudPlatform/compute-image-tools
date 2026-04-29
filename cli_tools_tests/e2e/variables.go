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

package e2e

const (
	projectIDWithoutDefaultServiceAccountFlag                        = "project_id_without_default_service_account"
	projectIDWithoutDefaultServiceAccountPermissionFlag              = "project_id_without_default_service_account_permission"
	computeServiceAccountWithoutDefaultServiceAccountFlag            = "compute_service_account_without_default_service_account"
	computeServiceAccountWithoutDefaultServiceAccountPermissionFlag  = "compute_service_account_without_default_service_account_permission"
	instanceServiceAccountWithoutDefaultServiceAccountFlag           = "instance_service_account_without_default_service_account"
	instanceServiceAccountWithoutDefaultServiceAccountPermissionFlag = "instance_service_account_without_default_service_account_permission"
)

// ServiceAccountTestVariables contains service-account related test variables.
type ServiceAccountTestVariables struct {
	ProjectID              string
	ComputeServiceAccount  string
	InstanceServiceAccount string
}

// GetServiceAccountTestVariables extract extra test variables related to service account from input variable map.
func GetServiceAccountTestVariables(argMap map[string]string, isDefaultServiceAccountDisabled bool) (v ServiceAccountTestVariables, ok bool) {
	for key, val := range argMap {
		if isDefaultServiceAccountDisabled {
			switch key {
			case projectIDWithoutDefaultServiceAccountFlag:
				v.ProjectID = val
			case computeServiceAccountWithoutDefaultServiceAccountFlag:
				v.ComputeServiceAccount = val
			case instanceServiceAccountWithoutDefaultServiceAccountFlag:
				v.InstanceServiceAccount = val
			default:
				// args not related
			}
		} else {
			switch key {
			case projectIDWithoutDefaultServiceAccountPermissionFlag:
				v.ProjectID = val
			case computeServiceAccountWithoutDefaultServiceAccountPermissionFlag:
				v.ComputeServiceAccount = val
			case instanceServiceAccountWithoutDefaultServiceAccountPermissionFlag:
				v.InstanceServiceAccount = val
			default:
				// args not related
			}
		}
	}
	ok = v.ProjectID != "" && v.ComputeServiceAccount != ""
	return
}
