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

// During image import, we require that the scratch bucket is owned by
// the project that's performing the import. When it's owned by a different
// project, the import should be terminated, and the source file should be
// removed if it's a single file that's located in the scratch bucket.
//
// To run this, ensure your account has 'Storage Object Creator' permissions
// for the bucket `gs://bucket-owned-by-compute-image-test-resources`, which
// is in the project `compute-image-test-resources`.

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
	"github.com/GoogleCloudPlatform/compute-image-tools/common/runtime"
)

const (
	sharedBucket = "bucket-owned-by-compute-image-test-resources"
)

var (
	project       = runtime.GetConfig("GOOGLE_CLOUD_PROJECT", "project")
	zone          = runtime.GetConfig("GOOGLE_CLOUD_ZONE", "compute/zone")
	privateBucket = setupPrivateBucket(project)
)

func init() {
	if project == "compute-image-test-resources" {
		panic("Execute test using a project other than compute-image-test-resources.")
	}
}

func Test_DeleteSourceFile_WhenScratchOwnedByDifferentProject(t *testing.T) {
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
			imageName := "i" + uuid.New().String()
			tt.fileToImport.write(t, imageName)

			err := cli.Main([]string{
				"-image_name", imageName,
				"-data_disk",
				"-client_id", "e2e",
				"-source_file", tt.fileToImport.toURI(),
				"-scratch_bucket_gcs_path", tt.scratch.toURI(),
				"-project", project,
				"-zone", zone,
			})

			// The import should always fail if the scratch bucket
			// is owned by a different project.
			assert.Error(t, err)
			assert.Regexp(t, tt.expectedError, err.Error())

			// Remove -source_file when it's a single file
			// residing in the non-owned scratch bucket.
			fileDeleted := !tt.fileToImport.exists(t)
			if tt.deleteExpected && !fileDeleted {
				t.Errorf("Expected %s to be deleted, but it wasn't", tt.fileToImport.toURI())
			} else if !tt.deleteExpected && fileDeleted {
				t.Errorf("Unexpected deletion of %s", tt.fileToImport.toURI())
			}
		})
	}
}

func Test_DontDeleteSourceFile_WhenDirectory(t *testing.T) {

	for _, tt := range []struct {
		// Whether the directory passed to -source_file includes a trailing slash
		trailingSlash bool

		// Whether to create an empty file to represent the directory. See:
		//   https://stackoverflow.com/questions/38416598/
		makeDirObject bool

		expectedError string
	}{
		{
			trailingSlash: true,
			makeDirObject: true,
			expectedError: "cannot import an image from an empty file",
		},
		{
			trailingSlash: true,
			expectedError: "failed to read GCS file when validating resource file",
		},
		{
			makeDirObject: true,
			expectedError: "failed to read GCS file when validating resource file",
		},
		{
			expectedError: "failed to read GCS file when validating resource file",
		},
	} {
		caseName := fmt.Sprintf("dirObject=%v, trailingSlash=%v",
			tt.makeDirObject, tt.trailingSlash)
		t.Run(caseName, func(t *testing.T) {
			namespace := uuid.New().String()

			var objectsToCheck []gcsPath
			for i := 0; i < 2; i++ {
				fname := fmt.Sprintf("file_%d", i)
				file := gcsPath{sharedBucket, namespace, fname}
				file.write(t, fname)
				objectsToCheck = append(objectsToCheck, file)
			}

			sourceFileArg := gcsPath{sharedBucket, namespace, ""}.toURI()
			if tt.trailingSlash {
				sourceFileArg += "/"
			}
			if tt.makeDirObject {
				directory := gcsPath{sharedBucket, namespace, "/"}
				directory.write(t, "")
				objectsToCheck = append(objectsToCheck, directory)
			}

			err := cli.Main([]string{
				"-image_name", "i" + uuid.New().String(),
				"-data_disk",
				"-client_id", "e2e",
				"-source_file", sourceFileArg,
				"-scratch_bucket_gcs_path", "gs://" + sharedBucket,
				"-project", project,
				"-zone", zone,
			})

			// The import should always fail if the scratch bucket
			// is owned by a different project.
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)

			for _, child := range objectsToCheck {
				if !child.exists(t) {
					t.Errorf("Unexpected deletion of %s", child.toURI())
				}
			}
		})
	}
}

func Test_DontDeleteSourceFile_WhenAnotherFileHasSamePrefix(t *testing.T) {
	namespace := uuid.New().String()
	scratch := gcsPath{
		bucket: sharedBucket,
		dir:    namespace,
	}
	file1 := gcsPath{bucket: sharedBucket, dir: scratch.dir, file: "disk1.vmdk"}
	file1.write(t, "disk1 contents")
	file2 := gcsPath{bucket: sharedBucket, dir: scratch.dir, file: "disk1.vmdk.sha256"}
	file2.write(t, "ovf contents")

	err := cli.Main([]string{
		"-image_name", "i" + uuid.New().String(),
		"-data_disk",
		"-client_id", "e2e",
		"-source_file", file1.toURI(),
		"-scratch_bucket_gcs_path", scratch.toURI(),
		"-project", project,
		"-zone", zone,
	})

	assert.Error(t, err)

	for _, path := range []gcsPath{file1, file2} {
		if !path.exists(t) {
			t.Errorf("Contents of %s should have been retained.", path.toURI())
		}
	}
}

type gcsPath struct {
	bucket, dir, file string
}

func (path gcsPath) write(t *testing.T, content string) {
	ctx := context.Background()
	writer := path.openHandle(t, ctx).NewWriter(ctx)
	if _, err := fmt.Fprintf(writer, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func (path gcsPath) openHandle(t *testing.T, ctx context.Context) *storage.ObjectHandle {
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return client.Bucket(path.bucket).Object(path.dir + "/" + path.file)
}

func (path gcsPath) exists(t *testing.T) bool {
	ctx := context.Background()
	_, err := path.openHandle(t, ctx).NewReader(ctx)
	return err == nil
}

func (path gcsPath) toURI() string {
	uri := fmt.Sprintf("gs://%s/%s/", path.bucket, path.dir)
	if path.file != "" {
		uri += path.file
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
