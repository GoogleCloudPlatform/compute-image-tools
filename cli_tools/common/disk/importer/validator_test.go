//  Copyright 2020  Licensed under the Apache License, Version 2.0 (the "License");
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

package importer

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"testing"
)

func TestPreValidator(t *testing.T) {
	var cases = []struct {
		testName        string
		imageFromClient *compute.Image
		errorFromClient error
		expectedError   string
	}{
		{
			testName: "pass when image not found",
		},
		{
			testName:        "pass when image not found, even if error returned",
			errorFromClient: errors.New("image not found"),
		},
		{
			testName:        "fail when image already exists",
			imageFromClient: &compute.Image{},
			expectedError:   "The resource 'image-name' already exists. Please pick an image name that isn't already used.",
		},
	}

	project := "project-name"
	imageName := "image-name"
	for _, tt := range cases {
		t.Run(tt.testName, func(t *testing.T) {
			client := mockGetImageClient{
				t:                 t,
				expectedProject:   project,
				expectedImageName: imageName,
				img:               tt.imageFromClient,
				err:               tt.errorFromClient,
			}

			validator := newPreValidator(ImportArguments{
				Project:   project,
				ImageName: imageName,
			}, client)

			actualError := validator.validate()
			if tt.expectedError == "" {
				assert.NoError(t, actualError)
			} else {
				assert.EqualError(t, actualError, tt.expectedError)
			}
		})
	}
}

type mockGetImageClient struct {
	t                                  *testing.T
	expectedProject, expectedImageName string
	img                                *compute.Image
	err                                error
}

func (m mockGetImageClient) GetImage(project, name string) (*compute.Image, error) {
	assert.Equal(m.t, m.expectedImageName, name)
	assert.Equal(m.t, m.expectedProject, project)
	return m.img, m.err
}
