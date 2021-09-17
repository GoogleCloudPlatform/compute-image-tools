//  Copyright 2021 Google Inc. All Rights Reserved.
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

package precheck

import (
	"testing"

	"github.com/GoogleCloudPlatform/osconfig/osinfo"
	"github.com/stretchr/testify/assert"
)

func Test_osVersionCheck(t *testing.T) {
	tests := []struct {
		name        string
		osInfo      *osinfo.OSInfo
		expectedLog string
		expectFail  bool
	}{
		{
			name: "Linux happy case - no arch",
			osInfo: &osinfo.OSInfo{
				ShortName: "ubuntu",
				Version:   "14.04",
			},
			expectedLog: "Detected system: ubuntu-1404",
		}, {
			name: "Linux happy case - with arch",
			osInfo: &osinfo.OSInfo{
				ShortName:    "ubuntu",
				Version:      "14.04",
				Architecture: "x86_64",
			},
			expectedLog: "Detected system: ubuntu-1404",
		}, {
			name: "Windows happy case - no arch",
			osInfo: &osinfo.OSInfo{
				ShortName: "windows",
				Version:   "10",
			},
			expectedLog: "Detected Windows version number: NT 10",
		}, {
			name: "Windows happy case - with arch",
			osInfo: &osinfo.OSInfo{
				ShortName:    "windows",
				Version:      "6.3",
				Architecture: "x86_32",
			},
			expectedLog: "Detected Windows version number: NT 6.3",
		}, {
			name: "Windows supports NT versions",
			osInfo: &osinfo.OSInfo{
				ShortName:    "windows",
				Version:      "6.1.1234",
				Architecture: "x86_32",
			},
			expectedLog: "Detected Windows version number: NT 6.1.1234",
		}, {
			name: "Fail when version is not supported for the distro",
			osInfo: &osinfo.OSInfo{
				ShortName: "ubuntu",
				Version:   "10.04",
			},
			expectedLog: "ubuntu-1004 is not supported for import",
			expectFail:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, e := (&OSVersionCheck{OSInfo: tt.osInfo}).Run()
			assert.Nil(t, e, "nil is always returned")
			assert.Regexp(t, tt.expectedLog, r)
			assert.Equal(t, tt.expectFail, r.Failed())
		})
	}
}

func Test_osVersionCheck_skipWhenOSDetectionFails(t *testing.T) {
	tests := []struct {
		name        string
		osInfo      *osinfo.OSInfo
		expectedLog string
	}{
		{
			name: "skip when shortName is unknown distro",
			osInfo: &osinfo.OSInfo{
				ShortName: "distro-no-recognized",
				Version:   "10",
			},
			expectedLog: "Unrecognized distro `distro-no-recognized`",
		}, {
			name: "skip when arch is unknown",
			osInfo: &osinfo.OSInfo{
				ShortName:    "ubuntu",
				Version:      "14.04",
				Architecture: "mips",
			},
			expectedLog: "Unrecognized architecture `mips`",
		}, {
			name: "Skip when OS config returns 'linux'; don't emit message saying that linux isn't supported",
			osInfo: &osinfo.OSInfo{
				ShortName: "linux",
			},
			expectedLog: "Detected generic Linux system",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, e := (&OSVersionCheck{OSInfo: tt.osInfo}).Run()
			assert.Nil(t, e, "nil is always returned")
			assert.Contains(t, r.String(), tt.expectedLog)
			assert.Contains(t, r.String(), "Unable to determine whether your system is supported for import. "+
				"For supported versions, see https://cloud.google.com/sdk/gcloud/reference/compute/images/import")
			assert.Equal(t, Skipped, r.result)
			t.Logf("\n%s", r)
		})
	}
}

func Test_splitOSVersion(t *testing.T) {
	tests := []struct {
		version       string
		expectedMajor string
		expectedMinor string
	}{
		{
			version:       "",
			expectedMajor: "",
			expectedMinor: "",
		}, {
			version:       "vista",
			expectedMajor: "vista",
			expectedMinor: "",
		}, {
			version:       "14.04",
			expectedMajor: "14",
			expectedMinor: "04",
		}, {
			version:       "10.0",
			expectedMajor: "10",
			expectedMinor: "0",
		}, {
			version:       "6",
			expectedMajor: "6",
			expectedMinor: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			actualMajor, actualMinor := splitOSVersion(tt.version)
			assert.Equal(t, tt.expectedMajor, actualMajor)
			assert.Equal(t, tt.expectedMinor, actualMinor)
		})
	}
}
