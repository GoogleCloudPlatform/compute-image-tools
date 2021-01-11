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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

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
