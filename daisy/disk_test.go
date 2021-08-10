package daisy

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/api/compute/v1"
)

func TestDiskPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient = nil
	w.StorageClient = nil
	w.images.m = map[string]*Resource{"i1": {RealName: "ifoo", link: "http://ifoo"}}
	s, _ := w.NewStep("s")

	name := "foo"
	genName := w.genName(name)
	defType := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", w.Project, w.Zone)
	ssdType := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-ssd", w.Project, w.Zone)
	tests := []struct {
		desc        string
		input, want *Disk
		wantErr     bool
	}{
		{
			"defaults case",
			&Disk{Disk: compute.Disk{Name: name}},
			&Disk{Disk: compute.Disk{Name: genName, Type: defType, Zone: w.Zone}},
			false,
		},
		{
			"input size",
			&Disk{Disk: compute.Disk{Name: name}, SizeGb: "10"},
			&Disk{Disk: compute.Disk{Name: genName, Type: defType, SizeGb: 10, Zone: w.Zone}, SizeGb: "10"},
			false,
		},
		{
			"extend Type URL case",
			&Disk{Disk: compute.Disk{Name: name, Type: "pd-ssd"}, SizeGb: "10"},
			&Disk{Disk: compute.Disk{Name: genName, Type: ssdType, SizeGb: 10, Zone: w.Zone}, SizeGb: "10"},
			false,
		},
		{
			"extend Type URL case 2",
			&Disk{Disk: compute.Disk{Name: name, Type: fmt.Sprintf("zones/%s/diskTypes/pd-ssd", w.Zone)}},
			&Disk{Disk: compute.Disk{Name: genName, Type: ssdType, Zone: w.Zone}},
			false,
		},
		{
			"extend SourceImage URL case",
			&Disk{Disk: compute.Disk{Name: name, SourceImage: "global/images/ifoo"}},
			&Disk{Disk: compute.Disk{Name: genName, Type: defType, SourceImage: fmt.Sprintf("projects/%s/global/images/ifoo", w.Project), Zone: w.Zone}},
			false,
		},
		{
			"SourceImage daisy name case",
			&Disk{Disk: compute.Disk{Name: name, SourceImage: "ifoo"}},
			&Disk{Disk: compute.Disk{Name: genName, SourceImage: "ifoo", Type: defType, Zone: w.Zone}},
			false,
		},
		{
			"bad SizeGb case",
			&Disk{Disk: compute.Disk{Name: "foo"}, SizeGb: "ten"},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		err := tt.input.populate(ctx, s)

		// Test sanitation -- clean/set irrelevant fields.
		if tt.want != nil {
			tt.want.Description = tt.input.Description
		}
		tt.input.Resource = Resource{} // These fields are tested in resource_test.

		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diffRes := diff(tt.input, tt.want, 0); diffRes != "" {
			t.Errorf("%s: populated Disk does not match expectation: (-got +want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestDiskRegAttach(t *testing.T) {
	// Test:
	// - normal attachment
	// - repeat attachment (does nothing in API, even if different mode)
	// - concurrent attachment in RO mode
	// - concurrent attachment conflict
	// - instance/disk resource DNE
	// - attach detached disk
	// - attach detached disk between sibling IncludeWorkflows (detached in iwSubStep, attached in iw2SubStep)
	// - attachment conflict between sibling IncludeWorkflows (attached in iwSubStep, attached in iw2SubStep)

	// Workflow maps
	// w:
	// att ---> det ---> s
	// iwStep ---> iw2Step
	// s2
	//
	// iw:
	// iwSubStep
	//
	// iw2:
	// iw2SubStep
	w := testWorkflow()

	iw := New()
	w.includeWorkflow(iw)
	iw2 := New()
	w.includeWorkflow(iw2)

	iwStep, e1 := w.NewStep("iwStep")
	iwStep.IncludeWorkflow = &IncludeWorkflow{Workflow: iw}
	iwSubStep, e2 := iw.NewStep("iwSubStep")

	iw2Step, e3 := w.NewStep("iw2Step")
	iw2Step.IncludeWorkflow = &IncludeWorkflow{Workflow: iw2}
	iw2SubStep, e4 := iw2.NewStep("iw2SubStep")

	s, e5 := w.NewStep("s")
	s2, e6 := w.NewStep("s2")
	att, e7 := w.NewStep("att")
	det, e8 := w.NewStep("det")
	e9 := w.AddDependency(det, att)
	e10 := w.AddDependency(s, det)
	e11 := w.AddDependency(iw2Step, iwStep)
	w.disks.m = map[string]*Resource{"d": nil, "dDetached": nil, "dIWDetached": nil, "dIWAttached": nil}
	w.instances.m = map[string]*Resource{"i": nil, "i2": nil, "i3": nil, "iPrevAtt": nil, "iIW": nil, "iIW2": nil}
	w.disks.attachments = map[string]map[string]*diskAttachment{
		"dDetached":   {"iPrevAtt": {mode: diskModeRW, attacher: att, detacher: det}},
		"dIWDetached": {"iIW": {mode: diskModeRW, detacher: iwSubStep}},
		"dIWAttached": {"iIW": {mode: diskModeRW}},
	}

	if errs := addErrs(nil, e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	tests := []struct {
		desc, d, i, mode string
		s                *Step
		shouldErr        bool
	}{
		{"normal case", "d", "i", diskModeRO, s, false},
		{"repeat attachment case", "d", "i", diskModeRW, s2, false},
		{"concurrent RO case", "d", "i2", diskModeRO, s, false},
		{"concurrent conflict case", "d", "i3", diskModeRW, s, true},
		{"attach detached case", "dDetached", "i", diskModeRW, s, false},
		{"attach detached between IncludeWorkflows case", "dIWDetached", "iIW2", diskModeRW, iw2SubStep, false},
		{"attachment conflict between IncludeWorkflows case", "dIWAttached", "iIW2", diskModeRW, iw2SubStep, true},
	}

	for _, tt := range tests {
		err := w.disks.regAttach(tt.d, tt.d, tt.i, tt.mode, tt.s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have err'ed but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	want := map[string]map[string]*diskAttachment{
		"dDetached": {
			"iPrevAtt": w.disks.attachments["dDetached"]["iPrevAtt"], // no changes
			"i":        {"dDetached", diskModeRW, s, nil},
		},
		"dIWDetached": {
			"iIW":  w.disks.attachments["dIWDetached"]["iIW"], // no changes
			"iIW2": {"dIWDetached", diskModeRW, iw2SubStep, nil},
		},
		"dIWAttached": w.disks.attachments["dIWAttached"], // no changes
		"d": {
			"i":  {"d", diskModeRO, s, nil},
			"i2": {"d", diskModeRO, s, nil},
		},
	}
	if diffRes := diff(w.disks.attachments, want, 7); diffRes != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDiskDetachHelper(t *testing.T) {
	// Test:
	// - normal detach
	// - detacher doesn't depend on attacher
	// - already detached
	// - not attached
	// - disk DNE

	w := testWorkflow()
	s, _ := w.NewStep("s")
	att1, _ := w.NewStep("att1")
	att2, _ := w.NewStep("att2")
	det, _ := w.NewStep("det")
	w.AddDependency(s, att1)

	reset := func() {
		w.disks.attachments = map[string]map[string]*diskAttachment{
			"d1": {"i": {attacher: att1}},
			"d2": {"i": {attacher: att2}},
			"d3": {"i": {detacher: det}},
			"d4": {},
			"d5": {},
		}
	}

	tests := []struct {
		desc, dName           string
		isAttached, shouldErr bool
	}{
		{"normal case", "d1", false, false},
		{"not dependent on attacher case", "d2", false, true},
		{"already detached case", "d3", false, true},
		{"not attached case", "d4", false, true},
		{"attached before workflow case", "d5", true, false},
		{"disk DNE case", "d6", false, true},
	}

	for _, tt := range tests {
		reset()
		if err := w.disks.detachHelper(tt.dName, "i", tt.isAttached, s); err != nil {
			if !tt.shouldErr {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		} else if tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.isAttached && w.disks.attachments[tt.dName]["i"].detacher != s {
			t.Errorf("%s: step s should have been registered as disconnector", tt.desc)
		}
	}
}

func TestDiskRegDetach(t *testing.T) {
	// Test:
	// - no error from helper
	// - error from helper
	w := testWorkflow()

	var helperErr *DError
	w.disks.testDetachHelper = func(_, _ string, _ *Step) DError {
		return *helperErr
	}

	tests := []struct {
		desc       string
		isAttached bool
		helperErr  DError
		shouldErr  bool
	}{
		{"normal case", false, nil, false},
		{"disconnect helper error case", true, Errf("error!"), true},
		{"normal attached case", false, nil, false},
		{"disconnect helper attached error case", true, Errf("error!"), true},
	}

	for _, tt := range tests {
		helperErr = &tt.helperErr
		if err := w.disks.regDetach("", "", tt.isAttached, nil); tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestDiskRegDetachAll(t *testing.T) {
	// Test:
	// - no error from helper
	// - error from helper
	// - skip already detached
	w := testWorkflow()
	w.cloudLoggingClient = nil
	s, _ := w.NewStep("s")
	otherDetacher, _ := w.NewStep("other-detacher")

	var callsArgs [][]interface{}
	var helperErr *DError
	w.disks.testDetachHelper = func(dName, iName string, s *Step) DError {
		callsArgs = append(callsArgs, []interface{}{dName, iName, s})
		return *helperErr
	}

	reset := func() {
		callsArgs = nil
		w.disks.attachments = map[string]map[string]*diskAttachment{
			"d1": {"i": {}},
			"d2": {},
			"d3": {"i": {detacher: otherDetacher}},
		}
	}

	tests := []struct {
		desc      string
		helperErr DError
		shouldErr bool
	}{
		{"normal case", nil, false},
		{"disconnect helper error case", Errf("error!"), true},
	}

	for _, tt := range tests {
		reset()
		helperErr = &tt.helperErr
		wantCallsArgs := [][]interface{}{{"d1", "i", s}}
		if err := w.disks.regDetachAll("i", s); err != nil {
			if !tt.shouldErr {
				t.Errorf("%s: unexpected error: %v", tt.desc, err)
			}
		} else if tt.shouldErr {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if diffRes := diff(callsArgs, wantCallsArgs, 0); diffRes != "" {
			t.Errorf("%s: detachHelper not called as expected: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestDiskValidate(t *testing.T) {
	w := testWorkflow()
	s, e1 := w.NewStep("s")
	iCreator, e2 := w.NewStep("iCreator") // Step that created image "i1"
	e3 := w.AddDependency(s, iCreator)
	if errs := addErrs(nil, e1, e2, e3); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}
	w.images.m = map[string]*Resource{"i1": {creator: iCreator}}                    // "i1" resource
	w.snapshots.m = map[string]*Resource{"ss1": {RealName: "ss1", link: "ss1link"}} // "ss1" resource

	ty := fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", w.Project, w.Zone, "pd-standard")
	tests := []struct {
		desc      string
		d         *Disk
		shouldErr bool
	}{
		{
			"source image case",
			&Disk{Disk: compute.Disk{Name: "d1", SourceImage: "i1", Type: ty}},
			false,
		},
		{
			"source image url case",
			&Disk{Disk: compute.Disk{Name: "d2", SourceImage: fmt.Sprintf("projects/%s/global/images/%s", testProject, testImage), Type: ty}},
			false,
		},
		{
			"image from family case",
			&Disk{Disk: compute.Disk{Name: "d3", SourceImage: fmt.Sprintf("projects/%s/global/images/family/%s", testProject, testFamily), Type: ty}},
			false,
		},
		{
			"blank disk case",
			&Disk{Disk: compute.Disk{Name: "d4", SizeGb: 1, Type: ty}},
			false,
		},
		{
			"image OBSOLETE case",
			&Disk{Disk: compute.Disk{Name: "d1", SourceImage: fmt.Sprintf("projects/foo/global/images/%s", testImage), Type: ty}},
			true,
		},
		{
			"source image dne case",
			&Disk{Disk: compute.Disk{Name: "d5", SourceImage: "dne", Type: ty}},
			true,
		},
		{
			"dupe disk case",
			&Disk{Disk: compute.Disk{Name: "d1", SizeGb: 1, Type: ty}},
			true,
		},
		{
			"no size/source case",
			&Disk{Disk: compute.Disk{Name: "d6", Type: ty}},
			true,
		},
		{
			"bad type case",
			&Disk{Disk: compute.Disk{Name: "d7", SizeGb: 1, Type: "t!"}},
			true,
		},
		{
			"source snapshot case",
			&Disk{Disk: compute.Disk{Name: "d8", SourceSnapshot: "ss1", Type: ty}},
			false,
		},
		{
			"source snapshot url case",
			&Disk{Disk: compute.Disk{Name: "d9", SourceSnapshot: fmt.Sprintf("projects/%s/global/snapshots/%s", testProject, testSnapshot), Type: ty}},
			false,
		},
		{
			"source snapshot dne case",
			&Disk{Disk: compute.Disk{Name: "d10", SourceSnapshot: "dne", Type: ty}},
			true,
		},
		{
			"source snapshot with size",
			&Disk{Disk: compute.Disk{Name: "d11", SourceSnapshot: "ss1", Type: ty}, SizeGb: "50"},
			false,
		},
	}

	for _, tt := range tests {
		// Test sanitation -- clean/set irrelevant fields.
		tt.d.daisyName = tt.d.Name
		tt.d.RealName = tt.d.Name
		tt.d.link = fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, tt.d.Name)
		tt.d.Project = w.Project
		tt.d.Zone = w.Zone

		s.CreateDisks = &CreateDisks{tt.d}
		err := s.validate(context.Background())
		if err == nil {
			if tt.shouldErr {
				t.Errorf("%s: did not return an error as expected", tt.desc)
			}
			if res, _ := w.disks.get(tt.d.Name); res != &tt.d.Resource {
				t.Errorf("%s: %q not in disk registry as expected: got=%v want=%v", tt.desc, tt.d.Name, &tt.d.Resource, res)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
