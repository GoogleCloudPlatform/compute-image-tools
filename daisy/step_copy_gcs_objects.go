//  Copyright 2017 Google Inc. All Rights Reserved.
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

package daisy

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// CopyGCSFiles is a Daisy CopyGCSFiles workflow step.
type CopyGCSObjects []CopyGCSObject

// CopyGCSFile copies a GCS file from Source to Destination.
type CopyGCSObject struct {
	Source, Destination string
	ACLRules            []storage.ACLRule
}

func (c *CopyGCSObjects) populate(ctx context.Context, s *Step) error { return nil }

func (c *CopyGCSObjects) validate(ctx context.Context, s *Step) error { return nil }

func recursiveGCS(ctx context.Context, w *Workflow, sBkt, sPrefix, dBkt, dPrefix string) error {
	it := w.StorageClient.Bucket(sBkt).Objects(ctx, &storage.Query{Prefix: sPrefix})
	for objAttr, err := it.Next(); err != iterator.Done; objAttr, err = it.Next() {
		if err != nil {
			return err
		}
		if objAttr.Size == 0 {
			continue
		}
		srcPath := w.StorageClient.Bucket(sBkt).Object(objAttr.Name)
		o := path.Join(dPrefix, strings.TrimPrefix(objAttr.Name, sPrefix))
		dstPath := w.StorageClient.Bucket(dBkt).Object(o)
		if _, err := dstPath.CopierFrom(srcPath).Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *CopyGCSObjects) run(ctx context.Context, s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, co := range *c {
		wg.Add(1)
		go func(co CopyGCSObject) {
			defer wg.Done()
			sBkt, sObj, err := splitGCSPath(co.Source)
			if err != nil {
				e <- err
				return
			}
			dBkt, dObj, err := splitGCSPath(co.Destination)
			if err != nil {
				e <- err
				return
			}

			if sObj == "" || strings.HasSuffix(sObj, "/") {
				if err := recursiveGCS(ctx, s.w, sBkt, sObj, dBkt, dObj); err != nil {
					e <- fmt.Errorf("error copying from %s to %s: %v", co.Source, co.Destination, err)
					return
				}
				return
			}

			src := s.w.StorageClient.Bucket(sBkt).Object(sObj)
			dstPath := s.w.StorageClient.Bucket(dBkt).Object(dObj)
			if _, err := dstPath.CopierFrom(src).Run(ctx); err != nil {
				e <- fmt.Errorf("error copying from %s to %s: %v", co.Source, co.Destination, err)
				return
			}
			for _, acl := range co.ACLRules {
				if err := dstPath.ACL().Set(ctx, acl.Entity, acl.Role); err != nil {
					e <- fmt.Errorf("error setting ACLRule on %s: %v", co.Destination, err)
					return
				}
			}
		}(co)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		return nil
	}
}
