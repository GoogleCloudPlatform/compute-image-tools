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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
)

const bucketName = "compute-image-tools-test"

var (
	artifactsDir = os.Getenv("ARTIFACTS")
	gcsURLBase   = getBase()
	gcsBucket    *storage.BucketHandle
	buildID      = os.Getenv("BUILD_ID")
	jobName      = os.Getenv("JOB_NAME")
	jobType      = os.Getenv("JOB_TYPE")
	pullNum      = os.Getenv("PULL_NUMBER")
	pullRefs     = os.Getenv("PULL_REFS")
	pullSHA      = os.Getenv("PULL_PULL_SHA")
	repoName     = os.Getenv("REPO_NAME")
	repoOwner    = os.Getenv("REPO_OWNER")

	buildLog *log.Logger
)

func finished(result string) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"result":    result,
	})
	return data
}

func gcsWrite(ctx context.Context, p string, data []byte, dataR io.Reader, ct string) error {
	var err error
	o := gcsBucket.Object(gcsURLBase + p)
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
	switch jobType {
	case "batch":
		return fmt.Sprintf("pr-logs/pull/batch/%s/%s/", jobName, buildID)
	case "periodic":
		return fmt.Sprintf("logs/%s/%s/", jobName, buildID)
	case "postsumbit":
		return fmt.Sprintf("logs/%s/%s/", jobName, buildID)
	case "presubmit":
		return fmt.Sprintf("pr-logs/pull/%s_%s/%s/%s/%s/", repoOwner, repoName, pullNum, jobName, buildID)
	}
	return ""
}

func started() []byte {
	var s map[string]interface{}
	switch jobType {
	case "batch":
		s = map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"pull":      pullRefs,
		}
	case "periodic":
		s = map[string]interface{}{
			"timestamp": time.Now().Unix(),
		}
	case "postsubmit":
		s = map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"pull":      pullRefs,
		}
	case "presubmit":
		s = map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"repos":     map[string]string{fmt.Sprintf("%s/%s", repoOwner, repoName): pullSHA},
			"pull":      pullRefs,
		}
	}
	data, _ := json.Marshal(s)
	return data
}

func main() {
	ctx := context.Background()
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	gcsBucket = gcs.Bucket(bucketName)
	logfile, err := ioutil.TempFile("/tmp", "build-log")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create build log.", err)
	}
	buildLog = log.New(io.MultiWriter(logfile, os.Stdout), "", 0)

	// Write started.json
	buildLog.Println("Writing started.json to GCS.")
	if err := gcsWrite(ctx, "started.json", started(), nil, "application/json"); err != nil {
		buildLog.Println("ERROR: ", err)
	}

	// Run the main process.
	buildLog.Println("Running wrapped logic.")
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	out, err := cmd.CombinedOutput()
	buildLog.Println(string(out))
	if err != nil {
		buildLog.Println("ERROR: ", err)
	}
	buildLog.Println("Main logic finished.")
	result := "SUCCESS"
	if err != nil {
		result = "FAILURE"
	}

	// Copy artifacts.
	buildLog.Println("Writing artifacts to GCS.")
	filepath.Walk("artifacts", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			if p == "artifacts" {
				buildLog.Println("No artifacts to write")
				return nil
			}
			buildLog.Println("ERROR: ", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		buildLog.Printf("Writing artifact %s to GCS.", p)
		if f, err := os.Open(p); err != nil {
			buildLog.Println("ERROR: ", err)
		} else {
			gcsP := "artifacts/" + p[len(artifactsDir):] // remove ARTIFACTS dir, and slash, prefix from p.
			if err := gcsWrite(ctx, gcsP, nil, f, ""); err != nil {
				buildLog.Println("ERROR: ", err)
			}
		}
		return nil
	})

	// Write finished.json
	buildLog.Println("Writing finished.json to GCS.")
	if err := gcsWrite(ctx, "finished.json", finished(result), nil, "application/json"); err != nil {
		buildLog.Println("ERROR: ", err)
	}

	// Close and write build-log.txt.
	buildLog.Println("Closing and writing build-log.txt to GCS.")
	logfile.Seek(0, 0)
	gcsWrite(ctx, "build-log.txt", nil, logfile, "text/plain")
	logfile.Close()

	if result != "SUCCESS" {
		os.Exit(1)
	}
}
