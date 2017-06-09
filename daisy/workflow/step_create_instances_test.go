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
	"errors"
	"path"
	"sort"
	"testing"

	daisy_compute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateInstanceProcessDisks(t *testing.T) {
	w := testWorkflow()
	validatedDisks.add(w, "d1")
	validatedDisks.add(w, "d2")

	tests := []struct {
		desc       string
		ad, wantAd []*compute.AttachedDisk
		shouldErr  bool
	}{
		{"normal case", []*compute.AttachedDisk{{Source: "d1"}}, []*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: "READ_WRITE"}}, false},
		{"multiple disks case", []*compute.AttachedDisk{{Source: "d1"}, {Source: "d2"}}, []*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: "READ_WRITE"}, {Boot: false, Source: "d2", Mode: "READ_WRITE"}}, false},
		{"mode specified case", []*compute.AttachedDisk{{Source: "d1", Mode: "READ_ONLY"}}, []*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: "READ_ONLY"}}, false},
		{"bad mode specified case", []*compute.AttachedDisk{{Source: "d1", Mode: "FOO"}}, nil, true},
		{"no disks case", []*compute.AttachedDisk{}, nil, true},
		{"disk dne case", []*compute.AttachedDisk{{Source: "dne"}}, nil, true},
	}

	for _, tt := range tests {
		ci := CreateInstance{Instance: compute.Instance{Disks: tt.ad}}
		err := ci.processDisks(w)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: processDisks should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: processDisks returned an unexpected error: %v", tt.desc, err)
		} else if diff := pretty.Compare(tt.ad, tt.wantAd); err == nil && diff != "" {
			t.Errorf("%s: AttachedDisks not modified as expected: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateInstanceProcessMachineType(t *testing.T) {
	tests := []struct {
		desc, mt, wantMt string
		shouldErr        bool
	}{
		{"normal case", "mt", "zones/foo/machineTypes/mt", false},
		{"bad machine type case", "garbage/url", "", true},
		{"bad machine type case 2", "illegal-machine-type-name!!!", "", true},
	}

	for _, tt := range tests {
		ci := CreateInstance{Instance: compute.Instance{MachineType: tt.mt}, Zone: "foo"}
		err := ci.processMachineType()
		if tt.shouldErr && err == nil {
			t.Errorf("%s: processMachineType should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: processMachineType returned an unexpected error: %v", tt.desc, err)
		} else if err == nil && ci.MachineType != tt.wantMt {
			t.Errorf("%s: MachineType not modified as expected: got: %q, want: %q", tt.desc, ci.MachineType, tt.wantMt)
		}
	}
}

func TestCreateInstanceProcessMetadata(t *testing.T) {
	w := testWorkflow()
	w.populate()
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
		err := ci.processMetadata(w)
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: processMetadata should have erred but didn't", tt.desc)
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
			t.Errorf("%s: processMetadata returned an unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstanceProcessNetworks(t *testing.T) {
	defaultAcs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	tests := []struct {
		desc        string
		input, want []*compute.NetworkInterface
		shouldErr   bool
	}{
		{"default case", nil, []*compute.NetworkInterface{{Network: "global/networks/default", AccessConfigs: defaultAcs}}, false},
		{"default AccessConfig case", []*compute.NetworkInterface{{Network: "global/networks/foo"}}, []*compute.NetworkInterface{{Network: "global/networks/foo", AccessConfigs: defaultAcs}}, false},
		{"network URL resolution case", []*compute.NetworkInterface{{Network: "foo", AccessConfigs: []*compute.AccessConfig{}}}, []*compute.NetworkInterface{{Network: "global/networks/foo", AccessConfigs: []*compute.AccessConfig{}}}, false},
		{"bad network case", []*compute.NetworkInterface{{Network: "bad network!"}}, nil, true},
	}

	for _, tt := range tests {
		ci := &CreateInstance{Instance: compute.Instance{NetworkInterfaces: tt.input}}
		err := ci.processNetworks()
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error", tt.desc)
			} else if diff := pretty.Compare(ci.NetworkInterfaces, tt.want); diff != "" {
				t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc, diff)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestCreateInstanceProcessScopes(t *testing.T) {
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
		err := ci.processScopes()
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
	var createErr error
	w := testWorkflow()
	w.ComputeClient.(*daisy_compute.TestClient).CreateInstanceFn = func(p, z string, i *compute.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	s := &Step{w: w}
	w.Sources = map[string]string{"file": "gs://some/file"}
	disks[w].m = map[string]*resource{
		"d0": {name: "d0", real: w.genName("d0"), link: "diskLink0"},
	}

	// Good case: check disk link gets resolved. Check instance reference map updates.
	i0 := &CreateInstance{daisyName: "i0", Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}}
	i1 := &CreateInstance{daisyName: "i1", Project: "foo", Zone: "bar", Instance: compute.Instance{Name: "realI1", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "other"}}}}
	ci := &CreateInstances{i0, i1}
	if err := ci.run(s); err != nil {
		t.Errorf("unexpected error running CreateInstances.run(): %v", err)
	}
	if i0.Disks[0].Source != disks[w].m["d0"].link {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", disks[w].m["d0"].link, i0.Disks[0].Source)
	}
	if i1.Disks[0].Source != "other" {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", "other", i1.Disks[0].Source)
	}
	wantM := map[string]*resource{
		"i0": {name: "i0", real: "realI0", link: i0.SelfLink},
		"i1": {name: "i1", real: "realI1", link: i1.SelfLink},
	}
	if diff := pretty.Compare(instances[w].m, wantM); diff != "" {
		t.Errorf("instanceRefs do not match expectation: (-got +want)\n%s", diff)
	}

	// Bad case: compute client CreateInstance error. Check instance ref map doesn't update.
	instances[w].m = map[string]*resource{}
	createErr = errors.New("client error")
	ci = &CreateInstances{
		{daisyName: "i0", Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}},
	}
	if err := ci.run(s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
	if diff := pretty.Compare(instances[w].m, map[string]*resource{}); diff != "" {
		t.Errorf("instanceRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateInstancesValidate(t *testing.T) {
	w := testWorkflow()
	validatedDisks.add(w, "d1")
	ad := []*compute.AttachedDisk{{Source: "d1"}}

	tests := []struct {
		desc        string
		input, want *CreateInstance
		shouldErr   bool
	}{
		{"good normal case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: ad}}, &CreateInstance{Instance: compute.Instance{Name: w.genName("foo")}, Project: w.Project, Zone: w.Zone, daisyName: "foo"}, false},
		{"nondefault zone/project case", &CreateInstance{Instance: compute.Instance{Name: "bar", Disks: ad}, Project: "p", Zone: "z"}, &CreateInstance{Instance: compute.Instance{Name: w.genName("bar")}, Project: "p", Zone: "z", daisyName: "bar"}, false},
		{"exact name case", &CreateInstance{Instance: compute.Instance{Name: "baz", Disks: ad}, ExactName: true}, &CreateInstance{Instance: compute.Instance{Name: "baz"}, Project: w.Project, Zone: w.Zone, daisyName: "baz"}, false},
		{"bad dupe case", &CreateInstance{Instance: compute.Instance{Name: "foo", Disks: ad}}, nil, true},
		{"bad processing (no disks) case", &CreateInstance{Instance: compute.Instance{Name: "gaz"}}, nil, true},
	}

	for _, tt := range tests {
		s := &Step{name: "s", w: w, CreateInstances: &CreateInstances{tt.input}}
		err := s.validate()
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error", tt.desc)
			} else if tt.input.daisyName != tt.want.daisyName {
				t.Errorf("%s: incorrect internal daisy name: got: %q, want: %q", tt.desc, tt.input.daisyName, tt.want.daisyName)
			} else if tt.input.Name != tt.want.Name {
				t.Errorf("%s: incorrect real name: got: %q, want: %q", tt.desc, tt.input.Name, tt.want.Name)
			} else if tt.input.Project != tt.want.Project {
				t.Errorf("%s: incorrect project: got: %q, want: %q", tt.desc, tt.input.Project, tt.want.Project)
			} else if tt.input.Zone != tt.want.Zone {
				t.Errorf("%s: incorrect internal daisy name: got: %q, want: %q", tt.desc, tt.input.daisyName, tt.want.daisyName)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error %v", tt.desc, err)
		}
	}
}
