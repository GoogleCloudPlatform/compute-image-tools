//  Copyright 2018 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//	  http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	tmpFolder   = ""
	errNonFatal = errors.New("method succeeded with errors")
)

type runner interface {
	run() (string, error)
}

type logFolder struct {
	name  string
	files []string
}

func zipFiles(logs []logFolder, outputPath string) (err error) {
	newFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		// This takes priority over the non-fatal errors
		if cErr := newFile.Close(); cErr != nil && (err == nil || err == errNonFatal) {
			err = cErr
		}
	}()

	writer := zip.NewWriter(newFile)
	defer writer.Close()

	err = nil
	for _, folder := range logs {
		for _, path := range folder.files {
			file, zErr := os.Open(path)
			if zErr != nil {
				log.Printf("Error opening file %s for zipping with error %v\n", path, err)
				err = errNonFatal
				continue
			}
			defer func() {
				if cErr := file.Close(); cErr != nil {
					err = errNonFatal
				}
			}()

			p := fmt.Sprintf("%s/%s", folder.name, filepath.Base(path))
			zf, zErr := writer.Create(p)
			if zErr != nil {
				log.Printf("Error saving file %s to zip with error %v\n", path, err)
				err = errNonFatal
				continue
			}

			if _, zErr = io.Copy(zf, file); zErr != nil {
				log.Printf("Error saving contents of file %s with error %v\n", path, err)
				err = errNonFatal
			}
		}
	}
	return err
}

func uploadToSignedURL(uploadPath string, signedURL string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// The signed Url gives us the actual url to upload to
	req, err := http.NewRequest("POST", signedURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-goog-resumable", "start")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	uploadURL := resp.Header.Get("Location")

	// Upload the file
	f, err := os.Open(uploadPath)
	if err != nil {
		return err
	}
	bodyReader, bodyWriter := io.Pipe()
	go func() {
		defer bodyWriter.Close()
		defer f.Close()
		io.Copy(bodyWriter, f)
	}()

	req, err = http.NewRequest("PUT", uploadURL, bodyReader)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	return err
}

func moveZipFile(path string) (string, error) {
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	knownZipPath := filepath.Join(currDir, filepath.Base(path))
	return knownZipPath, os.Rename(path, knownZipPath)
}

func main() {
	var err error
	tmpFolder, err = os.MkdirTemp("", "diagnostics")
	if err != nil {
		log.Fatal("Error creating a temporary folder. Exiting")
	}

	signedURL := flag.String("signedUrl", "", "The Signed Url to upload the zipped logs to.")
	traceFlag := flag.Bool("trace", false, "Take a 10 minute trace of the system using wpr.")
	flag.Parse()

	nonFatalErrorsPresent := false
	paths, err := gatherLogs(*traceFlag)
	if err != nil {
		nonFatalErrorsPresent = true
	}

	zipFile := filepath.Join(tmpFolder, "logs.zip")
	err = zipFiles(paths, zipFile)
	if err == errNonFatal {
		nonFatalErrorsPresent = true
	} else if err != nil {
		log.Fatalf("Error zipping files: %v", err)
	}

	if *signedURL != "" {
		log.Printf("Diagnostics: logs uploading to [[%s]].", *signedURL)
		if err = uploadToSignedURL(zipFile, *signedURL); err != nil {
			log.Fatalf("Error uploading to signed url: %v. Logs can be found at %s", err, zipFile)
		}
		log.Print("Diagnostics: logs uploaded to the supplied url successfully.")
	} else {
		knownZipPath, err := moveZipFile(zipFile)
		if err != nil {
			log.Fatalf("Error moving logs to well known directory. They can be found instead at: %s", zipFile)
		}
		log.Printf("Logs can be found at %s", knownZipPath)
	}
	os.RemoveAll(tmpFolder)

	if nonFatalErrorsPresent {
		log.Fatal("Errors occured while collecting and zipping some logs.\nUnaffected logs were still packaged and available.")
	}
}
