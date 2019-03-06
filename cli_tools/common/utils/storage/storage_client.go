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
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	storagev1 "google.golang.org/api/storage/v1"
	htransport "google.golang.org/api/transport/http"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var (
	gsRegex = regexp.MustCompile(`^gs://([a-z0-9][-_.a-z0-9]*)/(.+)$`)
)

// StorageClient implements domain.StorageClientInterface. It implements main Storage functions
// used by image import features.
type StorageClient struct {
	ObjectDeleter commondomain.StorageObjectDeleterInterface
	StorageClient *storage.Client
	HTTPClient    domain.HTTPClientInterface
	Ctx           context.Context
	Oic           commondomain.ObjectIteratorCreatorInterface
}

// NewStorageClient creates a StorageClient
func NewStorageClient(ctx context.Context, client *storage.Client) (*StorageClient, error) {
	o := []option.ClientOption{option.WithScopes(storagev1.DevstorageReadOnlyScope)}
	hc, _, err := htransport.NewClient(ctx, o...)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage HTTP client %v", err.Error())
	}
	sc := &StorageClient{StorageClient: client, HTTPClient: &HTTPClient{hc},
		Ctx: ctx, Oic: &ObjectIteratorCreator{ctx: ctx, sc: client}}

	sc.ObjectDeleter = &StorageObjectDeleter{sc}
	return sc, nil
}

// CreateBucket creates a GCS bucket
func (sc *StorageClient) CreateBucket(
	bucketName string, project string, attrs *storage.BucketAttrs) error {
	return sc.StorageClient.Bucket(bucketName).Create(sc.Ctx, project, attrs)
}

// Buckets returns a bucket iterator for all buckets within a project
func (sc *StorageClient) Buckets(projectID string) *storage.BucketIterator {
	return sc.StorageClient.Buckets(sc.Ctx, projectID)
}

// GetBucketAttrs returns bucket attributes for given bucket
func (sc *StorageClient) GetBucketAttrs(bucket string) (*storage.BucketAttrs, error) {

	resp, err := sc.HTTPClient.Get("https://www.googleapis.com/storage/v1/b/" + bucket + "?fields=location%2CstorageClass")
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		bucketAttrs := &storage.BucketAttrs{}

		if err = json.Unmarshal(body, &bucketAttrs); err != nil {
			return nil, err
		}
		return bucketAttrs, nil
	}
	return nil, fmt.Errorf("error while retrieving `%v` bucket attributes: Error %v, %v", bucket, resp.StatusCode, resp.Status)
}

// GetObjectReader creates a new Reader to read the contents of the object.
func (sc *StorageClient) GetObjectReader(bucket string, objectPath string) (io.ReadCloser, error) {
	return sc.GetBucket(bucket).Object(objectPath).NewReader(sc.Ctx)
}

// GetBucket returns a BucketHandle, which provides operations on the named bucket.
func (sc *StorageClient) GetBucket(bucket string) *storage.BucketHandle {
	return sc.StorageClient.Bucket(bucket)
}

// GetObjects returns object iterator for given bucket and path
func (sc *StorageClient) GetObjects(bucket string, objectPath string) commondomain.ObjectIteratorInterface {
	return sc.Oic.CreateObjectIterator(bucket, objectPath)
}

// DeleteObject deletes GCS object in given bucket and object path
func (sc *StorageClient) DeleteObject(bucket string, objectPath string) error {
	return sc.ObjectDeleter.DeleteObject(bucket, objectPath)
}

// DeleteGcsPath deletes a GCS path, including files
func (sc *StorageClient) DeleteGcsPath(gcsPath string) error {
	bucketName, objectPath, err := SplitGCSPath(gcsPath)
	if err != nil {
		return err
	}
	log.Printf("Deleting content of: %v", gcsPath)

	it := sc.GetObjects(bucketName, objectPath)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Printf("Deleting gs://%v/%v\n", bucketName, attrs.Name)

		if err := sc.DeleteObject(bucketName, attrs.Name); err != nil {
			return err
		}
	}

	return nil
}

// FindGcsFile finds a file in a GCS directory path for given file extension. File extension can
// be a file name as well.
func (sc *StorageClient) FindGcsFile(gcsDirectoryPath string, fileExtension string) (*storage.ObjectHandle, error) {

	bucketName, objectPath, err := SplitGCSPath(gcsDirectoryPath)
	if err != nil {
		return nil, err
	}
	it := sc.GetObjects(bucketName, objectPath)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if !strings.HasSuffix(attrs.Name, fileExtension) {
			continue
		}
		fmt.Printf("Found gs://%v/%v\n", bucketName, attrs.Name)

		return sc.GetBucket(bucketName).Object(attrs.Name), nil
	}
	return nil, fmt.Errorf("path %v doesn't contain a file with %v extension", gcsDirectoryPath, fileExtension)
}

// GetGcsFileContent returns content of a GCS object as byte array
func (sc *StorageClient) GetGcsFileContent(gcsObject *storage.ObjectHandle) ([]byte, error) {
	reader, err := gcsObject.NewReader(sc.Ctx)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

// WriteToGCS writes content from a reader to destination bucket and path
func (sc *StorageClient) WriteToGCS(
	destinationBucketName string, destinationObjectPath string, reader io.Reader) error {
	destinationBucket := sc.GetBucket(destinationBucketName)
	fileWriter := destinationBucket.Object(destinationObjectPath).NewWriter(sc.Ctx)

	if _, err := io.Copy(fileWriter, reader); err != nil {
		return err
	}

	fileWriter.Close()
	return nil
}

// Close closes the Client.
//
// Close need not be called at program exit.
func (sc *StorageClient) Close() error {
	return sc.StorageClient.Close()
}

// SplitGCSPath splits GCS path into bucket and object path portions
func SplitGCSPath(p string) (string, string, error) {
	matches := gsRegex.FindStringSubmatch(p)
	if matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("%q is not a valid GCS path", p)
}

// HTTPClient implements domain.HTTPClientInterface which abstracts HTTP functionality used by
// image import features.
type HTTPClient struct {
	httpClient *http.Client
}

// Get executes HTTP GET request for given URL
func (hc *HTTPClient) Get(url string) (resp *http.Response, err error) {
	return hc.httpClient.Get(url)
}

// StorageObjectDeleter is responsible for deleting object
type StorageObjectDeleter struct {
	sc *StorageClient
}

// DeleteObject deletes GCS object in given bucket and path
func (sod *StorageObjectDeleter) DeleteObject(bucket string, objectPath string) error {
	return sod.sc.GetBucket(bucket).Object(objectPath).Delete(sod.sc.Ctx)
}
