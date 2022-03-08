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

package ovfexporter

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/iterator"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
)

type ovfManifestGeneratorImpl struct {
	storageClient domain.StorageClientInterface
	cancelChan    chan string
}

// NewManifestFileGenerator creates a new manifest file generator
func NewManifestFileGenerator(storageClient domain.StorageClientInterface) ovfexportdomain.OvfManifestGenerator {
	return &ovfManifestGeneratorImpl{
		storageClient: storageClient,
		cancelChan:    make(chan string),
	}
}

// GenerateAndWriteManifestFile generates a manifest file for all files in a GCS
// directory
func (g *ovfManifestGeneratorImpl) GenerateAndWriteToGCS(gcsPath, manifestFileName string) error {
	e := make(chan error)
	go func() {
		e <- g.generateAndWriteToGCS(gcsPath, manifestFileName)
	}()

	select {
	case err := <-e:
		return err
	case cancelReason := <-g.cancelChan:
		return fmt.Errorf("generating manifest file cancelled: %v", cancelReason)
	}
}

func (g *ovfManifestGeneratorImpl) generateAndWriteToGCS(gcsPath, manifestFileName string) error {
	bucketName, directoryPath, err := storageutils.SplitGCSPath(gcsPath)
	if err != nil {
		return err
	}
	manifestFileContent, err := g.generate(bucketName, directoryPath)
	if err != nil {
		return err
	}
	if err := g.storageClient.WriteToGCS(
		bucketName,
		storageutils.ConcatGCSPath(directoryPath, manifestFileName),
		strings.NewReader(manifestFileContent)); err != nil {
		return err
	}
	return nil
}

func (g *ovfManifestGeneratorImpl) generate(bucketName, directoryPath string) (string, error) {
	var manifest strings.Builder
	it := g.storageClient.GetObjects(bucketName, directoryPath)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		sha1, err := g.generateFileSHA1Signature(bucketName, attrs.Name)
		if err != nil {
			return "", err
		}
		manifest.WriteString(fmt.Sprintf("SHA1(%v)= %v\n", attrs.Name[strings.LastIndex(attrs.Name, "/")+1:], sha1))
	}
	return manifest.String(), nil
}

func (g *ovfManifestGeneratorImpl) generateFileSHA1Signature(bucketName, objectPath string) (string, error) {
	fileReader, err := g.storageClient.GetObject(bucketName, objectPath).NewReader()
	if err != nil {
		return "", err
	}
	defer fileReader.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, fileReader); err != nil {
		return "", daisy.Errf("error while generating SHA1 for gs://%v/%v: %v", bucketName, objectPath, err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (g *ovfManifestGeneratorImpl) Cancel(reason string) bool {
	g.cancelChan <- reason
	return true
}
