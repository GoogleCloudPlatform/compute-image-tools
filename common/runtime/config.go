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

package runtime

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// GetConfig reads environment-based configurations, using fallback.
// First envKey is looked up as an nvironment variable. If that's
// not found, gcloud is queried using `gcloud config get-value $gcloudConfig`.
//
// Examples:
//
//	GetConfig("GOOGLE_CLOUD_PROJECT", "project")
//	GetConfig("GOOGLE_CLOUD_ZONE", "compute/zone")
//
// For more info on using configurations with gcloud, see:
//
//	https://cloud.google.com/sdk/docs/configurations
func GetConfig(envKey, gcloudConfig string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}

	out, err := exec.Command("gcloud", "config", "get-value", gcloudConfig).Output()
	if err != nil {
		log.Panicf("Environment variable $%s is required", envKey)
	}
	return strings.TrimSpace(string(out))
}
