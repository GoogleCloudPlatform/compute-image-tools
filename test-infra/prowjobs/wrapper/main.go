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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
)

const (
	bucketName = "compute-image-tools-test"
	success    = "SUCCESS"
)

var (
	artifactsDir = os.Getenv("ARTIFACTS")
	gcsURLBase   = getBase()
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

func gcsWrite(ctx context.Context, client *storage.Client, p string, data []byte, dataR io.Reader, ct string) error {
	var err error
	w := client.Bucket(bucketName).Object(path.Join(gcsURLBase, p)).NewWriter(ctx)
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

func started(args ...string) ([]byte, error) {
	s := map[string]interface{}{
		"timestamp": time.Now().Unix(),
	}

	md := map[string]string{}
	for i, arg := range args {
		md[fmt.Sprintf("arg_%d", i+1)] = arg
	}
	if len(md) > 0 {
		s["metadata"] = md
	}

	switch jobType {
	case "batch":
		s["pull"] = pullRefs
	case "periodic":
	case "postsubmit":
		s["pull"] = pullRefs
	case "presubmit":
		s["pull"] = pullRefs
		s["repos"] = map[string]string{fmt.Sprintf("%s/%s", repoOwner, repoName): pullSHA}
	}

	return json.Marshal(s)
}

type gcsLogger struct {
	client *storage.Client
	object string
	buf    *bytes.Buffer
	ctx    context.Context
}

func (l *gcsLogger) Write(b []byte) (int, error) {
	if l.buf == nil {
		l.buf = new(bytes.Buffer)
	}
	l.buf.Write(b)

	return len(b), gcsWrite(l.ctx, l.client, l.object, l.buf.Bytes(), nil, "text/plain")
}

func runCommand(cmd *exec.Cmd, buildLog *log.Logger) (string, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer pr.Close()

	cmd.Stdout, cmd.Stderr = pw, pw

	if err := cmd.Start(); err != nil {
		return "", err
	}
	pw.Close()

	in := bufio.NewScanner(pr)
	for in.Scan() {
		buildLog.Println(in.Text())
	}

	if err := cmd.Wait(); err != nil {
		buildLog.Println("ERROR: ", err)
	}

	if cmd.ProcessState.Success() {
		return success, nil
	}
	return "FAILURE", nil
}

func main() {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	logfile := &gcsLogger{client: client, object: "build-log.txt", ctx: ctx}
	buffered := bufio.NewWriter(logfile)
	buildLog = log.New(io.MultiWriter(buffered, os.Stdout), "", 0)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if err := buffered.Flush(); err != nil {
				buffered.Reset(logfile)
				buildLog.Println("error flushing logger: ", err)
			}
		}
	}()

	// Write started.json
	data, err := started(os.Args[2:]...)
	if err != nil {
		log.Fatal(err)
	}
	buildLog.Println("Writing started.json to GCS.")
	if err := gcsWrite(ctx, client, "started.json", data, nil, "application/json"); err != nil {
		buildLog.Println("ERROR: ", err)
	}

	// Run the main process.
	buildLog.Println("Running wrapped logic.")
	result, err := runCommand(exec.Command(os.Args[1], os.Args[2:]...), buildLog)
	if err != nil {
		buildLog.Println("ERROR: ", err)
		buffered.Flush()
		os.Exit(1)
	}
	buildLog.Printf("Run logic finished with result %q.", result)

	// Copy artifacts.
	buildLog.Println("Writing artifacts to GCS.")
	filepath.Walk(artifactsDir, func(p string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			buildLog.Println("No artifacts to write")
			return nil
		}
		if err != nil {
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
			ft := ""
			if filepath.Ext(p) == ".xml" {
				ft = "application/xml"
			}
			if err := gcsWrite(ctx, client, p, nil, f, ft); err != nil {
				buildLog.Println("ERROR: ", err)
			}
		}
		return nil
	})

	// Write finished.json
	buildLog.Println("Writing finished.json to GCS.")
	if err := gcsWrite(ctx, client, "finished.json", finished(result), nil, "application/json"); err != nil {
		buildLog.Println("ERROR: ", err)
	}

	buffered.Flush()

	if result != success {
		os.Exit(1)
	}
}
