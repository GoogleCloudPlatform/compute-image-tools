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

package import_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/cli"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common/config"
)

const (
	// For this test to work, it needs to be run within
	// a project *other* than compute-image-test-resources.
	sharedBucket = "bucket-owned-by-compute-image-test-resources"
)

var (
	project       = config.GetConfig("GOOGLE_CLOUD_PROJECT", "project")
	zone          = config.GetConfig("GOOGLE_CLOUD_ZONE", "compute/zone")
	privateBucket = setupPrivateBucket(project)
)

func Test_ImportStops_WhenScratchOwnedByDifferentProject(t *testing.T) {
	namespace := uuid.New().String()
	for _, tt := range []struct {
		caseName       string
		deleteExpected bool
		fileToImport   gcsPath
		scratch        gcsPath
		expectedError  *regexp.Regexp
	}{
		{
			caseName:       "In scratch",
			deleteExpected: true,
			expectedError:  regexp.MustCompile("Scratch bucket.* is not in project.* Deleted"),
			fileToImport: gcsPath{
				bucket: sharedBucket,
				dir:    namespace + "-in-scratch",
				file:   "file.vmdk",
			},
			scratch: gcsPath{
				bucket: sharedBucket,
				dir:    namespace + "-in-scratch",
			},
		},
		{
			caseName:       "Not in scratch",
			deleteExpected: false,
			expectedError:  regexp.MustCompile("Scratch bucket.* is not in project"),
			fileToImport: gcsPath{
				bucket: privateBucket,
				dir:    namespace + "-not-in-scratch",
				file:   "file.vmdk",
			},
			scratch: gcsPath{
				bucket: sharedBucket,
				dir:    namespace + "-not-in-scratch",
			},
		},
	} {
		t.Run(tt.caseName, func(t *testing.T) {
			// 1. Write to `fileToImport`.
			ctx := context.Background()
			client, err := storage.NewClient(ctx)
			if err != nil {
				t.Fatal(err)
			}
			obj := client.Bucket(tt.fileToImport.bucket).Object(
				tt.fileToImport.dir + "/" + tt.fileToImport.file)
			writer := obj.NewWriter(ctx)
			if _, err := fmt.Fprintf(writer, "content to be imported"); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}

			// 2. Run the import
			err = cli.Main([]string{
				"-image_name", "i" + uuid.New().String(),
				"-data_disk",
				"-client_id", "e2e",
				"-source_file", tt.fileToImport.toURI(),
				"-scratch_bucket_gcs_path", tt.scratch.toURI(),
				"-project", project,
				"-zone", zone,
			})

			// 3. The import must fail
			assert.Error(t, err)
			assert.Regexp(t, tt.expectedError, err.Error())

			// 4. Only delete the file if it was in the scratch bucket
			_, err = obj.NewReader(ctx)
			if tt.deleteExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "object doesn't exist")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type gcsPath struct {
	bucket, dir, file string
}

func (p gcsPath) toURI() string {
	uri := fmt.Sprintf("gs://%s/%s/", p.bucket, p.dir)
	if p.file != "" {
		uri += p.file
	}
	return uri
}

// setupPrivateBucket makes a bucket within the current project.
// Naming is deterministic, and it's fine to re-use between tests.
func setupPrivateBucket(project string) string {
	bucketName := "compute-image-tools-e2e-" + project
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	if err := client.Bucket(bucketName).Create(ctx, project, nil); err != nil {
		realError := err.(*googleapi.Error)
		// Code 409 indicates the bucket already exists:
		// https://cloud.google.com/storage/docs/troubleshooting#bucket-name-conflict
		if realError.Code != 409 {
			panic(err)
		}
	}
	return bucketName
}
