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

package storage

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
)

// storageObjectCreator is responsible for creating GCS Object.
type storageObjectCreator struct {
	ctx context.Context
	sc  *storage.Client
}

// GetObject gets GCS object.
func (ohc *storageObjectCreator) GetObject(
	bucket string, object string) domain.StorageObject {
	return &storageObject{
		oh: ohc.sc.Bucket(bucket).Object(object), ctx: ohc.ctx}
}

// storageObject implements StorageObject.
type storageObject struct {
	oh  *storage.ObjectHandle
	ctx context.Context
}

// GetObjectHandle gets the storage object handle.
func (so *storageObject) GetObjectHandle() *storage.ObjectHandle {
	return so.oh
}

// Delete deletes GCS object.
func (so *storageObject) Delete() error {
	return so.oh.Delete(so.ctx)
}

// NewReader creates a new Reader to read the contents of the object.
func (so *storageObject) NewReader() (io.ReadCloser, error) {
	return so.oh.NewReader(so.ctx)
}

// NewWriter creates a new Writer to write to the object.
func (so *storageObject) NewWriter() io.WriteCloser {
	return so.oh.NewWriter(so.ctx)
}

// ObjectName returns the name of the object.
func (so *storageObject) ObjectName() string {
	return so.oh.ObjectName()
}

// Compose takes in srcs as source objects and compose them into the destination object (dst).
// Up to 32 objects can be composed into a one object.
func (so *storageObject) Compose(srcs ...domain.StorageObject) (*storage.ObjectAttrs, error) {
	var objs []*storage.ObjectHandle
	for _, obj := range srcs {
		objs = append(objs, obj.GetObjectHandle())
	}
	return so.oh.ComposerFrom(objs...).Run(so.ctx)
}

// Copy copies the src object into the dst object.
func (so *storageObject) Copy(src domain.StorageObject) (*storage.ObjectAttrs, error) {
	return so.oh.CopierFrom(src.GetObjectHandle()).Run(so.ctx)
}
