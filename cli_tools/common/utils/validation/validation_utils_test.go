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

package validation

import (
	"strings"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/stretchr/testify/assert"
)

func TestValidateOneOfStringFlagNotEmpty(t *testing.T) {
	if err := ValidateExactlyOneOfStringFlagNotEmpty(map[string]string{
		"source_image":         "anImage",
		"source_disk_snapshot": "",
	}); err != nil {
		t.Errorf("error should be nil, but got %v", err)
	}
	if err := ValidateExactlyOneOfStringFlagNotEmpty(map[string]string{
		"source_image":         "",
		"source_disk_snapshot": "aSnapshot",
	}); err != nil {
		t.Errorf("error should be nil, but got %v", err)
	}
	if err := ValidateExactlyOneOfStringFlagNotEmpty(map[string]string{
		"source_image":         "",
		"source_disk_snapshot": "",
	}); err == nil || !strings.HasPrefix(err.Error(), "Exactly one of") {
		t.Errorf("error should be 'Exactly one of ...', but got %v", err)
	}
	if err := ValidateExactlyOneOfStringFlagNotEmpty(map[string]string{
		"source_image":         "anImage",
		"source_disk_snapshot": "aSnapshot",
	}); err == nil || !strings.HasPrefix(err.Error(), "Exactly one of") {
		t.Errorf("error should be 'Exactly one of ...', but got %v", err)
	}
}

func TestValidateFqdnValidValue(t *testing.T) {
	err := ValidateFqdn("host.domain", "hostname")
	if err != nil {
		t.Errorf("error should be nil, but got %v", err)
	}
}

