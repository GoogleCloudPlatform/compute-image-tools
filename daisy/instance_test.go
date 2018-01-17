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
		shouldErr bool
	}{
		{"good case", &Instance{}, false},
		{"bad case", &Instance{StartupScript: "Workflow source DNE and can't resolve!"}, true},
	}

	for testNum, tt := range tests {
		s, _ := w.NewStep("s" + strconv.Itoa(testNum))
		err := tt.i.populate(context.Background(), s)

		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestInstancePopulateDisks(t *testing.T) {
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
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}},
		},
		{
			"multiple disks case",
			[]*compute.AttachedDisk{{Source: "d1"}, {Source: "d2"}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: defaultDiskMode, DeviceName: "d1"}, {Boot: false, Source: "d2", Mode: defaultDiskMode, DeviceName: "d2"}},
		},
		{
			"mode specified case",
			[]*compute.AttachedDisk{{Source: "d1", Mode: diskModeRO}},
			[]*compute.AttachedDisk{{Boot: true, Source: "d1", Mode: diskModeRO, DeviceName: "d1"}},
		},
		{
			"init params daisy image (and other defaults)",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params image short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "global/images/i"}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params image extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: fmt.Sprintf("projects/%s/global/images/i", testProject), DiskType: defDT}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params disk type short url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("zones/%s/diskTypes/dt", testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
		},
		{
			"init params disk type extended url",
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}}},
			[]*compute.AttachedDisk{{InitializeParams: &compute.AttachedDiskInitializeParams{DiskName: iName, SourceImage: "i", DiskType: fmt.Sprintf("projects/%s/zones/%s/diskTypes/dt", testProject, testZone)}, Mode: defaultDiskMode, Boot: true, DeviceName: iName}},
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
		},
	}

	for _, tt := range tests {
		i := Instance{Instance: compute.Instance{Name: iName, Disks: tt.ad, Zone: testZone}, Resource: Resource{Project: testProject}}
		err := i.populateDisks(w)
		if err != nil {
			t.Errorf("%s: populateDisks returned an unexpected error: %v", tt.desc, err)
		} else if diffRes := diff(tt.ad, tt.wantAd, 0); diffRes != "" {
			t.Errorf("%s: AttachedDisks not modified as expected: (-got +want)\n%s", tt.desc, diffRes)
		}
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

	for _, tt := range tests {
		i := Instance{Instance: compute.Instance{MachineType: tt.mt, Zone: "bar"}, Resource: Resource{Project: "foo"}}
		err := i.populateMachineType()
		if tt.shouldErr && err == nil {
			t.Errorf("%s: populateMachineType should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: populateMachineType returned an unexpected error: %v", tt.desc, err)
		} else if err == nil && i.MachineType != tt.wantMt {
			t.Errorf("%s: MachineType not modified as expected: got: %q, want: %q", tt.desc, i.MachineType, tt.wantMt)
		}
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
		i := Instance{Metadata: tt.md, StartupScript: tt.startupScript}
		err := i.populateMetadata(w)
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: populateMetadata should have erred but didn't", tt.desc)
			} else {
				compFactory := func(items []*compute.MetadataItems) func(i, j int) bool {
					return func(i, j int) bool { return items[i].Key < items[j].Key }
				}
				sort.Slice(i.Instance.Metadata.Items, compFactory(i.Instance.Metadata.Items))
				sort.Slice(tt.wantMd.Items, compFactory(tt.wantMd.Items))
				if diffRes := diff(i.Instance.Metadata, tt.wantMd, 0); diffRes != "" {
					t.Errorf("%s: Metadata not modified as expected: (-got +want)\n%s", tt.desc, diffRes)
				}
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: populateMetadata returned an unexpected error: %v", tt.desc, err)
		}
	}
}

