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
		} else if diffRes := diff(tt.input, tt.want); diffRes != "" {
			t.Errorf("%s: populated Disk does not match expectation: (-got +want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestDiskRegisterAttachment(t *testing.T) {
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
	iw := w.NewIncludedWorkflow()
	iw2 := w.NewIncludedWorkflow()
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
	d := &Resource{RealName: "d"}
	dDetached := &Resource{RealName: "dDetached"}
	dIWDetached := &Resource{RealName: "dIWDetached"}
	dIWAttached := &Resource{RealName: "dIWAttached"}
	disks[w].m = map[string]*Resource{"d": d, "dDetached": dDetached, "dIWDetached": dIWDetached, "dIWAttached": dIWAttached}
	i := &Resource{RealName: "i"}
	i2 := &Resource{RealName: "i2"}
	i3 := &Resource{RealName: "i3"}
	iPrevAtt := &Resource{RealName: "iPrevAtt"}
	iIW := &Resource{RealName: "iIW"}
	iIW2 := &Resource{RealName: "iIW2"}
	instances[w].m = map[string]*Resource{"i": i, "i2": i2, "i3": i3, "iPrevAtt": iPrevAtt, "iIW": iIW, "iIW2": iIW2}
	disks[w].attachments = map[*Resource]map[*Resource]*diskAttachment{
		dDetached:   {iPrevAtt: {mode: diskModeRW, attacher: att, detacher: det}},
		dIWDetached: {iIW: {mode: diskModeRW, detacher: iwSubStep}},
		dIWAttached: {iIW: {mode: diskModeRW}},
	}

	if errs := addErrs(nil, e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11); errs != nil {
		t.Fatalf("test set error: %v", errs)
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
		{"instance DNE case", "d", "dne", diskModeRO, s, true},
		{"disk DNE case", "dne", "i", diskModeRO, s, true},
		{"attach detached case", "dDetached", "i", diskModeRW, s, false},
		{"attach detached between IncludeWorkflows case", "dIWDetached", "iIW2", diskModeRW, iw2SubStep, false},
		{"attachment conflict between IncludeWorkflows case", "dIWAttached", "iIW2", diskModeRW, iw2SubStep, true},
	}

	for _, tt := range tests {
		err := disks[w].registerAttachment(tt.d, tt.i, tt.mode, tt.s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have err'ed but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	want := map[*Resource]map[*Resource]*diskAttachment{
		dDetached: {
			iPrevAtt: disks[w].attachments[dDetached][iPrevAtt], // no changes
			i:        {diskModeRW, s, nil},
		},
		dIWDetached: {
			iIW:  disks[w].attachments[dIWDetached][iIW], // no changes
			iIW2: {diskModeRW, iw2SubStep, nil},
		},
		dIWAttached: disks[w].attachments[dIWAttached], // no changes
		d: {
			i:  {diskModeRO, s, nil},
			i2: {diskModeRO, s, nil},
		},
	}
	if diffRes := diff(disks[w].attachments, want); diffRes != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDiskDetachHelper(t *testing.T) {
	// Test
	// - not attached
	// - detacher doesn't depend on attacher
	// - normal detach

	w := testWorkflow()
	s, _ := w.NewStep("s")
	bad, _ := w.NewStep("bad")
	att, _ := w.NewStep("att")
	det, _ := w.NewStep("det")
	w.AddDependency(s, att)
	w.AddDependency(det, att)
	d := &Resource{RealName: "d"}
	i := &Resource{RealName: "i"}
	i2 := &Resource{RealName: "i2"}
	i3 := &Resource{RealName: "i3"}

	if err := disks[w].detachHelper(d, i, s); err == nil {
		t.Error("detaching i from d (which has no attachments) should have failed")
	}
	disks[w].attachments[d] = map[*Resource]*diskAttachment{
		i:  {attacher: att},
		i2: {attacher: att, detacher: det},
	}

	tests := []struct {
		desc      string
		d, i      *Resource
		s         *Step
		shouldErr bool
	}{
		{"not attached case", d, i3, s, true},
		{"not attached (already detached) case", d, i2, s, true},
		{"detacher doesn't depend on attacher case", d, i, bad, true},
		{"normal detach", d, i, s, false},
	}

	for _, tt := range tests {
		err := disks[w].detachHelper(tt.d, tt.i, tt.s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	// Check state.
	want := map[*Resource]map[*Resource]*diskAttachment{
		d: {
			i:  {attacher: att, detacher: s},
			i2: disks[w].attachments[d][i2],
		},
	}
	if diffRess := diff(disks[w].attachments, want); diffRess != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diffRess)
	}
}

func TestDiskRegisterDetachment(t *testing.T) {
	// Test:
	// - normal detachment
	// - instance/disk resource DNE
	// - error from detachHelper

	w := testWorkflow()
	s, _ := w.NewStep("s")
	att, _ := w.NewStep("att")
	w.AddDependency(s, att)
	d := &Resource{RealName: "d"}
	i := &Resource{RealName: "i"}
	i2 := &Resource{RealName: "i2"}
	disks[w].m = map[string]*Resource{"d": d}
	instances[w].m = map[string]*Resource{"i": i, "i2": i2}
	disks[w].attachments = map[*Resource]map[*Resource]*diskAttachment{
		disks[w].m["d"]: {
			instances[w].m["i"]:  {diskModeRW, att, nil},
			instances[w].m["i2"]: {diskModeRW, att, nil},
		},
	}

	errDetachHelper := func(d, i *Resource, s *Step) dErr {
		return errf("error")
	}

	tests := []struct {
		desc, d, i   string
		s            *Step
		detachHelper func(d, i *Resource, s *Step) dErr
		shouldErr    bool
	}{
		{"normal case", "d", "i", s, nil, false},
		{"disk dne case", "bad", "i", s, nil, true},
		{"instance dne case", "d", "bad", s, nil, true},
		{"detachHelper error case", "d", "i2", s, errDetachHelper, true},
	}

	for _, tt := range tests {
		disks[w].testDetachHelper = tt.detachHelper
		err := disks[w].registerDetachment(tt.d, tt.i, tt.s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have err'ed but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	// Check state.
	want := map[*Resource]map[*Resource]*diskAttachment{
		d: {
			i:  &diskAttachment{diskModeRW, att, s},
			i2: disks[w].attachments[d][i2], // Not modified.
		},
	}
	if diffRes := diff(disks[w].attachments, want); diffRes != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDiskRegisterAllDetachments(t *testing.T) {
	// Test:
	// - detachHelper error
	// - normal case

	w := testWorkflow()
	s, _ := w.NewStep("s")
	att, _ := w.NewStep("att")
	w.AddDependency(s, att)
	d := &Resource{RealName: "d"}
	d2 := &Resource{RealName: "d2"}
	i := &Resource{RealName: "i"}
	instances[w].m["i"] = i
	disks[w].attachments[d] = map[*Resource]*diskAttachment{i: {attacher: att}}
	disks[w].attachments[d2] = map[*Resource]*diskAttachment{i: {attacher: att}}

	errDetachHelper := func(d, i *Resource, s *Step) dErr {
		return errf("error")
	}

	tests := []struct {
		desc, iName  string
		s            *Step
		detachHelper func(d, i *Resource, s *Step) dErr
		shouldErr    bool
	}{
		{"detachHelper error case", "i", s, errDetachHelper, true},
		{"normal case", "i", s, nil, false},
	}

	for _, tt := range tests {
		disks[w].testDetachHelper = tt.detachHelper
		err := disks[w].registerAllDetachments(tt.iName, tt.s)
		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have erred but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}

	// Check state.
	want := map[*Resource]map[*Resource]*diskAttachment{
		d:  {i: {attacher: att, detacher: s}},
		d2: {i: {attacher: att, detacher: s}},
	}
	if diffRes := diff(disks[w].attachments, want); diffRes != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diffRes)
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
	images[w].m = map[string]*Resource{"i1": {creator: iCreator}} // "i1" resource

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
			if res, _ := disks[w].get(tt.d.Name); res != &tt.d.Resource {
				t.Errorf("%s: %q not in disk registry as expected: got=%v want=%v", tt.desc, tt.d.Name, &tt.d.Resource, res)
			}
		} else if !tt.shouldErr {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
