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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var (
	testProject  = "test-project"
	testZone     = "test-zone"
	testDisk     = "test-disk"
	testImage    = "test-image"
	testInstance = "test-instance"
)

func newTestClient(handleFunc http.HandlerFunc) (*httptest.Server, *Client, error) {
	ts := httptest.NewServer(handleFunc)
	c, err := NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return nil, nil, err
	}

	return ts, c, nil
}

func testCreateDisk(got, want *compute.Disk) error {
	if got.Name != want.Name {
		return fmt.Errorf("unexpected Name, got: %s, want: %s", got.Name, want.Name)
	}
	if got.SizeGb != want.SizeGb {
		return fmt.Errorf("unexpected SizeGb, got: %s, want: %s", got.SizeGb, want.SizeGb)
	}
	if got.Type != want.Type {
		return fmt.Errorf("unexpected Type, got: %s, want: %s", got.Type, want.Type)
	}
	if got.SourceImage != want.SourceImage {
		return fmt.Errorf("unexpected SourceImage, got: %s, want: %s", got.SourceImage, want.SourceImage)
	}
	return nil
}

func TestCreateDisk(t *testing.T) {
	var body string
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/disks?alt=json", testProject, testZone) {
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(r.Body); err != nil {
				t.Fatal(err)
			}
			body = buf.String()
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/operations/?alt=json", testProject, testZone) {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/disks/%s?alt=json", testProject, testZone, testDisk) {
			fmt.Fprint(w, body)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	// Blank disk.
	want := &compute.Disk{Name: testDisk, SizeGb: 100, Type: fmt.Sprintf("zones/%s/diskTypes/pd-standard", testZone)}
	got, err := c.CreateDisk(testDisk, testProject, testZone, "", want.SizeGb, false)
	if err != nil {
		t.Fatalf("error running CreateDisk: %v", err)
	}
	if err := testCreateDisk(got, want); err != nil {
		t.Error(err)
	}

	// Test SSD and non blank disk.
	want = &compute.Disk{Name: testDisk, SizeGb: 50, Type: fmt.Sprintf("zones/%s/diskTypes/pd-ssd", testZone), SourceImage: "some-image"}
	got, err = c.CreateDisk(testDisk, testProject, testZone, "some-image", 50, true)
	if err != nil {
		t.Fatalf("error running CreateDisk: %v", err)
	}
	if err := testCreateDisk(got, want); err != nil {
		t.Error(err)
	}
}

func TestDeleteDisk(t *testing.T) {
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func testCreateImage(got, want *compute.Image) error {
	if got.Name != want.Name {
		return fmt.Errorf("unexpected Name, got: %s, want: %s", got.Name, want.Name)
	}
	if got.Family != want.Family {
		return fmt.Errorf("unexpected Family, got: %s, want: %s", got.Family, want.Family)
	}
	if got.SourceDisk != want.SourceDisk {
		return fmt.Errorf("unexpected SourceDisk, got: %s, want: %s", got.SourceDisk, want.SourceDisk)
	}
	if !reflect.DeepEqual(got.Licenses, want.Licenses) {
		return fmt.Errorf("unexpected Licenses, got: %q, want: %q", got.Licenses, want.Licenses)
	}
	if !reflect.DeepEqual(got.RawDisk, want.RawDisk) {
		return fmt.Errorf("unexpected RawDisk, got: %q, want: %q", got.RawDisk, want.RawDisk)
	}
	if !reflect.DeepEqual(got.GuestOsFeatures, want.GuestOsFeatures) {
		return fmt.Errorf("unexpected GuestOsFeatures, got: %q, want: %q", got.GuestOsFeatures, want.GuestOsFeatures)
	}
	return nil
}

func TestCreateImage(t *testing.T) {
	var body string
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.String() == fmt.Sprintf("/%s/global/images?alt=json", testProject) {
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(r.Body); err != nil {
				t.Fatal(err)
			}
			body = buf.String()
			fmt.Fprint(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/global/operations/?alt=json", testProject) {
			fmt.Fprint(w, `{"Status":"DONE"}`)
		} else if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/global/images/%s?alt=json", testProject, testImage) {
			fmt.Fprint(w, body)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	family := "somefamily"
	licenses := []string{"123456"}
	gosf := []*compute.GuestOsFeature{{Type: "somefeature"}}

	// Image from disk.
	want := &compute.Image{Name: testImage, Family: family, Licenses: licenses, GuestOsFeatures: gosf, SourceDisk: testDisk, RawDisk: &compute.ImageRawDisk{}}
	got, err := c.CreateImage(testImage, testProject, testDisk, "", family, licenses, []string{"somefeature"})
	if err != nil {
		t.Fatalf("error running CreateImage: %v", err)
	}
	if err := testCreateImage(got, want); err != nil {
		t.Error(err)
	}

	// Image from file.
	want = &compute.Image{Name: testImage, Family: family, Licenses: licenses, GuestOsFeatures: gosf, RawDisk: &compute.ImageRawDisk{Source: "some/file"}}
	got, err = c.CreateImage(testImage, testProject, "", "some/file", family, licenses, []string{"somefeature"})
	if err != nil {
		t.Fatalf("error running CreateImage: %v", err)
	}
	if err := testCreateImage(got, want); err != nil {
		t.Error(err)
	}

	// Test error.
	got, err = c.CreateImage(testImage, testProject, testDisk, "some/file", family, licenses, []string{"somefeature"})
	if err.Error() != "you must provide either a sourceDisk or a sourceFile but not both to create an image" {
		t.Errorf("did not receive expected error from CreateImage, err: %q", err)
	}
}

func TestDeleteImage(t *testing.T) {
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestWaitForInstanceStopped(t *testing.T) {
	svr, c, err := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.String() == fmt.Sprintf("/%s/zones/%s/instances/%s?alt=json", testProject, testZone, testInstance) {
			fmt.Fprint(w, `{"Status":"TERMINATED"}`)
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, r.URL)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer svr.Close()

	if err := c.WaitForInstanceStopped(testProject, testZone, testInstance); err != nil {
		t.Fatalf("error running WaitForInstanceStopped: %v", err)
	}
}
