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

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils/junitxml"
)

// GuestOSFeatures asserts expected & unexpected guestOSFeatures.
func GuestOSFeatures(requiredGuestOsFeatures []string, notAllowedGuestOsFeatures []string, actualGuestOSFeatures []*compute.GuestOsFeature, junit *junitxml.TestCase, logger *log.Logger) {
	guestOsFeatures := make([]string, 0, len(actualGuestOSFeatures))
	for _, f := range actualGuestOSFeatures {
		guestOsFeatures = append(guestOsFeatures, f.Type)
	}

	if requiredGuestOsFeatures != nil {
		ContainsAll(guestOsFeatures, requiredGuestOsFeatures, junit, logger,
			fmt.Sprintf("GuestOsFeatures expect: %v, actual: %v", strings.Join(requiredGuestOsFeatures, ","), strings.Join(guestOsFeatures, ",")))
	}

	if notAllowedGuestOsFeatures != nil {
		ContainsNone(guestOsFeatures, notAllowedGuestOsFeatures, junit, logger,
			fmt.Sprintf("GuestOsFeatures unexpect: %v, actual: %v", strings.Join(notAllowedGuestOsFeatures, ","), strings.Join(guestOsFeatures, ",")))
	}
}

// ContainsAll asserts all given strings in subarr exists in arr
func ContainsAll(arr []string, subarr []string, junit *junitxml.TestCase, logger *log.Logger, failureMessage string) {
	if !containsAll(arr, subarr) {
		Failure(junit, logger, failureMessage)
	}
}

// containsAll checks whether all given strings in subarr exists in arr
func containsAll(arr []string, subarr []string) bool {
	for _, item := range subarr {
		exists := false
		for _, i := range arr {
			if item == i {
				exists = true
				break
			}
		}
		if !exists {
			return false
		}
	}
	return true
}

// ContainsNone asserts some given strings in subarr exists in arr
func ContainsNone(arr []string, subarr []string, junit *junitxml.TestCase, logger *log.Logger, failureMessage string) {
	if containsAny(arr, subarr) {
		Failure(junit, logger, failureMessage)
	}
}

// containsAny checks whether any given strings in subarr exists in arr
func containsAny(arr []string, subarr []string) bool {
	for _, item := range subarr {
		for _, i := range arr {
			if item == i {
				return true
			}
		}
	}
	return false
}
