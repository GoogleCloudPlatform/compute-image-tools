// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"context"
	"encoding/json"
	"path/filepath"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"cloud.google.com/go/storage"
)

const BUCKET  = "compute-image-tools-test"

var (
	BASE       = getBase()
	BKT        *storage.BucketHandle
	BUILDID    = os.Getenv("BUILD_ID")
	JOBNAME    = os.Getenv("JOB_NAME")
	JOBTYPE    = os.Getenv("JOB_TYPE")
	PULLNUM    = os.Getenv("PULL_NUMBER")
	PULLREFS   = os.Getenv("PULL_REFS")
	PULLSHA    = os.Getenv("PULL_PULL_SHA")
	REPONAME   = os.Getenv("REPO_NAME")
	REPOOWNER  = os.Getenv("REPO_OWNER")

	LOG *log.Logger
)

func main() {
	ctx := context.Background()
	gcs, _ := storage.NewClient(ctx)
	BKT = gcs.Bucket(BUCKET)
	logfile, err := ioutil.TempFile("/tmp", "build-log")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create build log.", err)
	}
	LOG = log.New(io.MultiWriter(logfile, os.Stdout), "", 0)

	// Write started.json
	LOG.Println("Writing started.json to GCS.")
	logIfErr(gcsWrite("started.json", started(), nil, "application/json", ctx))

	// Run the main process.
	LOG.Println("Running wrapped logic.")
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	out, err := cmd.Output()
	LOG.Println(string(out))
	logIfErr(err)
	LOG.Println("Main logic finished.")
	result := "SUCCESS"
	if err != nil {
		result = "FAILURE"
	}

	// Copy artifacts.
	LOG.Println("Writing artifacts to GCS.")
	filepath.Walk("artifacts", func (p string, info os.FileInfo, err error) error {
		if err != nil {
			logIfErr(err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		LOG.Printf("Writing artifact %s to GCS.", p)
		if f, err := os.Open(p); err != nil {
			logIfErr(err)
		} else {
			logIfErr(gcsWrite(p, nil, f, "", ctx))
		}
		return nil
	})

	// Write finished.json
	LOG.Println("Writing finished.json to GCS.")
	logIfErr(gcsWrite("finished.json", finished(result), nil, "application/json", ctx))

	// Close and write build-log.txt.
	LOG.Println("Closing and writing build-log.txt to GCS.")
	logfile.Seek(0, 0)
	gcsWrite("build-log.txt", nil, logfile, "text/plain", ctx)
	logfile.Close()
}

func finished(result string) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"result": result,
	})
	return data
}

func gcsWrite(p string, data []byte, dataR io.Reader, ct string, ctx context.Context) error {
	var err error
	o := BKT.Object(BASE + p)
	w := o.NewWriter(ctx)
	w.ObjectAttrs.ContentType = ct
	if len(data) > 0 {
		_, err = w.Write(data)
	} else if dataR != nil {
		_, err = io.Copy(w, dataR)
	}
	if err != nil {
		return err
	}
	return w.Close()
}

func getBase() string {
	switch JOBTYPE {
	case "batch":
		return fmt.Sprintf("pr-logs/pull/batch/%s/%s/", JOBNAME, BUILDID)
	case "periodic":
		return fmt.Sprintf("logs/%s/%s/", JOBNAME, BUILDID)
	case "postsumbit":
		return fmt.Sprintf("logs/%s/%s/", JOBNAME, BUILDID)
	case "presubmit":
		return fmt.Sprintf("pr-logs/pull/%s_%s/%s/%s/%s/", REPOOWNER, REPONAME, PULLNUM, JOBNAME, BUILDID)
	}
	return ""
}

func logIfErr(err error) {
	if err == nil {
		return
	}
	LOG.Println("ERROR: ", err)
}

func started() []byte {
	var s map[string]interface{}
	switch JOBTYPE {
	case "batch":
		s = map[string]interface{} {
			"timestamp": time.Now().Unix(),
			"pull": PULLREFS,
		}
	case "periodic":
		s = map[string]interface{} {
			"timestamp": time.Now().Unix(),
		}
	case "postsubmit":
		s = map[string]interface{} {
			"timestamp": time.Now().Unix(),
			"pull": PULLREFS,
		}
	case "presubmit":
		s = map[string]interface{} {
			"timestamp": time.Now().Unix(),
			"repos": map[string]string{fmt.Sprintf("%s/%s", REPOOWNER, REPONAME): PULLSHA},
			"pull": PULLREFS,
		}
	}
	data, _ := json.Marshal(s)
	return data
}
