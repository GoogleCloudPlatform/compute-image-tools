package workflow

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func (w *Workflow) recursiveGCS(bkt, prefix, dst string) error {
	it := w.StorageClient.Bucket(bkt).Objects(w.Ctx, &storage.Query{Prefix: prefix})
	for objAttr, err := it.Next(); err != iterator.Done; objAttr, err = it.Next() {
		if err != nil {
			return err
		}
		if objAttr.Size == 0 {
			continue
		}
		srcPath := w.StorageClient.Bucket(bkt).Object(objAttr.Name)
		o := path.Join(w.sourcesPath, dst, strings.TrimPrefix(objAttr.Name, prefix))
		dstPath := w.StorageClient.Bucket(w.bucket).Object(o)
		if _, err := dstPath.CopierFrom(srcPath).Run(w.Ctx); err != nil {
			return err
		}
	}
	return nil
}

func (w *Workflow) sourceExists(s string) bool {
	_, ok := w.Sources[s]
	return ok
}

func (w *Workflow) uploadFile(src, obj string) error {
	obj = filepath.ToSlash(obj)
	dstPath := w.StorageClient.Bucket(w.bucket).Object(path.Join(w.sourcesPath, obj))
	gcs := dstPath.NewWriter(w.Ctx)
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	if _, err := io.Copy(gcs, f); err != nil {
		return err
	}
	return gcs.Close()
}

func (w *Workflow) uploadSources() error {
	for dst, origPath := range w.Sources {
		// GCS to GCS.
		if bkt, objPath, err := splitGCSPath(origPath); err == nil {
			if objPath == "" || strings.HasSuffix(objPath, "/") {
				if err := w.recursiveGCS(bkt, objPath, dst); err != nil {
					return fmt.Errorf("error copying from bucket %s: %v", origPath, err)
				}
				continue
			}
			src := w.StorageClient.Bucket(bkt).Object(objPath)
			dstPath := w.StorageClient.Bucket(w.bucket).Object(path.Join(w.sourcesPath, dst))
			if _, err := dstPath.CopierFrom(src).Run(w.Ctx); err != nil {
				return fmt.Errorf("error copying from file %s: %v", origPath, err)
			}

			continue
		}

		// Local to GCS.
		if !filepath.IsAbs(origPath) {
			origPath = filepath.Join(w.workflowDir, origPath)
		}
		fi, err := os.Stat(origPath)
		if err != nil {
			return err
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
				return err
			}
			for _, file := range files {
				obj := path.Join(dst, strings.TrimPrefix(file, filepath.Clean(origPath)))
				if err := w.uploadFile(file, obj); err != nil {
					return err
				}
			}
			continue
		}
		if err := w.uploadFile(origPath, dst); err != nil {
			return err
		}
	}
	for _, step := range w.Steps {
		if step.SubWorkflow != nil {
			step.SubWorkflow.workflow.uploadSources()
		}
	}
	return nil
}
