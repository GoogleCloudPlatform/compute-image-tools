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

package importer

import (
	"compress/gzip"
	"net/url"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// A resource that can be imported to GCE disk images. If an instance of this
// interface exists, it is expected that validation has already occurred,
// and that the caller can safely use the resource.
type resource interface {
	path() string
}

// initAndValidateSource takes the sourceFile and sourceImage specified by the user
// and determines which, if any, is importable. It is an error if both sourceFile and
// sourceImage are specified.
func initAndValidateSource(sourceFile, sourceImage string,
	storageClient domain.StorageClientInterface) (resource, error) {
	sourceFile = strings.TrimSpace(sourceFile)
	sourceImage = strings.TrimSpace(sourceImage)

	if sourceFile == "" && sourceImage == "" {
		return nil, daisy.Errf(
			"either -source_file or -source_image has to be specified")
	} else if sourceFile != "" && sourceImage != "" {
		return nil, daisy.Errf(
			"either -source_file or -source_image has to be specified, but not both %v %v",
			sourceFile, sourceImage)
	}

	if sourceFile != "" {
		return newFileSource(sourceFile, storageClient)
	}

	return newImageSource(sourceImage)
}

// Whether the resource is a file in GCS.
func isFile(s resource) bool {
	_, ok := s.(fileSource)
	return ok
}

// Whether the resource is a GCE image.
func isImage(s resource) bool {
	_, ok := s.(imageSource)
	return ok
}

// An importable source backed by a GCS object.
type fileSource struct {
	gcsPath string
	bucket  string
	object  string
}

// Create a fileSource from a gcsPath to a disk image. This method uses storageClient
// to read a few bytes from the file. It is an error if the file is empty, or if
// the file is compressed with gzip.
func newFileSource(gcsPath string, storageClient domain.StorageClientInterface) (resource, error) {
	sourceBucketName, sourceObjectName, err := storage.GetGCSObjectPathElements(gcsPath)
	if err != nil {
		return nil, err
	}
	source := fileSource{
		gcsPath: gcsPath,
		bucket:  sourceBucketName,
		object:  sourceObjectName,
	}
	return source, source.validate(storageClient)
}

// The resource path for fileSource is its GCS path.
func (s fileSource) path() string {
	return s.gcsPath
}

// Performs basic validation, focusing on error cases that we've seen in the past.
// This reads a few bytes from the file in GCS. It is an error if the file
// is empty, or if the file is compressed with gzip.
func (s fileSource) validate(storageClient domain.StorageClientInterface) error {
	rc, err := storageClient.GetObjectReader(s.bucket, s.object)
	if err != nil {
		return daisy.Errf("failed to read GCS file when validating resource file: unable to open "+
			"file from bucket %q, file %q: %v", s.bucket, s.object, err)
	}
	defer rc.Close()

	byteCountingReader := daisycommon.NewByteCountingReader(rc)
	// Detect whether it's a compressed file by extracting compressed file header
	if _, err = gzip.NewReader(byteCountingReader); err == nil {
		return daisy.Errf("the input file is a gzip file, which is not supported by " +
			"image import. To import a file that was exported from Google Compute " +
			"Engine, please use image create. To import a file that was exported " +
			"from a different system, decompress it and run image import on the " +
			"disk image file directly")
	}

	// By calling gzip.NewReader above, a few bytes were read from the Reader in
	// an attempt to decode the compression header. If the Reader represents
	// an empty file, then BytesRead will be zero.
	if byteCountingReader.BytesRead <= 0 {
		return daisy.Errf("cannot import an image from an empty file")
	}

	return nil
}

// An importable source backed by a GCE disk image.
type imageSource struct {
	uri string
}

// Creates an imageSource from a reference to a GCE disk image. The syntax of the
// reference is validated, but no I/O is performed to determine whether the image
// exists or whether the calling user has access to it.
func newImageSource(imagePath string) (resource, error) {
	source := imageSource{
		uri: param.GetGlobalResourcePath("images", imagePath),
	}
	return source, source.validate()
}

var imageNamePattern = regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")

// Performs basic validation, focusing on error cases that
// we've seen in the past.
//
// Two key offenders:
//   1. Using a file path instead of an image name.
//     Examples:
//        - file://home/user/image.vmdk
//        - gs://bucket/image.vmdk
//        - gs://global/images/image
//        - https://storage.googleapis.com/bucket/image
//   2. Using a relative file path instead of an image name.
//     Examples:
//        - image.vmdk
//        - path/to/image.vmdk
//
// This method does not validate whether the image exist, or whether
// the calling user has access to it.
func (s imageSource) validate() error {
	parsed, err := url.Parse(s.uri)
	if err != nil {
		return err
	}
	if parsed.Scheme != "" {
		return daisy.Errf(
			"invalid image reference %q.", s.uri)
	}

	var imageName string
	if slash := strings.LastIndex(s.uri, "/"); slash > -1 {
		imageName = s.uri[slash+1:]
	} else {
		imageName = s.uri
	}
	if imageName == "" || len(imageName) > 63 {
		return daisy.Errf(
			"invalid image name %q. Image name must be 1-63 characters long, inclusive", imageName)
	}
	if !imageNamePattern.MatchString(imageName) {
		return daisy.Errf(
			"invalid image name %q. The first character must be a lowercase letter, and all "+
				"following characters must be a dash, lowercase letter, or digit, except the last "+
				"character, which cannot be a dash.", imageName)
	}
	return nil
}

// The path to an imageSource is a fully-qualified global GCP resource path.
func (s imageSource) path() string {
	return s.uri
}
