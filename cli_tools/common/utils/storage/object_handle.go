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
	"io"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
)

// ObjectHandleCreator is responsible for creating GCS Object Handle
type ObjectHandleCreator struct {
	ctx context.Context
	sc  *storage.Client
}

// CreateObjectHandle creates GCS Object Handle
func (ohc *ObjectHandleCreator) CreateObjectHandle(
		bucket string, object string) domain.ObjectHandleInterface {
	return &ObjectHandle{
		ob: ohc.sc.Bucket(bucket).Object(object), ctx: ohc.ctx}
}

// ObjectHandle is a wrapper around storage.ObjectHandle. Implements ObjectHandleInterface.
type ObjectHandle struct {
	ob  *storage.ObjectHandle
	ctx context.Context
}

func (oh *ObjectHandle) GetObjectHandle() *storage.ObjectHandle {
	return oh.ob
}

// DeleteObject deletes GCS object in given bucket and object path
func (oh *ObjectHandle) Delete() error {
	return oh.ob.Delete(oh.ctx)
}
func (oh *ObjectHandle) NewReader() (io.ReadCloser, error) {
	return oh.ob.NewReader(oh.ctx)
}

func (oh *ObjectHandle) NewWriter() io.WriteCloser {
	return oh.ob.NewWriter(oh.ctx)
}

func (oh *ObjectHandle) ObjectName() string {
	return oh.ob.ObjectName()
}

func (oh *ObjectHandle) RunComposer(srcs ...domain.ObjectHandleInterface) (*storage.ObjectAttrs, error) {
	var objs []*storage.ObjectHandle
	for _, obj := range srcs {
		objs = append(objs, obj.GetObjectHandle())
	}
	return oh.ob.ComposerFrom(objs...).Run(oh.ctx)
}

func (oh *ObjectHandle) RunCopier(src domain.ObjectHandleInterface) (*storage.ObjectAttrs, error) {
	return oh.ob.CopierFrom(src.GetObjectHandle()).Run(oh.ctx)
}