func TestInstancePopulateNetworks(t *testing.T) {
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
		i := &Instance{Instance: compute.Instance{NetworkInterfaces: tt.input}, Resource: Resource{Project: testProject}}
		err := i.populateNetworks()
		if err != nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if diffRes := diff(i.NetworkInterfaces, tt.want, 0); diffRes != "" {
			t.Errorf("%s: NetworkInterfaces not modified as expected: (-got +want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestInstancePopulateScopes(t *testing.T) {
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
		i := &Instance{Scopes: tt.input, Instance: compute.Instance{ServiceAccounts: tt.inputSas}}
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

	tests := []struct {
		desc      string
		i         *Instance
		shouldErr bool
	}{
		{"good simple case", &Instance{Instance: compute.Instance{Name: "i", Disks: ad, MachineType: mt}}, false},
		{"bad dupe case", &Instance{Instance: compute.Instance{Name: "i", Disks: ad, MachineType: mt}}, true},
	}

	for _, tt := range tests {
		s.CreateInstances = &CreateInstances{tt.i}

		// Test sanitation -- clean/set irrelevant fields.
		tt.i.daisyName = tt.i.Name
		tt.i.RealName = tt.i.Name
		tt.i.link = fmt.Sprintf("projects/%s/zones/%s/instances/%s", w.Project, w.Zone, tt.i.Name)
		tt.i.Project = w.Project // Resource{} fields are tested in resource_test.
		tt.i.Zone = w.Zone

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
		shouldErr bool
	}{
		{"good case reference", &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: testDisk, Mode: m}}, Zone: testZone}}, false},
		{"good case url", &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, testDisk), Mode: m}}}}, false},
		{"project mismatch case", &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/foo/zones/%s/disks/%s", w.Zone, testDisk), Mode: m}}}}, true},
		{"bad no disks case", &Instance{Instance: compute.Instance{}}, true},
		{"bad disk mode case", &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{Source: testDisk, Mode: "bad mode!"}}, Zone: testZone}}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)

		// Test sanitation -- clean/set irrelevant fields.
		tt.i.Project = w.Project
		tt.i.Zone = w.Zone

		if err := tt.i.validateDisks(s); tt.shouldErr && err == nil {
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
		shouldErr bool
	}{
		{"good case", []*compute.AttachedDisk{{Source: "d", Mode: m}}, false},
		{"disk dne case", []*compute.AttachedDisk{{Source: "dne", Mode: m}}, true},
		{"bad project case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/bad/zones/%s/disks/d", z), Mode: m}}, true},
		{"bad zone case", []*compute.AttachedDisk{{Source: fmt.Sprintf("projects/%s/zones/bad/disks/d", p), Mode: m}}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		i := &Instance{Instance: compute.Instance{Disks: tt.ads, Zone: z}, Resource: Resource{Project: p}}
		err := i.validateDiskSource(tt.ads[0], s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
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
		ci := &Instance{Instance: compute.Instance{Disks: []*compute.AttachedDisk{{InitializeParams: tt.p}}, Zone: testZone}, Resource: Resource{Project: testProject}}
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

	for _, tt := range tests {
		ci := &Instance{Instance: compute.Instance{MachineType: tt.mt, Zone: testZone}, Resource: Resource{Project: testProject}}
		if err := ci.validateMachineType(c); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestInstanceValidateNetworks(t *testing.T) {
	w := testWorkflow()
	acs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	w.networks.m = map[string]*Resource{testNetwork: {link: fmt.Sprintf("projects/%s/global/networks/%s", testProject, testNetwork)}}

	r := Resource{Project: testProject}
	tests := []struct {
		desc      string
		ci        *Instance
		shouldErr bool
	}{
		{"good case reference", &Instance{Resource: r, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: testNetwork, AccessConfigs: acs}}}}, false},
		{"good case url", &Instance{Resource: r, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/%s", testProject, testNetwork), AccessConfigs: acs}}}}, false},
		{"bad name case", &Instance{Resource: r, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/%s/global/networks/bad!", testProject), AccessConfigs: acs}}}}, true},
		{"bad project case", &Instance{Resource: r, Instance: compute.Instance{NetworkInterfaces: []*compute.NetworkInterface{{Network: fmt.Sprintf("projects/bad!/global/networks/%s", testNetwork), AccessConfigs: acs}}}}, true},
	}

	for _, tt := range tests {
		s, _ := w.NewStep(tt.desc)
		s.CreateInstances = &CreateInstances{tt.ci}
		if err := tt.ci.validateNetworks(s); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
