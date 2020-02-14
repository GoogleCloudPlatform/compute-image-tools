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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	bucketNameRegex = `[a-z0-9][-_.a-z0-9]*`
	gsPathRegex     = regexp.MustCompile(fmt.Sprintf(`^gs://(%s)(\/.*)?$`, bucketNameRegex))
)

// Client implements domain.StorageClientInterface. It implements main Storage functions
// used by image import features.
type Client struct {
	ObjectDeleter domain.StorageObjectDeleterInterface
	StorageClient *storage.Client
	Logger        logging.LoggerInterface
	Ctx           context.Context
	Oic           domain.ObjectIteratorCreatorInterface
}

// NewStorageClient creates a Client
func NewStorageClient(ctx context.Context,
	logger logging.LoggerInterface, oauth string) (*Client, error) {

	storageOptions := []option.ClientOption{}
	if oauth != "" {
		storageOptions = append(storageOptions, option.WithCredentialsFile(oauth))
	}
	client, err := storage.NewClient(ctx, storageOptions...)
	if err != nil {
		return nil, daisy.Errf("error creating storage client: %v", err)
	}
	sc := &Client{StorageClient: client, Ctx: ctx,
		Oic: &ObjectIteratorCreator{ctx: ctx, sc: client}, Logger: logger}

	sc.ObjectDeleter = &ObjectDeleter{sc}
	return sc, nil
}

// CreateBucket creates a GCS bucket
func (sc *Client) CreateBucket(
	bucketName string, project string, attrs *storage.BucketAttrs) error {
	if err := sc.StorageClient.Bucket(bucketName).Create(sc.Ctx, project, attrs); err != nil {
		return daisy.Errf("Error creating bucket `%v` in project `%v`: %v", bucketName, project, err)
	}
	return nil
}

// Buckets returns a bucket iterator for all buckets within a project
func (sc *Client) Buckets(projectID string) *storage.BucketIterator {
	return sc.StorageClient.Buckets(sc.Ctx, projectID)
}

// GetBucketAttrs returns bucket attributes for given bucket
func (sc *Client) GetBucketAttrs(bucket string) (*storage.BucketAttrs, error) {
	bucketAttrs, err := sc.StorageClient.Bucket(bucket).Attrs(sc.Ctx)
	if err != nil {
		return nil, daisy.Errf("Error getting bucket attributes for bucket `%v`: %v", bucket, err)
	}
	return bucketAttrs, nil
}

// GetObjectReader creates a new Reader to read the contents of the object.
func (sc *Client) GetObjectReader(bucket string, objectPath string) (io.ReadCloser, error) {
	objectReader, err := sc.GetBucket(bucket).Object(objectPath).NewReader(sc.Ctx)
	if err != nil {
		return nil, daisy.Errf("Error getting object reader for bucket `%v` and object path `%v`: %v", bucket, objectPath, err)
	}
	return objectReader, nil
}

// GetBucket returns a BucketHandle, which provides operations on the named bucket.
func (sc *Client) GetBucket(bucket string) *storage.BucketHandle {
	return sc.StorageClient.Bucket(bucket)
}

// GetObjects returns object iterator for given bucket and path
func (sc *Client) GetObjects(bucket string, objectPath string) domain.ObjectIteratorInterface {
	return sc.Oic.CreateObjectIterator(bucket, objectPath)
}

// DeleteObject deletes GCS object in given bucket and object path
func (sc *Client) DeleteObject(bucket string, objectPath string) error {
	if err := sc.ObjectDeleter.DeleteObject(bucket, objectPath); err != nil {
		return daisy.Errf("Error deleting object for bucket `%v` and object path `%v`: %v", bucket, objectPath, err)
	}
	return nil
}

