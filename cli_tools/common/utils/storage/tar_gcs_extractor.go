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

package storageutils

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"

	"io"
	"path"
)

// TarGcsExtractor is responsible for extracting TAR archives from GCS to GCS
type TarGcsExtractor struct {
	ctx           context.Context
	storageClient commondomain.StorageClientInterface
	logger        logging.LoggerInterface
}

// NewTarGcsExtractor creates new TarGcsExtractor
func NewTarGcsExtractor(ctx context.Context, sc commondomain.StorageClientInterface, logger logging.LoggerInterface) *TarGcsExtractor {
	return &TarGcsExtractor{ctx: ctx, storageClient: sc, logger: logger}
}

// ExtractTarToGcs extracts a tar file in GCS back into GCS directory
func (tge *TarGcsExtractor) ExtractTarToGcs(tarGcsPath string, destinationGcsPath string) error {

	tarBucketName, tarPath, err := SplitGCSPath(tarGcsPath)
	if err != nil {
		return err
	}
	tarGcsReader, err := tge.storageClient.GetObjectReader(tarBucketName, tarPath)
	if err != nil {
		return fmt.Errorf("error while opening archive %v: %v", tarGcsPath, err)
	}
	tarReader := tar.NewReader(tarGcsReader)
	defer tarGcsReader.Close()

	destinationBucketName, destinationPath, err := SplitGCSPath(destinationGcsPath)
	if err != nil {
		return fmt.Errorf("invalid destination path: %v", destinationGcsPath)
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
			return errors.New("tar subdirectories not supported")

		case tar.TypeReg:
			destinationFilePath := pathutils.JoinURL(destinationPath, header.Name)
			tge.logger.Log(fmt.Sprintf("Extracting: %v to gs://%v", header.Name, path.Join(destinationBucketName, destinationFilePath)))

			if err := tge.storageClient.WriteToGCS(destinationBucketName, destinationFilePath, tarReader); err != nil {
				return err
			}
		}
	}
}
