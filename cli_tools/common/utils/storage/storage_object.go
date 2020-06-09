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

// StorageObjectCreator is responsible for creating GCS Object.
type StorageObjectCreator struct {
	ctx context.Context
	sc  *storage.Client
}

// GetObject gets GCS object interface.
func (ohc *StorageObjectCreator) GetObject(
	bucket string, object string) domain.StorageObjectInterface {
	return &StorageObject{
		oh: ohc.sc.Bucket(bucket).Object(object), ctx: ohc.ctx}
}

// StorageObject is a wrapper around storage.StorageObject. Implements StorageObjectInterface.
type StorageObject struct {
	oh  *storage.ObjectHandle
	ctx context.Context
}

// GetObjectHandle gets the storage object handle.
func (so *StorageObject) GetObjectHandle() *storage.ObjectHandle {
	return so.oh
}

// Delete deletes GCS object.
func (so *StorageObject) Delete() error {
	return so.oh.Delete(so.ctx)
}

// NewReader creates a new Reader to read the contents of the object.
func (so *StorageObject) NewReader() (io.ReadCloser, error) {
	return so.oh.NewReader(so.ctx)
}

// NewWriter creates a new Writer to write to the object.
func (so *StorageObject) NewWriter() io.WriteCloser {
	return so.oh.NewWriter(so.ctx)
}

// ObjectName returns the name of the object.
func (so *StorageObject) ObjectName() string {
	return so.oh.ObjectName()
}

// Compose takes in srcs as source objects and compose them into the destination object (dst).
// Up to 32 objects can be composed into a one object.
func (dst *StorageObject) Compose(srcs ...domain.StorageObjectInterface) (*storage.ObjectAttrs, error) {
	var objs []*storage.ObjectHandle
	for _, obj := range srcs {
		objs = append(objs, obj.GetObjectHandle())
	}
	return dst.oh.ComposerFrom(objs...).Run(dst.ctx)
}

// Copy copies the src object into the dst object.
func (dst *StorageObject) Copy(src domain.StorageObjectInterface) (*storage.ObjectAttrs, error) {
	return dst.oh.CopierFrom(src.GetObjectHandle()).Run(dst.ctx)
}
