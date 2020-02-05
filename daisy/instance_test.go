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
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strconv"
	"testing"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func TestCheckDiskMode(t *testing.T) {
	tests := []struct {
		desc, input string
		want        bool
	}{
		{"default case", defaultDiskMode, true},
		{"ro case", diskModeRO, true},
		{"rw case", diskModeRW, true},
		{"bad mode case", "bad!", false},
	}

	for _, tt := range tests {
		got := checkDiskMode(tt.input)
		if got != tt.want {
			t.Errorf("%s: want: %t, got: %t", tt.desc, got, tt.want)
		}
	}
}

func TestInstancePopulate(t *testing.T) {
	w := testWorkflow()

	// We use a bad StartupScript (the only time an error can be thrown for now), to test for proper error returning.
	tests := []struct {
		desc      string
		i         *Instance
		iBeta     *InstanceBeta
		shouldErr bool
	}{
		{"good case", &Instance{}, &InstanceBeta{}, false},
		{"bad case", &Instance{InstanceBase: InstanceBase{StartupScript: "Workflow source DNE and can't resolve!"}}, &InstanceBeta{InstanceBase: InstanceBase{StartupScript: "Workflow source DNE and can't resolve!"}}, true},
	}

	assertTest := func(shouldErr bool, desc string, err DError) {
		if shouldErr && err == nil {
			t.Errorf("%s: should have returned error but didn't", desc)
		} else if !shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", desc, err)
		}
	}
	for testNum, tt := range tests {
		s, _ := w.NewStep("s" + strconv.Itoa(testNum))
		assertTest(tt.shouldErr, tt.desc, (&tt.i.InstanceBase).populate(context.Background(), tt.i, s))
		assertTest(tt.shouldErr, tt.desc+" beta", (&tt.i.InstanceBase).populate(context.Background(), tt.i, s))
	}
}

