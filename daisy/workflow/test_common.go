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

package workflow

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

type mockStep struct {
	runImpl      func(*Workflow) error
	validateImpl func(*Workflow) error
}

func (m *mockStep) run(w *Workflow) error {
	if m.runImpl != nil {
		return m.runImpl(w)
	}
	return nil
}

func (m *mockStep) validate(w *Workflow) error {
	if m.validateImpl != nil {
		return m.validateImpl(w)
	}
	return nil
}

var (
	testGCEClient *compute.Client
	testGCSClient *storage.Client
	testGCSDNEVal = "dne"
	testID        = randString(5)
	testWf        = "test-wf"
	testProject   = "test-project"
	testZone      = "test-zone"
	testBucket    = "test-bucket"
)

func init() {
	var err error
	testGCEClient, err = newTestGCEClient()
	if err != nil {
		panic(err)
	}
	testGCSClient, err = newTestGCSClient()
	if err != nil {
		panic(err)
	}
}

func testWorkflow() *Workflow {
	return &Workflow{
		Name:          testWf,
		Bucket:        testBucket,
		Project:       testProject,
		Zone:          testZone,
		ComputeClient: testGCEClient,
		StorageClient: testGCSClient,
		id:            testID,
		Ctx:           context.Background(),
		diskRefs:      &refMap{},
		imageRefs:     &refMap{},
		instanceRefs:  &refMap{},
	}
}

func newTestGCEClient() (*compute.Client, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"TERMINATED","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/machineTypes", testProject, testZone)) {
			fmt.Fprintln(w, `{"Items":[{"Name": "foo-type"}]}`)
		} else {
			fmt.Fprintln(w, `{"Status":"DONE","SelfLink":"link"}`)
		}
	}))

	return compute.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
}

func newTestGCSClient() (*storage.Client, error) {
	nameRgx := regexp.MustCompile(`"name":"([^"].*)"`)
	rewriteRgx := regexp.MustCompile("/b/([^/]+)/o/([^/]+)/rewriteTo/b/([^/]+)/o/([^?]+)")
	uploadRgx := regexp.MustCompile("/b/([^/]+)/o?.*uploadType=multipart.*")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		m := r.Method

		if match := uploadRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			body, _ := ioutil.ReadAll(r.Body)
			n := nameRgx.FindStringSubmatch(string(body))[1]
			fmt.Fprintf(w, `{"kind":"storage#object","bucket":"%s","name":"%s"}\n`, match[1], n)
		} else if match := rewriteRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			if strings.Contains(match[1], testGCSDNEVal) || strings.Contains(match[2], testGCSDNEVal) {
				w.WriteHeader(http.StatusNotFound)
			}
			o := fmt.Sprintf(`{"bucket":"%s","name":"%s"}`, match[3], match[4])
			fmt.Fprintf(w, `{"kind": "storage#rewriteResponse", "done": true, "objectSize": "1", "totalBytesRewritten": "1", "resource": %s}\n`, o)
		} else {
			fmt.Println("got something else")
		}
	}))

	return storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
}
