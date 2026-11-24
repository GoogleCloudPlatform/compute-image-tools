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

package storage

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

// TarGcsExtractor is responsible for extracting TAR archives from GCS to GCS
type TarGcsExtractor struct {
	ctx           context.Context
	storageClient domain.StorageClientInterface
	logger        logging.Logger
}

// NewTarGcsExtractor creates new TarGcsExtractor
func NewTarGcsExtractor(ctx context.Context, sc domain.StorageClientInterface, logger logging.Logger) *TarGcsExtractor {
	return &TarGcsExtractor{ctx: ctx, storageClient: sc, logger: logger}
}

// ExtractTarToGcs extracts a tar file in GCS back into GCS directory
func (tge *TarGcsExtractor) ExtractTarToGcs(tarGcsPath string, destinationGcsPath string) error {

	tarBucketName, tarPath, err := GetGCSObjectPathElements(tarGcsPath)
	if err != nil {
		return err
	}
	tarGcsReader, err := tge.storageClient.GetObject(tarBucketName, tarPath).NewReader()
	if err != nil {
		return daisy.Errf("error while opening archive %v: %v", tarGcsPath, err)
	}
	tarReader := tar.NewReader(tarGcsReader)
	defer tarGcsReader.Close()

	destinationBucketName, destinationPath, err := SplitGCSPath(destinationGcsPath)
	if err != nil {
		return daisy.Errf("invalid destination path: %v", destinationGcsPath)
	}

	for {
		header, err := tarReader.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it
		case header == nil:
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			return daisy.Errf("tar subdirectories not supported")

		case tar.TypeReg:
			destinationFilePath, err := safeDestinationFilePath(destinationPath, header.Name)
			if err != nil {
				return err
			}
			tge.logger.User(fmt.Sprintf("Extracting: %v to gs://%v", header.Name, path.Join(destinationBucketName, destinationFilePath)))

			if err := tge.storageClient.WriteToGCS(destinationBucketName, destinationFilePath, tarReader); err != nil {
				return err
			}
		}
	}
}

// safeDestinationFilePath returns the object path for tar member memberName
// under destinationPath. OVA packages are flat, and the OVF importer only
// resolves files at the top level of the extraction directory, so a member
// name is rejected if it is empty, absolute, contains a path separator, or
// refers to "." or "..". This prevents an archive from writing objects
// outside destinationPath (CWE-22).
func safeDestinationFilePath(destinationPath, memberName string) (string, error) {
	name := strings.TrimPrefix(memberName, "./")
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", daisy.Errf("invalid file name in OVA archive: %q", memberName)
	}
	destinationFilePath := path.Join(destinationPath, name)
	// Defense in depth: verify the object lands directly under destinationPath.
	if path.Dir(destinationFilePath) != path.Clean(destinationPath) {
		return "", daisy.Errf("invalid file name in OVA archive: %q", memberName)
	}
	return destinationFilePath, nil
}