func TestInstancePopulateDisks(t *testing.T) {
	w := testWorkflow()

	iName := "foo"
	defDT := fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", testProject, testZone, defaultDiskType)
	tests := []struct {
		desc               string
		ad, wantAd         []*compute.AttachedDisk
		adBeta, wantAdBeta []*computeBeta.AttachedDisk
	}{
		{
			"normal case",
			[]*compute.AttachedDisk{{Source: "d1"}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}},
			[]*computeBeta.AttachedDisk{{Source: "d1"}},
			[]*computeBeta.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}},
		},
		{
			"multiple disks case",
			[]*compute.AttachedDisk{{Source: "d1"}, {Source: "d2"}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}, {Boot: false, Source: "d2", Mode: defaultDiskMode, DeviceName: "d2"}},
			[]*computeBeta.AttachedDisk{{Source: "d1"}, {Source: "d2"}},
			[]*computeBeta.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}, {Boot: false, Source: "d2", Mode: defaultDiskMode, DeviceName: "d2"}},
		},
		{
			"mode specified case",
			[]*compute.AttachedDisk{{Source: "d1", Mode: diskModeRO}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: diskModeRO, DeviceName: "d1"}},
			[]*computeBeta.AttachedDisk{{Source: "d1", Mode: diskModeRO}},
			[]*computeBeta.AttachedDisk{{Boot: true, Source: "d1", Mode: diskModeRO, DeviceName: "d1"}},
		},
		{
			"init params daisy image (and other defaults)",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "i"}}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params image short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "global/images/i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "global/images/i"}}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params image extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject)}}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params disk type short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("zones/%s/diskTypes/dt", testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("zones/%s/diskTypes/dt", testZone)}}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params disk type extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}}},
			[]*computeBeta.AttachedDisk{{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
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
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName},
				{Source: "d", Mode: defaultDiskMode, DeviceName: "d"},
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, DeviceName: "foo"},
				{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: fmt.Sprintf("%s-2", iName), SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, DeviceName: fmt.Sprintf("%s-2", iName)},
			},
			[]*computeBeta.AttachedDisk{
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "i"}},
				{Source: "d"},
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i"}},
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{SourceImage: "i"}},
			},
			[]*computeBeta.AttachedDisk{
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName},
				{Source: "d", Mode: defaultDiskMode, DeviceName: "d"},
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, DeviceName: "foo"},
				{InitializeParams: &computeBeta.AttachedDiskInitializeParams{DiskName: fmt.Sprintf("%s-2", iName), SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, DeviceName: fmt.Sprintf("%s-2", iName)},
			},
		},
	}

	assertTest := func(err DError, desc string, ad, wantAd interface{}) {
		if err != nil {
			t.Errorf("%s: populateDisks returned an unexpected error: %v", desc, err)
		} else if diffRes := diff(ad, wantAd, 0); diffRes != "" {
			t.Errorf("%s: AttachedDisks not modified as expected: (-got +want)\n%s", desc, diffRes)
		}

	}
	for _, tt := range tests {
		i := Instance{Instance: compute.Instance{Name: iName, Disks: tt.ad, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(i.populateDisks(w), tt.desc, tt.ad, tt.wantAd)

		iBeta := InstanceBeta{Instance: computeBeta.Instance{Name: iName, Disks: tt.adBeta, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(iBeta.populateDisks(w), tt.desc+" beta", tt.adBeta, tt.wantAdBeta)

	}
}

func TestInstancePopulateMachineType(t *testing.T) {
	tests := []struct {
		desc, mt, wantMt string
		shouldErr        bool
	}{
		{"normal case", "mt", "projects/foo/zones/bar/machineTypes/mt", false},
		{"expand case", "zones/bar/machineTypes/mt", "projects/foo/zones/bar/machineTypes/mt", false},
	}

	assertTest := func(shouldErr bool, err DError, desc string, machineType string, wantMachineType string) {
		if shouldErr && err == nil {
			t.Errorf("%s: populateMachineType should have erred but didn't", desc)
		} else if !shouldErr && err != nil {
			t.Errorf("%s: populateMachineType returned an unexpected error: %v", desc, err)
		} else if err == nil && machineType != wantMachineType {
			t.Errorf("%s: MachineType not modified as expected: got: %q, want: %q", desc, machineType, wantMachineType)
		}
	}

	for _, tt := range tests {
		i := Instance{Instance: compute.Instance{MachineType: tt.mt, Zone: "bar"}, InstanceBase: InstanceBase{Resource: Resource{Project: "foo"}}}
		assertTest(tt.shouldErr, (&i.InstanceBase).populateMachineType(&i), tt.desc, i.MachineType, tt.wantMt)

		iBeta := InstanceBeta{Instance: computeBeta.Instance{MachineType: tt.mt, Zone: "bar"}, InstanceBase: InstanceBase{Resource: Resource{Project: "foo"}}}
		assertTest(tt.shouldErr, (&i.InstanceBase).populateMachineType(&iBeta), tt.desc+" beta", iBeta.MachineType, tt.wantMt)
	}
}

func TestInstancePopulateMetadata(t *testing.T) {
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
		if md == nil {
			return nil
		}
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
	getWantMdBeta := func(md map[string]string) *computeBeta.Metadata {
		if md == nil {
			return nil
		}
		for k, v := range baseMd {
			md[k] = v
		}
		result := &computeBeta.Metadata{}
		for k, v := range md {
			vCopy := v
			result.Items = append(result.Items, &computeBeta.MetadataItems{Key: k, Value: &vCopy})
		}
		return result
	}

	tests := []struct {
		desc          string
		md            map[string]string
		startupScript string
		wantMd        map[string]string
		shouldErr     bool
	}{
		{"defaults case", nil, "", map[string]string{}, false},
		{"startup script case", nil, "file", map[string]string{"startup-script-url": filePath, "windows-startup-script-url": filePath}, false},
		{"bad startup script case", nil, "foo", nil, true},
	}
	compFactory := func(items []*compute.MetadataItems) func(i, j int) bool {
		return func(i, j int) bool { return items[i].Key < items[j].Key }
	}
	compFactoryBeta := func(items []*computeBeta.MetadataItems) func(i, j int) bool {
		return func(i, j int) bool { return items[i].Key < items[j].Key }
	}

	assertTest := func(shouldErr bool, err DError, desc string, md, wantMd interface{}) {
		if err == nil {
			if shouldErr {
				t.Errorf("%s: populateMetadata should have errored but didn't", desc)
			} else {

				if diffRes := diff(md, wantMd, 0); diffRes != "" {
					t.Errorf("%s: Metadata not modified as expected: (-got +want)\n%s", desc, diffRes)
				}
			}
		} else if !shouldErr {
			t.Errorf("%s: populateMetadata returned an unexpected error: %v", desc, err)
		}
	}

	for _, tt := range tests {
		wantMd := getWantMd(tt.wantMd)
		wantMdBeta := getWantMdBeta(tt.wantMd)
		if tt.wantMd != nil {
			sort.Slice(wantMd.Items, compFactory(wantMd.Items))
			sort.Slice(wantMdBeta.Items, compFactoryBeta(wantMdBeta.Items))
		}

		i := Instance{InstanceBase: InstanceBase{StartupScript: tt.startupScript}, Metadata: tt.md}
		err := (&i.InstanceBase).populateMetadata(&i, w)
		sort.Slice(i.Instance.Metadata.Items, compFactory(i.Instance.Metadata.Items))
		assertTest(tt.shouldErr, err, tt.desc, i.Instance.Metadata, wantMd)

		iBeta := Instance{InstanceBase: InstanceBase{StartupScript: tt.startupScript}, Metadata: tt.md}
		err = (&iBeta.InstanceBase).populateMetadata(&iBeta, w)
		sort.Slice(iBeta.Instance.Metadata.Items, compFactory(iBeta.Instance.Metadata.Items))
		assertTest(tt.shouldErr, err, tt.desc+" beta", iBeta.Instance.Metadata, wantMdBeta)
	}
}

func TestInstancePopulateNetworks(t *testing.T) {
	defaultAcs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	defaultAcsBeta := []*computeBeta.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	tests := []struct {
		desc                string
		input, want         []*compute.NetworkInterface
		inputBeta, wantBeta []*computeBeta.NetworkInterface
	}{
		{
			"default case",
			nil,
			[]*compute.NetworkInterface{{
				Network:       fmt.Sprintf("projects/%s/global/networks/default", testProject),
				AccessConfigs: defaultAcs,
			}},
			nil,
			[]*computeBeta.NetworkInterface{{
				Network:       fmt.Sprintf("projects/%s/global/networks/default", testProject),
				AccessConfigs: defaultAcsBeta,
			}},
		},
		{
			"default AccessConfig case",
			[]*compute.NetworkInterface{{
				Network:    "global/networks/foo",
				Subnetwork: fmt.Sprintf("regions/%s/subnetworks/bar", getRegionFromZone(testZone)),
			}},
			[]*compute.NetworkInterface{{
				Network:       fmt.Sprintf("projects/%s/global/networks/foo", testProject),
				AccessConfigs: defaultAcs,
				Subnetwork:    fmt.Sprintf("projects/%s/regions/%s/subnetworks/bar", testProject, getRegionFromZone(testZone)),
			}},
			[]*computeBeta.NetworkInterface{{
				Network:    "global/networks/foo",
				Subnetwork: fmt.Sprintf("regions/%s/subnetworks/bar", getRegionFromZone(testZone)),
			}},
			[]*computeBeta.NetworkInterface{{
				Network:       fmt.Sprintf("projects/%s/global/networks/foo", testProject),
				AccessConfigs: defaultAcsBeta,
				Subnetwork:    fmt.Sprintf("projects/%s/regions/%s/subnetworks/bar", testProject, getRegionFromZone(testZone)),
			}},
		},
		{
			"subnetwork case",
			[]*compute.NetworkInterface{{
				Subnetwork: fmt.Sprintf("regions/%s/subnetworks/bar", getRegionFromZone(testZone)),
			}},
			[]*compute.NetworkInterface{{
				AccessConfigs: defaultAcs,
				Subnetwork:    fmt.Sprintf("projects/%s/regions/%s/subnetworks/bar", testProject, getRegionFromZone(testZone)),
			}},
			[]*computeBeta.NetworkInterface{{
				Subnetwork: fmt.Sprintf("regions/%s/subnetworks/bar", getRegionFromZone(testZone)),
			}},
			[]*computeBeta.NetworkInterface{{
				AccessConfigs: defaultAcsBeta,
				Subnetwork:    fmt.Sprintf("projects/%s/regions/%s/subnetworks/bar", testProject, getRegionFromZone(testZone)),
			}},
		},
	}

	assertTest := func(err DError, desc string, got, want interface{}) {
		if err != nil {
			t.Errorf("%s: should have returned an error", desc)
		} else if diffRes := diff(got, want, 0); diffRes != "" {
			t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", desc, diffRes)
		}
	}

	for _, tt := range tests {
		i := &Instance{Instance: compute.Instance{NetworkInterfaces: tt.input}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(i.populateNetworks(), tt.desc, i.NetworkInterfaces, tt.want)

		iBeta := &InstanceBeta{Instance: computeBeta.Instance{NetworkInterfaces: tt.inputBeta}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(iBeta.populateNetworks(), tt.desc, iBeta.NetworkInterfaces, tt.wantBeta)
	}
}

func TestInstancePopulateScopes(t *testing.T) {
	defaultScopes := []string{"https://www.googleapis.com/auth/devstorage.read_only"}
	tests := []struct {
		desc                   string
		input                  []string
		inputSas, want         []*compute.ServiceAccount
		inputSasBeta, wantBeta []*computeBeta.ServiceAccount
		shouldErr              bool
	}{
		{"default case", nil, nil, []*compute.ServiceAccount{{Email: "default", Scopes: defaultScopes}}, nil, []*computeBeta.ServiceAccount{{Email: "default", Scopes: defaultScopes}}, false},
		{"nondefault case", []string{"foo"}, nil, []*compute.ServiceAccount{{Email: "default", Scopes: []string{"foo"}}}, nil, []*computeBeta.ServiceAccount{{Email: "default", Scopes: []string{"foo"}}}, false},
		{"service accounts override case", []string{"foo"}, []*compute.ServiceAccount{}, []*compute.ServiceAccount{}, []*computeBeta.ServiceAccount{}, []*computeBeta.ServiceAccount{}, false},
	}

	for _, tt := range tests {
		i := &Instance{InstanceBase: InstanceBase{Scopes: tt.input}, Instance: compute.Instance{ServiceAccounts: tt.inputSas}}
		err := i.populateScopes()
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error", tt.desc)
			} else if diffRes := diff(i.ServiceAccounts, tt.want, 0); diffRes != "" {
				t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc, diffRes)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}

		iBeta := &InstanceBeta{InstanceBase: InstanceBase{Scopes: tt.input}, Instance: computeBeta.Instance{ServiceAccounts: tt.inputSasBeta}}
		err = iBeta.populateScopes()
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: should have returned an error", tt.desc+" beta")
			} else if diffRes := diff(iBeta.ServiceAccounts, tt.wantBeta, 0); diffRes != "" {
				t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc+" beta", diffRes)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc+" beta", err)
		}
	}
}

func TestInstancesValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s, e1 := w.NewStep("s")
	var e2 error
	w.ComputeClient, e2 = newTestGCEClient()
	if errs := addErrs(nil, e1, e2); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	mt := fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", testProject, testZone, testMachineType)
	ad := []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, testDisk), Mode: defaultDiskMode}}
	sourceMachineImage := fmt.Sprintf("projects/%s/global/machineImages/%s", w.Project, "test-machine-image")

	tests := []struct {
		desc      string
		i         *Instance
		iBeta     *InstanceBeta
		shouldErr bool
	}{
		{desc: "success simple case v1", i: &Instance{Instance: compute.Instance{Name: "i", Disks: ad, MachineType: mt}}, shouldErr: false},
		{desc: "failure dupe case v1", i: &Instance{Instance: compute.Instance{Name: "i", Disks: ad, MachineType: mt}}, shouldErr: true},
		{desc: "success simple case v0 beta", iBeta: &InstanceBeta{Instance: computeBeta.Instance{Name: "ib", MachineType: mt, SourceMachineImage: sourceMachineImage}}, shouldErr: false},
		{desc: "failure dupe case v0 beta", iBeta: &InstanceBeta{Instance: computeBeta.Instance{Name: "ib", MachineType: mt, SourceMachineImage: sourceMachineImage}}, shouldErr: true},
	}

	for _, tt := range tests {
		var ib *InstanceBase
		var ii InstanceInterface
		if tt.i != nil {
			s.CreateInstances = &CreateInstances{Instances: []*Instance{tt.i}}
			ib = &tt.i.InstanceBase
			ii = tt.i
		} else {
			s.CreateInstances = &CreateInstances{InstancesBeta: []*InstanceBeta{tt.iBeta}}
			ib = &tt.iBeta.InstanceBase
			ii = tt.iBeta
		}

		// Test sanitation -- clean/set irrelevant fields.
		ib.daisyName = ii.getName()
		ib.RealName = ii.getName()
		ib.link = fmt.Sprintf("projects/%s/zones/%s/instances/%s", w.Project, w.Zone, ii.getName())
		ib.Project = w.Project // Resource{} fields are tested in resource_test.
		ii.setZone(w.Zone)

		if err := s.validate(ctx); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestInstanceValidateDisks(t *testing.T) {
	// Test:
	// - good case
	// - no disks bad case
	// - bad disk mode case
	w := testWorkflow()
	w.disks.m = map[string]*Resource{
		testDisk: {link: fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, testDisk)},
	}
	m := defaultDiskMode

	tests := []struct {
		desc      string
		i         *Instance
		iBeta     *InstanceBeta
		shouldErr bool
	}{
		{desc: "success case reference", i: &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: testDisk, Mode: m}}, Zone: testZone}}, shouldErr: false},
		{desc: "success case url", i: &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, testDisk), Mode: m}}}}, shouldErr: false},
		{desc: "success source machine image provided no disks", iBeta: &InstanceBeta{Instance: computeBeta.Instance{Zone: testZone, SourceMachineImage: "source-machine-image"}}, shouldErr: false},
		{desc: "error project mismatch case", i: &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/foo/zones/%s/disks/%s", w.Zone, testDisk), Mode: m}}}}, shouldErr: true},
		{desc: "error no disks case", i: &Instance{Instance: compute.Instance{}}, shouldErr: true},
		{desc: "error disk mode case", i: &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: testDisk, Mode: "bad mode!"}}, Zone: testZone}}, shouldErr: true},
		{desc: "error both disks and source machine image provided", iBeta: &InstanceBeta{Instance: computeBeta.Instance{Disks: []*computeBeta.AttachedDisk{{Source: testDisk}}, Zone: testZone, SourceMachineImage: "source-machine-image"}}, shouldErr: true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)

		var ib *InstanceBase
		var ii InstanceInterface
		if tt.i != nil {
			// Test sanitation -- clean/set irrelevant fields.
			tt.i.Project = w.Project
			tt.i.Zone = w.Zone
			ib = &tt.i.InstanceBase
			ii = tt.i
		} else if tt.iBeta != nil {
			// Test sanitation -- clean/set irrelevant fields.
			tt.iBeta.Project = w.Project
			tt.iBeta.Zone = w.Zone
			ii = tt.iBeta
			ib = &tt.iBeta.InstanceBase
		}

		if err := ib.validateDisks(ii, s); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestInstanceValidateDiskSource(t *testing.T) {
	// Test:
	// - good case
	// - disk dne
	// - disk has wrong project/zone
	w := testWorkflow()
	w.disks.m = map[string]*Resource{"d": {link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}}
	m := defaultDiskMode
	p := testProject
	z := testZone

	tests := []struct {
		desc      string
		ads       []*compute.AttachedDisk
		adsBeta   []*computeBeta.AttachedDisk
		shouldErr bool
	}{
		{"good case", []*compute.AttachedDisk{{Source: "d", Mode: m}}, []*computeBeta.AttachedDisk{{Source: "d", Mode: m}}, false},
		{"disk dne case", []*compute.AttachedDisk{{Source: "dne", Mode: m}}, []*computeBeta.AttachedDisk{{Source: "dne", Mode: m}}, true},
		{"bad project case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/bad/zones/%s/disks/d", z), Mode: m}}, []*computeBeta.AttachedDisk{{Source: fmt.Sprintf("projects/bad/zones/%s/disks/d", z), Mode: m}}, true},
		{"bad zone case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/bad/disks/d", p), Mode: m}}, []*computeBeta.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/bad/disks/d", p), Mode: m}}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		i := &Instance{Instance: compute.Instance{Disks: tt.ads, Zone: z}, InstanceBase: InstanceBase{Resource: Resource{Project: p}}}
		err := (&i.InstanceBase).validateDiskSource(tt.ads[0].Source, i, s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}

		iBeta := &InstanceBeta{Instance: computeBeta.Instance{Disks: tt.adsBeta, Zone: z}, InstanceBase: InstanceBase{Resource: Resource{Project: p}}}
		err = (&iBeta.InstanceBase).validateDiskSource(tt.ads[0].Source, iBeta, s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc+" beta", err)
		}
	}
}

