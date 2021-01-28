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
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

type objectRegistry struct {
	created []string
	mx      sync.Mutex
}

func newObjectRegistry(w *Workflow) *objectRegistry {
	return &objectRegistry{}
}

func (o *objectRegistry) regCreate(object string) DError {
	o.mx.Lock()
	defer o.mx.Unlock()

	if strIn(object, o.created) {
		return Errf("cannot create object %q, object already created by another step", object)
	}

	o.created = append(o.created, object)
	return nil
}

var sourceVarRgx = regexp.MustCompile(`\$\{SOURCE:([^}]+)}`)

func (w *Workflow) recursiveGCS(ctx context.Context, bkt, prefix, dst string) DError {
	it := w.StorageClient.Bucket(bkt).Objects(ctx, &storage.Query{Prefix: prefix})
	for objAttr, err := it.Next(); err != iterator.Done; objAttr, err = it.Next() {
		if err != nil {
			return typedErr(apiError, "failed to iterate GCS objects for uploading", err)
		}
		if objAttr.Size == 0 {
			continue
		}
		srcPath := w.StorageClient.Bucket(bkt).Object(objAttr.Name)
		o := path.Join(w.sourcesPath, dst, strings.TrimPrefix(objAttr.Name, prefix))
		dstPath := w.StorageClient.Bucket(w.bucket).Object(o)
		if _, err := dstPath.CopierFrom(srcPath).Run(ctx); err != nil {
			return typedErr(apiError, "failed to upload GCS object", err)
		}
	}
	return nil
}

func (w *Workflow) sourceExists(s string) bool {
	_, ok := w.Sources[s]
	return ok
}

func (w *Workflow) sourceContent(ctx context.Context, s string) (string, error) {
	src, ok := w.Sources[s]
	if !ok {
		return "", Errf("source not found: %s", s)
	}
	// Try GCS file first.
	if bkt, objPath, err := splitGCSPath(src); err == nil {
		if objPath == "" || strings.HasSuffix(objPath, "/") {
			return "", Errf("source %s appears to be a GCS 'bucket'", src)

		}
		src := w.StorageClient.Bucket(bkt).Object(objPath)
		r, err := src.NewReader(ctx)
		if err != nil {
			return "", Errf("error reading from file %s/%s: %v", bkt, objPath, err)
		}
		defer r.Close()

		if r.Size() > 1024 {
			return "", Errf("file size is too large %s/%s: %d", bkt, objPath, r.Size())
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			return "", Errf("error reading from file %s/%s: %v", bkt, objPath, err)
		}

		return buf.String(), nil
	}
	// Fall back to local read.
	if !filepath.IsAbs(src) {
		src = filepath.Join(w.workflowDir, src)
	}
	if _, err := os.Stat(src); err != nil {
		return "", typedErr(fileIOError, "failed to find local file", err)
	}

	d, err := ioutil.ReadFile(src)
	if err != nil {
		return "", newErr("failed to read local file content", err)
	}
	return string(d), nil
}

func (w *Workflow) uploadFile(ctx context.Context, src, obj string) DError {
	obj = filepath.ToSlash(obj)
	dstPath := w.StorageClient.Bucket(w.bucket).Object(path.Join(w.sourcesPath, obj))
	gcs := dstPath.NewWriter(ctx)
	f, err := os.Open(src)
	if err != nil {
		return newErr("failed to open local file for uploading", err)
	}
	if _, err := io.Copy(gcs, f); err != nil {
		return newErr("failed to copy local file to GCS", err)
	}
	return newErr("failed to close GCS object", gcs.Close())
}

func (w *Workflow) uploadSources(ctx context.Context) DError {
	for dst, origPath := range w.Sources {
		if origPath == "" {
			continue
		}
		// GCS to GCS.
		if bkt, objPath, err := splitGCSPath(origPath); err == nil {
			if objPath == "" || strings.HasSuffix(objPath, "/") {
				if err := w.recursiveGCS(ctx, bkt, objPath, dst); err != nil {
					return Errf("error copying from bucket %s: %v", origPath, err)
				}
				continue
			}
			src := w.StorageClient.Bucket(bkt).Object(objPath)
			dstPath := w.StorageClient.Bucket(w.bucket).Object(path.Join(w.sourcesPath, dst))
			if _, err := dstPath.CopierFrom(src).Run(ctx); err != nil {
				if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
					return typedErrf(resourceDNEError, "error copying from file %s: %v", origPath, err)
				}
				return Errf("error copying from file %s: %v", origPath, err)
			}
			continue
		}

		// Local to GCS.
		if !filepath.IsAbs(origPath) {
			origPath = filepath.Join(w.workflowDir, origPath)
		}
		fi, err := os.Stat(origPath)
		if err != nil {
			return typedErr(fileIOError, "failed to open local file", err)
		}
		if fi.IsDir() {
			var files []string
			if err := filepath.Walk(origPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				files = append(files, path)
				return nil
			}); err != nil {
				return typedErr(fileIOError, "failed to walk file path", err)
			}
			for _, file := range files {
				obj := path.Join(dst, strings.TrimPrefix(file, filepath.Clean(origPath)))
				if err := w.uploadFile(ctx, file, obj); err != nil {
					return err
				}
			}
			continue
		}
		if err := w.uploadFile(ctx, origPath, dst); err != nil {
			return err
		}
	}
	return nil
}