func TestValidateFqdnSingleLabel(t *testing.T) {
	err := ValidateFqdn("host", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnEmptyString(t *testing.T) {
	err := ValidateFqdn("", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnInvalidChars(t *testing.T) {
	err := ValidateFqdn("host|name.domain", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateFqdnTooLong(t *testing.T) {
	err := ValidateFqdn("host.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain.domain", "hostname")
	if err == nil {
		t.Error("error expected")
	}
}

func TestValidateImageName_ExpectValid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/images
	for _, imgName := range []string{
		"dashes-allowed-inside",
		"a", // min length is 1
		"o-----equal-to-max-63-----------------------------------------o",
	} {
		t.Run(imgName, func(t *testing.T) {
			assert.NoError(t, ValidateImageName(imgName))
		})
	}
}

func TestValidateImageName_ExpectInvalid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/images
	for _, imgName := range []string{
		"-no-starting-dash",
		"no-ending-dash-",
		"dont/allow/slashes",
		"DontAllowCaps",
		"o-----longer-than-max-63---------------------------------------o",
	} {
		t.Run(imgName, func(t *testing.T) {
			err := ValidateImageName(imgName)
			assert.NotNil(t, err)
			assert.Regexp(t, "Image name .* must conform to https://cloud.google.com/compute/docs/reference/rest/v1/images", err)
			assert.Contains(t, err.Error(), imgName, "Raw error should include image's name")
			realError := err.(daisy.DError)
			for _, anonymizedErrs := range realError.AnonymizedErrs() {
				assert.NotContains(t, anonymizedErrs, imgName,
					"Anonymized error should not contain image's name")
			}
		})
	}
}

func TestValidateImageURI_ExpectValid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/images
	for _, testCase := range [][]string{
		{"projects/a-project/global/images/dashes-allowed-inside", "a-project", "dashes-allowed-inside"},
		{"projects/google.com:google-project/global/images/a", "google.com:google-project", "a"},
		{"projects/long-project-name-less-than-28/global/images/o-----equal-to-max-63-----------------------------------------o", "long-project-name-less-than-28", "o-----equal-to-max-63-----------------------------------------o"},
	} {
		t.Run(testCase[0], func(t *testing.T) {
			project, imageName, err := ValidateImageURI(testCase[0])
			assert.NoError(t, err)
			assert.Equal(t, testCase[1], project)
			assert.Equal(t, testCase[2], imageName)
		})
	}
}

func TestValidateImageURI_ExpectInvalid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/images
	for _, imageURI := range []string{
		"not-a-uri",
		"projects/a-project/global/images/-no-starting-dash",
		"projects/a-project/global/images/no-ending-dash-",
		"projects/a-project/global/images/dont/allow/slashes/in/image/name",
		"projects/a-project/global/images/DontAllowCaps",
		"projects/a-project/global/images/o-----longer-than-max-63---------------------------------------o",
		"abcde/a-project/global/images/image", // shorter than min length of 6
		"-no-leading-dash/a-project/global/images/image",
		"no-ending-dash-/a-project/global/images/image",
		"1-no-leading-numbers/a-project/global/images/image",
		"DontAllowCaps/a-project/global/images/image",
		"notgoogle.com:test-project/a-project/global/images/image",
		"o-----longer-than-max-30------o/a-project/global/images/image",
	} {
		t.Run(imageURI, func(t *testing.T) {
			imageName, project, err := ValidateImageURI(imageURI)

			assert.Empty(t, imageName)
			assert.Empty(t, project)

			assert.NotNil(t, err)
			assert.Regexp(t, "Image URI .* must conform to https://cloud.google.com/compute/docs/reference/rest/v1/images", err)
			assert.Contains(t, err.Error(), imageURI, "Raw error should include image's URI")
			realError := err.(daisy.DError)
			for _, anonymizedErrs := range realError.AnonymizedErrs() {
				assert.NotContains(t, anonymizedErrs, imageURI,
					"Anonymized error should not contain image's name")
			}

		})
	}
}
func TestValidateDiskSnapshotName_ExpectValid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/snapshot
	for _, snapshotName := range []string{
		"dashes-allowed-inside",
		"a", // min length is 1
		"o-----equal-to-max-63-----------------------------------------o",
	} {
		t.Run(snapshotName, func(t *testing.T) {
			assert.NoError(t, ValidateSnapshotName(snapshotName))
		})
	}
}

func TestValidateDiskSnapshotName_ExpectInvalid(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/snapshots
	for _, snapshotName := range []string{
		"-no-starting-dash",
		"no-ending-dash-",
		"dont/allow/slashes",
		"DontAllowCaps",
		"o-----longer-than-max-63---------------------------------------o",
	} {
		t.Run(snapshotName, func(t *testing.T) {
			err := ValidateSnapshotName(snapshotName)
			assert.NotNil(t, err)
			assert.Regexp(t, "Snapshot name .* must conform to https://cloud.google.com/compute/docs/reference/rest/v1/snapshots", err)
			assert.Contains(t, err.Error(), snapshotName, "Raw error should include snapshot's name")
			realError := err.(daisy.DError)
			for _, anonymizedErrs := range realError.AnonymizedErrs() {
				assert.NotContains(t, anonymizedErrs, snapshotName,
					"Anonymized error should not contain snapshot's name")
			}
		})
	}
}

func TestValidateProjectID_ExpectValid(t *testing.T) {
	for _, projectID := range []string{
		"abcdef", // equal to min length of 6
		"dashes-allowed-inside",
		"ending-numbers-allowed-1",
		"a1-inside-numbers-allowed",
		"google.com:test-project",
		"o-----equal-to-max-30--------o",
	} {
		t.Run(projectID, func(t *testing.T) {
			assert.NoError(t, ValidateProjectID(projectID))
		})
	}
}

func TestValidateProjectID_ExpectInvalid(t *testing.T) {
	for _, projectID := range []string{
		"abcde", // shorter than min length of 6
		"-no-leading-dash",
		"no-ending-dash-",
		"1-no-leading-numbers",
		"DontAllowCaps",
		"joonix.com:test-project",
		"o-----longer-than-max-30------o",
	} {
		t.Run(projectID, func(t *testing.T) {
			err := ValidateProjectID(projectID)
			assert.NotNil(t, err)
			assert.Regexp(t, "projectID .* must conform to https://cloud.google.com/resource-manager/reference/rest/v1/projects", err)
			assert.Contains(t, err.Error(), projectID, "Raw error should include projectID")
			realError := err.(daisy.DError)
			for _, anonymizedErrs := range realError.AnonymizedErrs() {
				assert.NotContains(t, anonymizedErrs, projectID,
					"Anonymized error should not contain projectID")
			}
		})
	}
}

func TestValidateStruct_SupportsCustomFieldNames(t *testing.T) {
	type User struct {
		Firstname string `name:"first_name" validate:"required"`
	}

	assert.EqualError(t, ValidateStruct(User{}), "first_name has to be specified")
}

func TestValidateStruct_UsesFieldStructNameByDefault(t *testing.T) {
	type User struct {
		Name string `validate:"required"`
	}

	assert.EqualError(t, ValidateStruct(User{}), "Name has to be specified")
}

func TestValidateStruct_SupportsImageNameValidation(t *testing.T) {
	type Disk struct {
		ImageName string `validate:"gce_disk_image_name"`
	}

	d := Disk{"uri/disk/path"}
	assert.Equal(t, ValidateImageName(d.ImageName), ValidateStruct(d))
}