func TestInstanceValidateDiskInitializeParams(t *testing.T) {
	// Test:
	// - good case
	// - bad disk name
	// - duplicate disk
	// - bad source given
	// - bad disk types (wrong project/zone)
	// - check that disks are created
	w := testWorkflow()
	w.images.m = map[string]*Resource{"i": {link: "iLink"}}
	dt := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-ssd", testProject, testZone)

	tests := []struct {
		desc      string
		p         *compute.AttachedDiskInitializeParams
		pBeta     *computeBeta.AttachedDiskInitializeParams
		shouldErr bool
	}{
		{"good case", &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: dt}, &computeBeta.AttachedDiskInitializeParams{DiskName: "foo-beta", SourceImage: "i", DiskType: dt}, false},
		{"bad disk name case", &compute.AttachedDiskInitializeParams{DiskName: "bad!", SourceImage: "i", DiskType: dt}, &computeBeta.AttachedDiskInitializeParams{DiskName: "bad!2", SourceImage: "i", DiskType: dt}, true},
		{"bad dupe disk case", &compute.AttachedDiskInitializeParams{DiskName: "foo", SourceImage: "i", DiskType: dt}, &computeBeta.AttachedDiskInitializeParams{DiskName: "foo-beta", SourceImage: "i", DiskType: dt}, true},
		{"bad source case", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: dt}, &computeBeta.AttachedDiskInitializeParams{DiskName: "bar-beta", SourceImage: "i2", DiskType: dt}, true},
		{"bad disk type case", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: fmt.Sprintf("projects/bad/zones/%s/diskTypes/pd-ssd", testZone)}, &computeBeta.AttachedDiskInitializeParams{DiskName: "bar-beta", SourceImage: "i2", DiskType: fmt.Sprintf("projects/bad/zones/%s/diskTypes/pd-ssd", testZone)}, true},
		{"bad disk type case 2", &compute.AttachedDiskInitializeParams{DiskName: "bar", SourceImage: "i2", DiskType: fmt.Sprintf("projects/%s/zones/bad/diskTypes/pd-ssd", testProject)}, &computeBeta.AttachedDiskInitializeParams{DiskName: "bar-beta", SourceImage: "i2", DiskType: fmt.Sprintf("projects/%s/zones/bad/diskTypes/pd-ssd", testProject)}, true},
	}

	assertTest := func(shouldErr bool, err DError, desc string) {
		if err == nil {
			if shouldErr {
				t.Errorf("%s: should have returned an error but didn't", desc)
			}
		} else if !shouldErr {
			t.Errorf("%s: unexpected error: %v", desc, err)
		}
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		ci := &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{InitializeParams: tt.p}}, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		s.CreateInstances = &CreateInstances{Instances: []*Instance{ci}}
		assertTest(tt.shouldErr, (&ci.InstanceBase).validateDiskInitializeParams(ci.getComputeDisks()[0], ci, s), tt.desc)

		sBeta, _ := w.NewStep(tt.desc + "Beta")
		ciBeta := &InstanceBeta{Instance: computeBeta.Instance{Disks: []*computeBeta.AttachedDisk{{InitializeParams: tt.pBeta}}, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		sBeta.CreateInstances = &CreateInstances{InstancesBeta: []*InstanceBeta{ciBeta}}
		assertTest(tt.shouldErr, (&ciBeta.InstanceBase).validateDiskInitializeParams(ciBeta.getComputeDisks()[0], ciBeta, sBeta), tt.desc+" beta")
	}

	// Check good disks were created.
	wantCreator := w.Steps["good case"]
	wantLink := fmt.Sprintf("projects/%s/zones/%s/disks/foo", testProject, testZone)
	wantFoo := &Resource{RealName: "foo", link: wantLink, creator: wantCreator}
	if gotFoo, ok := w.disks.m["foo"]; !ok || !reflect.DeepEqual(gotFoo, wantFoo) {
		t.Errorf("foo resource not added as expected: got: %+v, want: %+v", gotFoo, wantFoo)
	}

	// Check proper image user registrations.
	wantU := w.Steps["good case"]
	found := false
	for _, u := range w.images.m["i"].users {
		if u == wantU {
			found = true
		}
	}
	if !found {
		t.Error("good case should have been a registered user of image \"i\"")
	}
}

