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

// CopyGCSObjects is a Daisy CopyGCSObject workflow step.
type CopyGCSObjects []CopyGCSObject

// CopyGCSObject copies a GCS object from Source to Destination.
type CopyGCSObject struct {
	Source, Destination string
	ACLRules            []*storage.ACLRule `json:",omitempty"`
}

func (c *CopyGCSObjects) populate(ctx context.Context, s *Step) DError {
	for _, co := range *c {
		for _, acl := range co.ACLRules {
			acl.Role = storage.ACLRole(strings.ToUpper(string(acl.Role)))
		}
	}
	return nil
}

func (c *CopyGCSObjects) validate(ctx context.Context, s *Step) DError {
	for _, co := range *c {
		sBkt, _, err := splitGCSPath(co.Source)
		if err != nil {
			return err
		}
		dBkt, dObj, err := splitGCSPath(co.Destination)
		if err != nil {
			return err
		}

		// Add object to object list.
		if err := s.w.objects.regCreate(path.Join(dBkt, dObj)); err != nil {
			return err
		}

		// Check if source bucket exists and is readable.
		readableBkts.mx.Lock()
		if !strIn(sBkt, readableBkts.bkts) {
			if _, err := s.w.StorageClient.Bucket(sBkt).Attrs(ctx); err != nil {
				return Errf("error reading bucket %q: %v", sBkt, err)
			}
			readableBkts.bkts = append(readableBkts.bkts, sBkt)
		}
		readableBkts.mx.Unlock()

		// Check if destination bucket exists and is readable.
		writableBkts.mx.Lock()
		if !strIn(dBkt, writableBkts.bkts) {
			if _, err := s.w.StorageClient.Bucket(dBkt).Attrs(ctx); err != nil {
				return Errf("error reading bucket %q: %v", dBkt, err)
			}

			// Check if destination bucket is writable.
			tObj := s.w.StorageClient.Bucket(dBkt).Object(fmt.Sprintf("daisy-validate-%s-%s", s.name, s.w.id))
			w := tObj.NewWriter(ctx)
			if _, err := w.Write(nil); err != nil {
				return newErr("failed to ", err)
			}
			if err := w.Close(); err != nil {
				return Errf("error writing to bucket %q: %v", dBkt, err)
			}
			if err := tObj.Delete(ctx); err != nil {
				return Errf("error deleting file %+v after write validation: %v", tObj, err)
			}
			writableBkts.bkts = append(writableBkts.bkts, dBkt)
		}
		writableBkts.mx.Unlock()

		// Check each ACLRule
		for _, acl := range co.ACLRules {
			if acl.Entity == "" {
				return Errf("ACLRule.Entity must not be empty: %+v", acl)
			}
			roles := []string{string(storage.RoleOwner), string(storage.RoleReader), string(storage.RoleWriter)}
			if !strIn(string(acl.Role), roles) {
				return Errf("ACLRule.Role invalid: %q not one of %q", acl.Role, roles)
			}

			// Test ACLRule.Entity.
			tObj := s.w.StorageClient.Bucket(dBkt).Object(fmt.Sprintf("daisy-validate-%s-%s", s.name, s.w.id))
			w := tObj.NewWriter(ctx)
			if _, err := w.Write(nil); err != nil {
				return newErr("failed to write to GCS object when testing ACLRule.Entity", err)
			}
			if err := w.Close(); err != nil {
				return newErr("failed to close GCS object when testing ACLRule.Entity", err)
			}
			defer tObj.Delete(ctx)

			if err := tObj.ACL().Set(ctx, acl.Entity, acl.Role); err != nil {
				return Errf("error validating ACLRule %+v: %v", acl, err)
			}
		}
	}

	return nil
}

func recursiveGCS(ctx context.Context, w *Workflow, sBkt, sPrefix, dBkt, dPrefix string, acls []*storage.ACLRule) DError {
	it := w.StorageClient.Bucket(sBkt).Objects(ctx, &storage.Query{Prefix: sPrefix})
	for objAttr, err := it.Next(); err != iterator.Done; objAttr, err = it.Next() {
		if err != nil {
			return typedErr(apiError, "failed to iterate GCS objects for copying", err)
		}
		if objAttr.Size == 0 {
			continue
		}
		srcPath := w.StorageClient.Bucket(sBkt).Object(objAttr.Name)
		o := path.Join(dPrefix, strings.TrimPrefix(objAttr.Name, sPrefix))
		dstPath := w.StorageClient.Bucket(dBkt).Object(o)
		if _, err := dstPath.CopierFrom(srcPath).Run(ctx); err != nil {
			return typedErr(apiError, "failed to copy GCS object", err)
		}

		for _, acl := range acls {
			if err := dstPath.ACL().Set(ctx, acl.Entity, acl.Role); err != nil {
				return typedErr(apiError, "failed to set ACL for GCS object", err)
			}
		}
	}
	return nil
}

func (c *CopyGCSObjects) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
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
				if err := recursiveGCS(ctx, s.w, sBkt, sObj, dBkt, dObj, co.ACLRules); err != nil {
					e <- Errf("error copying from %s to %s: %v", co.Source, co.Destination, err)
					return
				}
				return
			}

			src := s.w.StorageClient.Bucket(sBkt).Object(sObj)
			dstPath := s.w.StorageClient.Bucket(dBkt).Object(dObj)
			if _, err := dstPath.CopierFrom(src).Run(ctx); err != nil {
				e <- Errf("error copying from %s to %s: %v", co.Source, co.Destination, err)
				return
			}
			for _, acl := range co.ACLRules {
				if err := dstPath.ACL().Set(ctx, acl.Entity, acl.Role); err != nil {
					e <- Errf("error setting ACLRule on %s: %v", co.Destination, err)
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
