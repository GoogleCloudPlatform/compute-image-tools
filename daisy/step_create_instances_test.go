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
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestLogSerialOutput(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	var get []string
	_, c, err := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=1") {
			if len(get) == 0 {
				fmt.Fprintln(w, `{"Contents":"test","Start":"0"}`)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			get = append(get, r.URL.String())
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=2") {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "500 error")
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=3") {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "400 error")
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i1", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"TERMINATED","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i2", testProject, testZone)) {
			fmt.Fprintln(w, `{"Status":"RUNNING","SelfLink":"link"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), fmt.Sprintf("/%s/zones/%s/instances/i3", testProject, testZone)) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "test error")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad request: %+v", r)
		}
		get = append(get, r.URL.String())
	}))
	if err != nil {
		t.Fatal(err)
	}
	w.ComputeClient = c
	w.bucket = "test-bucket"

	instances[w].m = map[string]*resource{
		"i1": {real: w.genName("i1"), link: "link"},
		"i2": {real: w.genName("i2"), link: "link"},
		"i3": {real: w.genName("i3"), link: "link"},
	}

	var buf bytes.Buffer
	w.logger = log.New(&buf, "", 0)

	tests := []struct {
		test, want, name string
		port             int64
		get              []string // Test expected api call flow.
	}{
		{
			"400 error but instance stopped",
			"CreateInstances: streaming instance \"i1\" serial port 2 output to gs://test-bucket/i1-serial-port2.log\n",
			"i1",
			2,
			[]string{"/test-project/zones/test-zone/instances/i1/serialPort?alt=json&port=2&start=0", "/test-project/zones/test-zone/instances/i1?alt=json"},
		},
		{
			"400 error but instance running",
			"CreateInstances: streaming instance \"i2\" serial port 3 output to gs://test-bucket/i2-serial-port3.log\nCreateInstances: instance \"i2\": error getting serial port: googleapi: got HTTP response code 400 with body: 400 error\n",
			"i2",
			3,
			[]string{"/test-project/zones/test-zone/instances/i2/serialPort?alt=json&port=3&start=0", "/test-project/zones/test-zone/instances/i2?alt=json"},
		},
		{
			"500 error but instance running",
			"CreateInstances: streaming instance \"i2\" serial port 2 output to gs://test-bucket/i2-serial-port2.log\nCreateInstances: instance \"i2\": error getting serial port: googleapi: got HTTP response code 500 with body: 500 error\n",
			"i2",
			2,
			[]string{"/test-project/zones/test-zone/instances/i2/serialPort?alt=json&port=2&start=0", "/test-project/zones/test-zone/instances/i2?alt=json", "/test-project/zones/test-zone/instances/i2/serialPort?alt=json&port=2&start=0", "/test-project/zones/test-zone/instances/i2?alt=json", "/test-project/zones/test-zone/instances/i2/serialPort?alt=json&port=2&start=0", "/test-project/zones/test-zone/instances/i2?alt=json", "/test-project/zones/test-zone/instances/i2/serialPort?alt=json&port=2&start=0", "/test-project/zones/test-zone/instances/i2?alt=json"},
		},
		{
			"500 error but instance deleted",
			"CreateInstances: streaming instance \"i4\" serial port 2 output to gs://test-bucket/i4-serial-port2.log\n",
			"i4",
			2,
			[]string{"/test-project/zones/test-zone/instances/i4/serialPort?alt=json&port=2&start=0"},
		},
		{
			"normal flow",
			"CreateInstances: streaming instance \"i1\" serial port 1 output to gs://test-bucket/i1-serial-port1.log\n",
			"i1",
			1,
			[]string{"/test-project/zones/test-zone/instances/i1/serialPort?alt=json&port=1&start=0", "/test-project/zones/test-zone/instances/i1/serialPort?alt=json&port=1&start=0", "/test-project/zones/test-zone/instances/i1/serialPort?alt=json&port=1&start=0", "/test-project/zones/test-zone/instances/i1/serialPort?alt=json&port=1&start=0", "/test-project/zones/test-zone/instances/i1?alt=json"},
		},
	}

	for _, tt := range tests {
		get = nil
		buf.Reset()
		logSerialOutput(ctx, w, tt.name, tt.port, 1*time.Microsecond)
		if !reflect.DeepEqual(get, tt.get) {
			t.Errorf("%s: got get calls: %q, want get calls: %q", tt.test, get, tt.get)
		}
		if buf.String() != tt.want {
			t.Errorf("%s: got: %q, want: %q", tt.test, buf.String(), tt.want)
		}
	}
}

func TestCreateInstancePopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	desc := "desc"
	defP := w.Project
	defZ := w.Zone
	defMT := fmt.Sprintf("projects/%s/zones/%s/machineTypes/n1-standard-1", defP, defZ)
	defDM := defaultDiskMode
	defDs := []*compute.AttachedDisk{{Boot: true, Source: "foo", Mode: defDM}}
	defAcs := []*compute.AccessConfig{{Type: defaultAccessConfigType}}
	defNs := []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/default", defP), AccessConfigs: defAcs}}
	defMD := map[string]string{"daisy-sources-path": "gs://", "daisy-logs-path": "gs://", "daisy-outs-path": "gs://"}
	defSs := []string{"https://www.googleapis.com/auth/devstorage.read_only"}
	defSAs := []*compute.ServiceAccount{{Email: "default", Scopes: defSs}}

	tests := []struct {
		desc      string
		ci, want  *CreateInstance
		shouldErr bool
	}{
		{
			"defaults, non exact name case",
			&CreateInstance{Instance: compute.Instance{Name: "foo", Description: desc, Disks: []*compute.AttachedDisk{{Source: "foo"}}}},
			&CreateInstance{Instance: compute.Instance{Name: w.genName("foo"), Description: desc, Disks: defDs, MachineType: defMT, NetworkInterfaces: defNs, ServiceAccounts: defSAs}, Metadata: defMD, Scopes: defSs, Project: defP, Zone: defZ, daisyName: "foo"},
			false,
		},
		{
			"nondefault zone/project case",
			&CreateInstance{
				Instance: compute.Instance{Name: "foo", Description: desc, Disks: []*compute.AttachedDisk{{Source: "foo"}}},
				Project:  "pfoo", Zone: "zfoo", ExactName: true,
			},
			&CreateInstance{
				Instance: compute.Instance{
					Name: "foo", Description: desc,
					Disks:             []*compute.AttachedDisk{{Boot: true, Source: "foo", Mode: defDM}},
					MachineType:       "projects/pfoo/zones/zfoo/machineTypes/n1-standard-1",
					NetworkInterfaces: []*compute.NetworkInterface{{Network: "projects/pfoo/global/networks/default", AccessConfigs: defAcs}},
					ServiceAccounts:   defSAs,
				},
				Metadata: defMD, Scopes: defSs, Project: "pfoo", Zone: "zfoo", daisyName: "foo", ExactName: true,
			},
			false,
		},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		s.CreateInstances = &CreateInstances{tt.ci}
		err := s.CreateInstances.populate(ctx, s)
		if tt.shouldErr {
			if err == nil {
				t.Errorf("%s: should have returned error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else {
			tt.ci.Instance.Metadata = nil // This is undeterministic, but we can check tt.input.Metadata.
			if diff := pretty.Compare(tt.ci, tt.want); diff != "" {
				t.Errorf("%s: CreateInstance not modified as expected: (-got +want)\n%s", tt.desc, diff)
			}
		}
	}
}

func TestCreateInstancePopulateDisks(t *testing.T) {
	w := testWorkflow()

	iName := "foo"
	defDT := fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", testProject, testZone, defaultDiskType)
	tests := []struct {
		desc       string
		ad, wantAd []*compute.AttachedDisk
	}{
		{
			"normal case",
			[]*compute.AttachedDisk{{Source: "d1"}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode}},
		},
		{
			"multiple disks case",
			[]*compute.AttachedDisk{{Source: "d1"}, {Source: "d2"}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode}, {Boot: false, Source: "d2", Mode: defaultDiskMode}},
		},
		{
			"mode specified case",
			[]*compute.AttachedDisk{{Source: "d1", Mode: diskModeRO}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: diskModeRO}},
		},
		{
			"init params daisy image (and other defaults)",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true}},
		},
		{
			"init params image short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "global/images/i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true}},
		},
		{
			"init params image extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true}},
		},
		{
			"init params disk type short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("zones/%s/diskTypes/dt", testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true}},
		},
		{
			"init params disk type extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true}},
		},
		{
			"init params name suffixes",
			[]*compute.AttachedDisk{
				{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i"}},
				{Source: "d"},
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i"}},
				{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i"}},
			},
			[]*compute.AttachedDisk{
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true},
				{Source: "d", Mode: defaultDiskMode},
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode},
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: fmt.Sprintf("%s-2", iName), SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode},
			},
		},
	}

	for _, tt := range tests {
		ci := CreateInstance{Instance: compute.Instance{Name: iName, Disks: tt.ad}, Project: testProject, Zone: testZone}
		err := ci.populateDisks(w)
		if err != nil {
			t.Errorf("%s: populateDisks returned an unexpected error: %v", tt.desc, err)
		} else if diff := pretty.Compare(tt.ad, tt.wantAd); diff != "" {
			t.Errorf("%s: AttachedDisks not modified as expected: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateInstancePopulateMachineType(t *testing.T) {
	tests := []struct {
		desc, mt, wantMt string
		shouldErr        bool
	}{
		{"normal case", "mt", "projects/foo/zones/bar/machineTypes/mt", false},
		{"expand case", "zones/bar/machineTypes/mt", "projects/foo/zones/bar/machineTypes/mt", false},
	}

	for _, tt := range tests {
		ci := CreateInstance{Instance: compute.Instance{MachineType: tt.mt}, Project: "foo", Zone: "bar"}
		err := ci.populateMachineType()
		if tt.shouldErr && err == nil {
			t.Errorf("%s: populateMachineType should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: populateMachineType returned an unexpected error: %v", tt.desc, err)
		} else if err == nil && ci.MachineType != tt.wantMt {
			t.Errorf("%s: MachineType not modified as expected: got: %q, want: %q", tt.desc, ci.MachineType, tt.wantMt)
		}
	}
}

func TestCreateInstancePopulateMetadata(t *testing.T) {
	w := testWorkflow()
	w.populate(context.Background())
	w.Sources = map[string]string{"file": "foo/bar"}
	filePath := "gs://" + path.Join(w.bucket, w.sourcesPath, "file")

	baseMd := map[string]string{
		"daisy-sources-path": "gs://" + path.Join(w.bucket, w.sourcesPath),
		"daisy-logs-path":    "gs://" + path.Join(w.bucket, w.logsPath),
		"daisy-outs-path":    "gs://" + path.Join(w.bucket, w.outsPath),
	}
	getWantMd := func(md map[string]string) *compute.Metadata {
		for k, v := range baseMd {
			md[k] = v
		}
		result := &compute.Metadata{}
		for k, v := range md {
			vCopy := v
			result.Items = append(result.Items, &compute.MetadataItems{Key: k, Value: &vCopy})
		}
		return result
	}

	tests := []struct {
		desc          string
		md            map[string]string
		startupScript string
		wantMd        *compute.Metadata
		shouldErr     bool
	}{
		{"defaults case", nil, "", getWantMd(map[string]string{}), false},
		{"startup script case", nil, "file", getWantMd(map[string]string{"startup-script-url": filePath, "windows-startup-script-url": filePath}), false},
		{"bad startup script case", nil, "foo", nil, true},
	}

	for _, tt := range tests {
		ci := CreateInstance{Metadata: tt.md, StartupScript: tt.startupScript}
		err := ci.populateMetadata(w)
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: populateMetadata should have erred but didn't", tt.desc)
			} else {
				compFactory := func(items []*compute.MetadataItems) func(i, j int) bool {
					return func(i, j int) bool { return items[i].Key < items[j].Key }
				}
				sort.Slice(ci.Instance.Metadata.Items, compFactory(ci.Instance.Metadata.Items))
				sort.Slice(tt.wantMd.Items, compFactory(tt.wantMd.Items))
				if diff := pretty.Compare(ci.Instance.Metadata, tt.wantMd); diff != "" {
					t.Errorf("%s: Metadata not modified as expected: (-got +want)\n%s", tt.desc, diff)
				}
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: populateMetadata returned an unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstancePopulateNetworks(t *testing.T) {
	defaultAcs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	tests := []struct {
		desc        string
		input, want []*compute.NetworkInterface
	}{
		{"default case", nil, []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/default", testProject), AccessConfigs: defaultAcs}}},
		{"default AccessConfig case", []*compute.NetworkInterface{{Network: "global/networks/foo"}}, []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/foo", testProject), AccessConfigs: defaultAcs}}},
		{"network URL resolution case", []*compute.NetworkInterface{{Network: "foo", AccessConfigs: []*compute.AccessConfig{}}}, []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/foo", testProject), AccessConfigs: []*compute.AccessConfig{}}}},
	}

	for _, tt := range tests {
		ci := &CreateInstance{Instance: compute.Instance{NetworkInterfaces: tt.input}, Project: testProject}
		err := ci.populateNetworks()
		if err != nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if diff := pretty.Compare(ci.NetworkInterfaces, tt.want); diff != "" {
			t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateInstancePopulateScopes(t *testing.T) {
	defaultScopes := []string{"https://www.googleapis.com/auth/devstorage.read_only"}
	tests := []struct {
		desc           string
		input          []string
		inputSas, want []*compute.ServiceAccount
		shouldErr      bool
	}{
		{"default case", nil, nil, []*compute.ServiceAccount{{Email: "default", Scopes: defaultScopes}}, false},
		{"nondefault case", []string{"foo"}, nil, []*compute.ServiceAccount{{Email: "default", Scopes: []string{"foo"}}}, false},
		{"service accounts override case", []string{"foo"}, []*compute.ServiceAccount{}, []*compute.ServiceAccount{}, false},
	}

	for _, tt := range tests {
		ci := &CreateInstance{Scopes: tt.input, Instance: compute.Instance{ServiceAccounts: tt.inputSas}}
		err := ci.populateScopes()
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error", tt.desc)
			} else if diff := pretty.Compare(ci.ServiceAccounts, tt.want); diff != "" {
				t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc, diff)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstancesRun(t *testing.T) {
	ctx := context.Background()
	var createErr error
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceFn = func(p, z string, i *compute.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	s := &Step{w: w}
	w.Sources = map[string]string{"file": "gs://some/file"}
	disks[w].m = map[string]*resource{
		"d0": {real: w.genName("d0"), link: "diskLink0"},
	}

	// Good case: check disk link gets resolved. Check instance reference map updates.
	i0 := &CreateInstance{daisyName: "i0", Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}}
	i1 := &CreateInstance{daisyName: "i1", Project: "foo", Zone: "bar", Instance: compute.Instance{Name: "realI1", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "other"}}}}
	ci := &CreateInstances{i0, i1}
	if err := ci.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateInstances.run(): %v", err)
	}
	if i0.Disks[0].Source != disks[w].m["d0"].link {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", disks[w].m["d0"].link, i0.Disks[0].Source)
	}
	if i1.Disks[0].Source != "other" {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", "other", i1.Disks[0].Source)
	}

	// Bad case: compute client CreateInstance error. Check instance ref map doesn't update.
	instances[w].m = map[string]*resource{}
	createErr = errors.New("client error")
	ci = &CreateInstances{
		{daisyName: "i0", Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}},
	}
	if err := ci.run(ctx, s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
}

func TestCreateInstanceValidateDisks(t *testing.T) {
	// Test:
	// - good case
	// - no disks bad case
	// - bad disk mode case
	ctx := context.Background()
	w := testWorkflow()
	disks[w].m = map[string]*resource{"d": {link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}}
	m := defaultDiskMode

	tests := []struct {
		desc      string
		ci        *CreateInstance
		shouldErr bool
	}{
		{"good case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: []*compute.AttachedDisk{{Source: "d", Mode: m}}}, Project: testProject, Zone: testZone}, false},
		{"good case 2", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone), Mode: m}}}, Project: testProject, Zone: testZone}, false},
		{"bad no disks case", &CreateInstance{Instance: compute.Instance{Name: "foo"}}, true},
		{"bad disk mode case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: []*compute.AttachedDisk{{Source: "d", Mode: "bad mode!"}}}, Project: testProject, Zone: testZone}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		s.CreateInstances = &CreateInstances{tt.ci}
		if err := tt.ci.validateDisks(ctx, s); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstanceValidateDiskSource(t *testing.T) {
	// Test:
	// - good case
	// - disk dne
	// - disk has wrong project/zone
	w := testWorkflow()
	disks[w].m = map[string]*resource{"d": {link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}}
	m := defaultDiskMode
	p := testProject
	z := testZone

	tests := []struct {
		desc      string
		ads       []*compute.AttachedDisk
		shouldErr bool
	}{
		{"good case", []*compute.AttachedDisk{{Source: "d", Mode: m}}, false},
		{"disk dne case", []*compute.AttachedDisk{{Source: "dne", Mode: m}}, true},
		{"bad project case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/bad/zones/%s/disks/d", z), Mode: m}}, true},
		{"bad zone case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/bad/disks/d", p), Mode: m}}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		ci := &CreateInstance{Instance: compute.Instance{Disks: tt.ads}, Project: p, Zone: z}
		s.CreateInstances = &CreateInstances{ci}
		err := ci.validateDiskSource(tt.ads[0], s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstanceValidateDiskInitializeParams(t *testing.T) {
	// Test:
	// - good case
	// - bad disk name
	// - duplicate disk
	// - bad source given
	// - bad disk types (wrong project/zone)
	// - check that disks are created
	w := testWorkflow()
	images[w].m = map[string]*resource{"i": {link: "iLink"}}
	dt := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-ssd", testProject, testZone)

	tests := []struct {
		desc      string
		p         *compute.AttachedDiskInitializeParams
		shouldErr bool
	}{
		{"good case", &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: dt}, false},
		{"bad disk name case", &compute.AttachedDiskInitializeParams{DiskName: "bad!", SourceImage: "i", DiskType: dt}, true},
		{"bad dupe disk case", &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: dt}, true},
		{"bad source case", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: dt}, true},
		{"bad disk type case", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: fmt.Sprintf("projects/bad/zones/%s/diskTypes/pd-ssd", testZone)}, true},
		{"bad disk type case 2", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: fmt.Sprintf("projects/%s/zones/bad/diskTypes/pd-ssd", testProject)}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		ci := &CreateInstance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{InitializeParams: tt.p}}}, Project: testProject, Zone: testZone}
		s.CreateInstances = &CreateInstances{ci}
		if err := ci.validateDiskInitializeParams(ci.Disks[0], s); err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	// Check good disks were created.
	wantCreator := w.Steps["good case"]
	wantLink := fmt.Sprintf("projects/%s/zones/%s/disks/foo", testProject, testZone)
	wantFoo := &resource{real: "foo", link: wantLink, creator: wantCreator}
	if gotFoo, ok := disks[w].m["foo"]; !ok || !reflect.DeepEqual(gotFoo, wantFoo) {
		t.Errorf("foo resource not added as expected: got: %+v, want: %+v", gotFoo, wantFoo)
	}

	// Check proper image user registrations.
	wantU := w.Steps["good case"]
	found := false
	for _, u := range images[w].m["i"].users {
		if u == wantU {
			found = true
		}
	}
	if !found {
		t.Error("good case should have been a registered user of image \"i\"")
	}
}

func TestCreateInstanceValidateMachineType(t *testing.T) {
	c, err := newTestGCEClient()
	if err != nil {
		t.Fatalf("error creating test client: %v", err)
	}

	tests := []struct {
		desc      string
		mt        string
		shouldErr bool
	}{
		{"good case", fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", testProject, testZone, testMachineType), false},
		{"bad machine type case", fmt.Sprintf("projects/%s/zones/%s/machineTypes/bad-mt", testProject, testZone), true},
		{"bad project case", fmt.Sprintf("projects/p2/zones/%s/machineTypes/%s", testZone, testMachineType), true},
		{"bad zone case", fmt.Sprintf("projects/%s/zones/z2/machineTypes/%s", testProject, testMachineType), true},
		{"bad zone case 2", "zones/z2/machineTypes/mt", true},
	}

	for _, tt := range tests {
		ci := &CreateInstance{Instance: compute.Instance{MachineType: tt.mt}, Project: testProject, Zone: testZone}
		if err := ci.validateMachineType(c); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstanceValidateNetworks(t *testing.T) {
	acs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}

	tests := []struct {
		desc      string
		nis       []*compute.NetworkInterface
		shouldErr bool
	}{
		{"good case", []*compute.NetworkInterface{{Network: "projects/p/global/networks/n", AccessConfigs: acs}}, false},
		{"good case 2", []*compute.NetworkInterface{{Network: "projects/p/global/networks/n", AccessConfigs: acs}}, false},
		{"bad name case", []*compute.NetworkInterface{{Network: "projects/p/global/networks/bad!", AccessConfigs: acs}}, true},
		{"bad project case", []*compute.NetworkInterface{{Network: "projects/bad-project/global/networks/n", AccessConfigs: acs}}, true},
	}

	for _, tt := range tests {
		ci := &CreateInstance{Instance: compute.Instance{NetworkInterfaces: tt.nis}, Project: "p"}
		if err := ci.validateNetworks(); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstancesValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	_, c, err := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.String() == "/p/zones/z?alt=json" {
			fmt.Fprintln(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == "/p/zones/z/machineTypes/mt?alt=json" {
			fmt.Fprintln(w, `{}`)
		} else if r.Method == "GET" && r.URL.String() == "/p?alt=json" {
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad request: %+v", r)
		}
	}))
	if err != nil {
		t.Fatalf("error creating test client: %v", err)
	}
	w.ComputeClient = c

	p := "p"
	z := "z"
	ad := []*compute.AttachedDisk{{Source: "d", Mode: defaultDiskMode}}
	mt := fmt.Sprintf("projects/%s/zones/%s/machineTypes/mt", p, z)
	dCreator := &Step{name: "dCreator", w: w}
	w.Steps["dCreator"] = dCreator
	disks[w].registerCreation("d", &resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", p, z)}, dCreator)

	tests := []struct {
		desc      string
		input     *CreateInstance
		shouldErr bool
	}{
		{"normal case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: ad, MachineType: mt}, Project: p, Zone: z}, false},
		{"bad dupe case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: ad, MachineType: mt}, Project: p, Zone: z}, true},
		{"bad name case", &CreateInstance{Instance: compute.Instance{Name: "bad!", Disks: ad, MachineType: mt}, Project: p, Zone: z}, true},
		{"bad project case", &CreateInstance{Instance: compute.Instance{Name: "bar", Disks: ad, MachineType: mt}, Project: "bad!", Zone: z}, true},
		{"bad zone case", &CreateInstance{Instance: compute.Instance{Name: "baz", Disks: ad, MachineType: mt}, Project: p, Zone: "bad!"}, true},
		{"machine type validation fails case", &CreateInstance{Instance: compute.Instance{Name: "gaz", Disks: ad, MachineType: "bad machine type!"}, Project: p, Zone: z, daisyName: "gaz"}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		w.AddDependency(tt.desc, "dCreator")
		s.CreateInstances = &CreateInstances{tt.input}
		if err := s.CreateInstances.validate(ctx, s); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