func TestInstanceValidateMachineType(t *testing.T) {
	c, err := newTestGCEClient()
	if err != nil {
		t.Fatal(err)
	}
	getMachineTypeFn := func(_, _, mt string) (*compute.MachineType, error) {
		if mt != "custom" {
			return nil, errors.New("bad machine type")
		}
		return nil, nil
	}

	c.GetMachineTypeFn = getMachineTypeFn

	tests := []struct {
		desc      string
		mt        string
		shouldErr bool
	}{
		{"good case", fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", testProject, testZone, testMachineType), false},
		{"custom case", fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", testProject, testZone, "custom"), false},
		{"bad machine type case", fmt.Sprintf("projects/%s/zones/%s/machineTypes/bad-mt", testProject, testZone), true},
		{"bad project case", fmt.Sprintf("projects/p2/zones/%s/machineTypes/%s", testZone, testMachineType), true},
		{"bad zone case", fmt.Sprintf("projects/%s/zones/z2/machineTypes/%s", testProject, testMachineType), true},
		{"bad zone case 2", "zones/z2/machineTypes/mt", true},
	}

	assertTest := func(shouldErr bool, err DError, desc string) {
		if shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", desc)
		} else if !shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", desc, err)
		}
	}
	for _, tt := range tests {
		ci := &Instance{Instance: compute.Instance{MachineType: tt.mt, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(tt.shouldErr, (&ci.InstanceBase).validateMachineType(ci, c), tt.desc)

		ciBeta := &InstanceBeta{Instance: computeBeta.Instance{MachineType: tt.mt, Zone: testZone}, InstanceBase: InstanceBase{Resource: Resource{Project: testProject}}}
		assertTest(tt.shouldErr, (&ciBeta.InstanceBase).validateMachineType(ciBeta, c), tt.desc+" beta")
	}
}

func TestInstanceValidateNetworks(t *testing.T) {
	w := testWorkflow()
	acs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	acsBeta := []*computeBeta.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	w.networks.m = map[string]*Resource{testNetwork: {link: fmt.Sprintf("projects/%s/global/networks/%s", testProject, testNetwork)}}
	w.subnetworks.m = map[string]*Resource{testSubnetwork: {link: fmt.Sprintf("projects/%s/global/subnetworks/%s", testProject, testSubnetwork)}}

	r := Resource{Project: testProject}
	tests := []struct {
		desc      string
		ci        *Instance
		ciBeta    *InstanceBeta
		shouldErr bool
	}{
		{
			"good case reference",
			&Instance{InstanceBase: InstanceBase{Resource: r}, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: testNetwork, AccessConfigs: acs}}}},
			&InstanceBeta{InstanceBase: InstanceBase{Resource: r}, Instance: computeBeta.Instance{NetworkInterfaces: []*computeBeta.NetworkInterface{{Network: testNetwork, AccessConfigs: acsBeta}}}},
			false,
		},
		{
			"good case only subnetwork",
			&Instance{InstanceBase: InstanceBase{Resource: r}, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Subnetwork: testSubnetwork, AccessConfigs: acs}}}},
			&InstanceBeta{InstanceBase: InstanceBase{Resource: r}, Instance: computeBeta.Instance{NetworkInterfaces: []*computeBeta.NetworkInterface{{Subnetwork: testSubnetwork, AccessConfigs: acsBeta}}}},
			false,
		},
		{
			"good case url",
			&Instance{InstanceBase: InstanceBase{Resource: r}, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/%s", testProject, testNetwork), AccessConfigs: acs}}}},
			&InstanceBeta{InstanceBase: InstanceBase{Resource: r}, Instance: computeBeta.Instance{NetworkInterfaces: []*computeBeta.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/%s", testProject, testNetwork), AccessConfigs: acsBeta}}}},
			false,
		},
		{
			"bad name case",
			&Instance{InstanceBase: InstanceBase{Resource: r}, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/bad!", testProject), AccessConfigs: acs}}}},
			&InstanceBeta{InstanceBase: InstanceBase{Resource: r}, Instance: computeBeta.Instance{NetworkInterfaces: []*computeBeta.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/bad!", testProject), AccessConfigs: acsBeta}}}},
			true,
		},
		{
			"bad project case",
			&Instance{InstanceBase: InstanceBase{Resource: r}, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/bad!/global/networks/%s", testNetwork), AccessConfigs: acs}}}},
			&InstanceBeta{InstanceBase: InstanceBase{Resource: r}, Instance: computeBeta.Instance{NetworkInterfaces: []*computeBeta.NetworkInterface{{Network: fmt.Sprintf("projects/bad!/global/networks/%s", testNetwork), AccessConfigs: acsBeta}}}},
			true,
		},
	}

	assertTest := func(shouldErr bool, err DError, desc string) {
		if shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", desc)
		} else if !shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", desc, err)
		}
	}
	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		s.CreateInstances = &CreateInstances{Instances: []*Instance{tt.ci}, InstancesBeta: []*InstanceBeta{tt.ciBeta}}
		assertTest(tt.shouldErr, tt.ci.validateNetworks(s), tt.desc)
		assertTest(tt.shouldErr, tt.ciBeta.validateNetworks(s), tt.desc+" beta")
	}
}