// DeleteGcsPath deletes a GCS path, including files
func (sc *Client) DeleteGcsPath(gcsPath string) error {
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
			return daisy.Errf("Error deleting Cloud Storage path `%v`: %v", gcsPath, err)
		}
		sc.Logger.Log(fmt.Sprintf("Deleting gs://%v/%v", bucketName, attrs.Name))

		if err := sc.DeleteObject(bucketName, attrs.Name); err != nil {
			return daisy.Errf("Error deleting Cloud Storage object `%v` in bucket `%v`: %v", attrs.Name, bucketName, err)
		}
	}

	return nil
}

// FindGcsFile finds a file in a GCS directory path for given file extension. File extension can
// be a file name as well.
func (sc *Client) FindGcsFile(gcsDirectoryPath string, fileExtension string) (*storage.ObjectHandle, error) {
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
			return nil, daisy.Errf("Error finding file with extension `%v` in Cloud Storage directory `%v`: %v", fileExtension, gcsDirectoryPath, err)
		}

		if !strings.HasSuffix(attrs.Name, fileExtension) {
			continue
		}
		sc.Logger.Log(fmt.Sprintf("Found gs://%v/%v", bucketName, attrs.Name))

		return sc.GetBucket(bucketName).Object(attrs.Name), nil
	}
	return nil, daisy.Errf(
		"path %v doesn't contain a file with %v extension", gcsDirectoryPath, fileExtension)
}

// GetGcsFileContent returns content of a GCS object as byte array
func (sc *Client) GetGcsFileContent(gcsObject *storage.ObjectHandle) ([]byte, error) {
	reader, err := gcsObject.NewReader(sc.Ctx)
	if err != nil {
		return nil, daisy.Errf("Error getting Cloud Storage file content: %v", err)
	}
	return ioutil.ReadAll(reader)
}

// WriteToGCS writes content from a reader to destination bucket and path
func (sc *Client) WriteToGCS(
	destinationBucketName string, destinationObjectPath string, reader io.Reader) error {
	destinationBucket := sc.GetBucket(destinationBucketName)
	fileWriter := destinationBucket.Object(destinationObjectPath).NewWriter(sc.Ctx)

	if _, err := io.Copy(fileWriter, reader); err != nil {
		return daisy.Errf("Error writing to Cloud Storage file path `%v` in bucket `%v`: %v", destinationObjectPath, destinationBucketName, err)
	}

	return fileWriter.Close()
}

// Close closes the Client.
//
// Close need not be called at program exit.
func (sc *Client) Close() error {
	if err := sc.StorageClient.Close(); err != nil {
		return daisy.Errf("Error closing storage client: %v", err)
	}
	return nil
}

// SplitGCSPath splits GCS path into bucket and object path portions
func SplitGCSPath(p string) (string, string, error) {
	matches := gsPathRegex.FindStringSubmatch(p)
	if matches != nil {
		return matches[1], strings.TrimLeft(matches[2], "/"), nil
	}

	return "", "", daisy.Errf("%q is not a valid Cloud Storage path", p)
}

// GetGCSObjectPathElements returns bucket name, object path within the bucket
// for a valid object path. Error is returned otherwise.
func GetGCSObjectPathElements(p string) (string, string, error) {
	bucket, object, err := SplitGCSPath(p)
	if err != nil || bucket == "" || object == "" {
		return "", "", daisy.Errf("%q is not a valid Cloud Storage object path", p)
	}
	return bucket, object, err
}

// GetBucketNameFromGCSPath splits GCS path to get bucket name
func GetBucketNameFromGCSPath(p string) (string, error) {
	bucket, _, err := SplitGCSPath(p)
	return bucket, err
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

// ObjectDeleter is responsible for deleting storage object
type ObjectDeleter struct {
	sc *Client
}

// DeleteObject deletes GCS object in given bucket and path
func (sod *ObjectDeleter) DeleteObject(bucket string, objectPath string) error {
	return sod.sc.GetBucket(bucket).Object(objectPath).Delete(sod.sc.Ctx)
}
