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

package compute

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

var (
	testProject  = "test-project"
	testZone     = "test-zone"
	testDisk     = "test-disk"
	testImage    = "test-image"
	testInstance = "test-instance"
)

func TestCreateDisk(t *testing.T) {
	var getErr, insertErr, waitErr error
	var getResp *compute.Disk
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/disks?alt=json", testProject, testZone) {
			if insertErr != nil {
				w.WriteHeader(400)
				fmt.Fprintln(w, insertErr)
				return
			}
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(r.Body); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/disks/%s?alt=json", testProject, testZone, testDisk) {
			if getErr != nil {
				w.WriteHeader(400)
				fmt.Fprintln(w, getErr)
				return
			}
			body, _ := json.Marshal(getResp)
			fmt.Fprintln(w, string(body))
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()
	c.operationsWaitFn = func(project, zone, name string) error { return waitErr }

	tests := []struct {
		desc                       string
		getErr, insertErr, waitErr error
		shouldErr                  bool
	}{
		{"normal case", nil, nil, nil, false},
		{"get err case", errors.New("get err"), nil, nil, true},
		{"insert err case", nil, errors.New("insert err"), nil, true},
		{"wait err case", nil, nil, errors.New("wait err"), true},
	}

	for _, tt := range tests {
		getErr, insertErr, waitErr = tt.getErr, tt.insertErr, tt.waitErr
		d := &compute.Disk{Name: testDisk}
		getResp = &compute.Disk{Name: testDisk, SelfLink: "foo"}
		err := c.CreateDisk(testProject, testZone, d)
		getResp.ServerResponse = d.ServerResponse // We have to fudge this part in order to check that d == getResp
		if err != nil && !tt.shouldErr {
			t.Errorf("%s: got unexpected error: %s", tt.desc, err)
		} else if diff := pretty.Compare(d, getResp); err == nil && diff != "" {
			t.Errorf("%s: Disk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateImage(t *testing.T) {
	var getErr, insertErr, waitErr error
	var getResp *compute.Image
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/%s/global/images?alt=json", testProject) {
			if insertErr != nil {
				w.WriteHeader(400)
				fmt.Fprintln(w, insertErr)
				return
			}
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(r.Body); err != nil {
				t.Fatal(err)
			}
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/global/images/%s?alt=json", testProject, testImage) {
			if getErr != nil {
				w.WriteHeader(400)
				fmt.Fprintln(w, getErr)
				return
			}
			body, _ := json.Marshal(getResp)
			fmt.Fprintln(w, string(body))
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()
	c.operationsWaitFn = func(project, zone, name string) error { return waitErr }

	tests := []struct {
		desc                       string
		getErr, insertErr, waitErr error
		shouldErr                  bool
	}{
		{"normal case", nil, nil, nil, false},
		{"get err case", errors.New("get err"), nil, nil, true},
		{"insert err case", nil, errors.New("insert err"), nil, true},
		{"wait err case", nil, nil, errors.New("wait err"), true},
	}

	for _, tt := range tests {
		getErr, insertErr, waitErr = tt.getErr, tt.insertErr, tt.waitErr
		i := &compute.Image{Name: testImage}
		getResp = &compute.Image{Name: testImage, SelfLink: "foo"}
		err := c.CreateImage(testProject, i)
		getResp.ServerResponse = i.ServerResponse // We have to fudge this part in order to check that i == getResp
		if err != nil && !tt.shouldErr {
			t.Errorf("%s: got unexpected error: %s", tt.desc, err)
		} else if diff := pretty.Compare(i, getResp); err == nil && diff != "" {
			t.Errorf("%s: Image does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateInstance(t *testing.T) {
	var getErr, insertErr, waitErr error
	var getResp *compute.Instance
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances?alt=json", testProject, testZone) {
			if insertErr != nil {
				w.WriteHeader(400)
				return
			}
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(r.Body); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances/%s?alt=json", testProject, testZone, testInstance) {
			if getErr != nil {
				w.WriteHeader(400)
				fmt.Fprintln(w, getErr)
				return
			}
			body, _ := json.Marshal(getResp)
			fmt.Fprintln(w, string(body))
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()
	c.operationsWaitFn = func(project, zone, name string) error { return waitErr }

	tests := []struct {
		desc                       string
		getErr, insertErr, waitErr error
		shouldErr                  bool
	}{
		{"normal case", nil, nil, nil, false},
		{"get err case", errors.New("get err"), nil, nil, true},
		{"insert err case", nil, errors.New("insert err"), nil, true},
		{"wait err case", nil, nil, errors.New("wait err"), true},
	}

	for _, tt := range tests {
		getErr, insertErr, waitErr = tt.getErr, tt.insertErr, tt.waitErr
		i := &compute.Instance{Name: testInstance}
		getResp = &compute.Instance{Name: testInstance, SelfLink: "foo"}
		err := c.CreateInstance(testProject, testZone, i)
		getResp.ServerResponse = i.ServerResponse // We have to fudge this part in order to check that i == getResp
		if err != nil && !tt.shouldErr {
			t.Errorf("%s: got unexpected error: %s", tt.desc, err)
		} else if diff := pretty.Compare(i, getResp); err == nil && diff != "" {
			t.Errorf("%s: Instance does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestDeleteDisk(t *testing.T) {
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/disks/%s?alt=json", testProject, testZone, testDisk) {
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/operations/?alt=json", testProject, testZone) {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	if err := c.DeleteDisk(testProject, testZone, testDisk); err != nil {
		t.Fatalf("error running DeleteDisk: %v", err)
	}
}

func TestDeleteImage(t *testing.T) {
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/%s/global/images/%s?alt=json", testProject, testImage) {
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/global/operations/?alt=json", testProject) {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	if err := c.DeleteImage(testProject, testImage); err != nil {
		t.Fatalf("error running DeleteImage: %v", err)
	}
}

func TestDeleteInstance(t *testing.T) {
	svr, c, err := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances/%s?alt=json", testProject, testZone, testInstance) {
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/operations/?alt=json", testProject, testZone) {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	if err := c.DeleteInstance(testProject, testZone, testInstance); err != nil {
		t.Fatalf("error running DeleteInstance: %v", err)
	}
}
