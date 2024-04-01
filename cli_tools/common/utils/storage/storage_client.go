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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

var (
	gsPathRegex       = regexp.MustCompile(fmt.Sprintf(`^gs://(%s)(\/.*)?$`, bucketNameRegex))
	slashCounterRegex = regexp.MustCompile("/")
)

// Client implements domain.StorageClientInterface. It implements main Storage functions
// used by image import features.
type Client struct {
	StorageClient *storage.Client
	Logger        logging.Logger
	Ctx           context.Context
	Oic           domain.ObjectIteratorCreatorInterface
	Soc           domain.StorageObjectCreatorInterface
}

// NewStorageClient creates a Client
func NewStorageClient(ctx context.Context,
	logger logging.Logger, option ...option.ClientOption) (*Client, error) {

	client, err := storage.NewClient(ctx, option...)
	if err != nil {
		return nil, daisy.Errf("error creating storage client: %v", err)
	}
	sc := &Client{StorageClient: client, Ctx: ctx,
		Oic: &ObjectIteratorCreator{ctx: ctx, sc: client}, Logger: logger}

	sc.Soc = &storageObjectCreator{ctx: ctx, sc: client}
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

// GetBucket returns a BucketHandle, which provides operations on the named bucket.
func (sc *Client) GetBucket(bucket string) *storage.BucketHandle {
	return sc.StorageClient.Bucket(bucket)
}

// GetBucketAttrs returns bucket attributes for given bucket
func (sc *Client) GetBucketAttrs(bucket string) (*storage.BucketAttrs, error) {
	bucketAttrs, err := sc.StorageClient.Bucket(bucket).Attrs(sc.Ctx)
	if err != nil {
		return nil, daisy.Errf("Error getting bucket attributes for bucket `%v`: %v", bucket, err)
	}
	return bucketAttrs, nil
}

// GetObject returns storage object for the given bucket and path
func (sc *Client) GetObject(bucket string, objectPath string) domain.StorageObject {
	return sc.Soc.GetObject(bucket, objectPath)
}

// GetObjects returns object iterator for given bucket and path
func (sc *Client) GetObjects(bucket string, objectPath string) domain.ObjectIteratorInterface {
	return sc.Oic.CreateObjectIterator(bucket, objectPath)
}

// GetObjectAttrs returns storage object attributes
func (sc *Client) GetObjectAttrs(bucket string, objectPath string) (*storage.ObjectAttrs, error) {
	objectAttrs, err := sc.StorageClient.Bucket(bucket).Object(objectPath).Attrs(sc.Ctx)
	if err != nil {
		return nil, daisy.Errf("Error getting object attributes for object `%v\\%v`: %v", bucket, objectPath, err)
	}
	return objectAttrs, nil
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
		sc.Logger.User(fmt.Sprintf("Deleting gs://%v/%v", bucketName, attrs.Name))

		if err := sc.GetObject(bucketName, attrs.Name).Delete(); err != nil {
			return daisy.Errf("Error deleting Cloud Storage object `%v` in bucket `%v`: %v", attrs.Name, bucketName, err)
		}
	}

	return nil
}

// DeleteObject deletes the object with the path `gcsPath`.
func (sc *Client) DeleteObject(gcsPath string) error {
	bucketName, objectPath, err := SplitGCSPath(gcsPath)
	if err != nil {
		return daisy.Errf("Error deleting `%v`: `%v`", gcsPath, err)
	}
	if err := sc.GetObject(bucketName, objectPath).Delete(); err != nil {
		return daisy.Errf("Error deleting `%v`: `%v`", gcsPath, err)
	}
	return nil
}

// FindGcsFile finds a file in a GCS directory path for given file extension. File extension can
// be a file name as well. The lookup is done recursively.
func (sc *Client) FindGcsFile(gcsDirectoryPath string, fileExtension string) (*storage.ObjectHandle, error) {
	return sc.FindGcsFileDepthLimited(gcsDirectoryPath, fileExtension, -1)
}

// FindGcsFileDepthLimited finds a file in a GCS directory path for given file
// extension up to lookupDepth deep. If lookup should be only for files directly in
// gcsDirectoryPath, lookupDepth should be set as 0. For recursive lookup with
// no limitations on depth, lookupDepth should be -1
// File extension can be a file name as well.
func (sc *Client) FindGcsFileDepthLimited(gcsDirectoryPath string, fileExtension string, lookupDepth int) (*storage.ObjectHandle, error) {
	bucketName, lookupPath, err := SplitGCSPath(gcsDirectoryPath)
	if err != nil {
		return nil, err
	}
	it := sc.GetObjects(bucketName, lookupPath)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, daisy.Errf("Error finding file with extension `%v` in Cloud Storage directory `%v`: %v", fileExtension, gcsDirectoryPath, err)
		}
		if !isDepthValid(lookupDepth, lookupPath, attrs.Name) {
			continue
		}
		if !strings.HasSuffix(attrs.Name, fileExtension) {
			continue
		}
		sc.Logger.User(fmt.Sprintf("Found gs://%v/%v", bucketName, attrs.Name))

		return sc.GetBucket(bucketName).Object(attrs.Name), nil
	}
	return nil, daisy.Errf(
		"path %v doesn't contain a file with %v extension", gcsDirectoryPath, fileExtension)
}

func isDepthValid(lookupDepth int, lookupPath, objectPath string) bool {
	if lookupDepth <= -1 {
		return true
	}
	if strings.HasSuffix(lookupPath, "/") {
		lookupPath = lookupPath[:len(lookupPath)-1]
	}
	lookupPathDepth := 0
	if len(lookupPath) > 0 {
		// lookup path is a "folder", have to count all elements as one level, thus the +1
		lookupPathDepth = 1 + getSlashCount(lookupPath)
	}
	// objectPath refers to an object so its path depth is one less than if it was a folder, thus no +1
	objectDepth := getSlashCount(objectPath)
	return objectDepth-lookupPathDepth <= lookupDepth
}

func getSlashCount(path string) int {
	return len(slashCounterRegex.FindAllStringIndex(path, -1))
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

// ConcatGCSPath concatenates multiple elements of GCS path into a GCS path.
func ConcatGCSPath(pathElements ...string) string {
	path := ""
	for i, pathElement := range pathElements {
		path += pathElement
		if i != len(pathElements)-1 && !strings.HasSuffix(pathElement, "/") {
			path += "/"
		}
	}
	return path
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
