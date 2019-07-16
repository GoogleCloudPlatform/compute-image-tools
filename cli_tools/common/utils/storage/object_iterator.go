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

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
)

// ObjectIteratorCreator is responsible for creating GCS Object iterator
type ObjectIteratorCreator struct {
	ctx context.Context
	sc  *storage.Client
}

// CreateObjectIterator creates GCS Object iterator
func (bic *ObjectIteratorCreator) CreateObjectIterator(
	bucket string, objectPath string) commondomain.ObjectIteratorInterface {
	return &ObjectIterator{
		it: bic.sc.Bucket(bucket).Objects(bic.ctx, &storage.Query{Prefix: objectPath})}
}

// ObjectIterator is a wrapper around storage.ObjectIterator. Implements ObjectIteratorInterface.
type ObjectIterator struct {
	it *storage.ObjectIterator
}

// Next returns the next result. Its second return value is iterator.Done if
// there are no more results. Once Next returns iterator.Done, all subsequent
// calls will return iterator.Done.
func (bi *ObjectIterator) Next() (*storage.ObjectAttrs, error) {
	return bi.it.Next()
}
