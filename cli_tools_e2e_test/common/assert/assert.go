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

// Package assert contains e2e tests assertion functions
package assert

import (
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
	"google.golang.org/api/compute/v1"
)

// GuestOSFeatures asserts expected & unexpected guestOSFeatures.
func GuestOSFeatures(expectedGuestOsFeatures []string, unexpectedGuestOsFeatures []string, actualGuestOSFeatures []*compute.GuestOsFeature, junit *junitxml.TestCase, logger *log.Logger) {
	guestOsFeatures := make([]string, 0, len(actualGuestOSFeatures))
	for _, f := range actualGuestOSFeatures {
		guestOsFeatures = append(guestOsFeatures, f.Type)
	}

	if expectedGuestOsFeatures != nil {
		if !utils.ContainsAll(guestOsFeatures, expectedGuestOsFeatures) {
			junit.WriteFailure("GuestOsFeatures expect: %v, actual: %v", strings.Join(expectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
			logger.Printf("GuestOsFeatures expect: %v, actual: %v", strings.Join(expectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
		}
	}

	if unexpectedGuestOsFeatures != nil {
		if utils.ContainsAny(guestOsFeatures, unexpectedGuestOsFeatures) {
			junit.WriteFailure("GuestOsFeatures unexpect: %v, actual: %v", strings.Join(unexpectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
			logger.Printf("GuestOsFeatures unexpect: %v, actual: %v", strings.Join(unexpectedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ","))
		}
	}
}
