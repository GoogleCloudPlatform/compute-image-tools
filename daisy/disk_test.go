package daisy

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

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

	w := testWorkflow()
	iw := w.NewIncludedWorkflow()
	iw2 := w.NewIncludedWorkflow()
	iwStep, _ := w.NewStep("iwStep")
	iwStep.IncludeWorkflow = &IncludeWorkflow{w: iw}
	iwSubStep, _ := iw.NewStep("iwSubStep")
	iw2Step, _ := w.NewStep("iw2Step")
	iw2Step.IncludeWorkflow = &IncludeWorkflow{w: iw2}
	iw2SubStep, _ := iw2.NewStep("iw2SubStep")
	s, _ := w.NewStep("s")
	s2, _ := w.NewStep("s2")
	att, _ := w.NewStep("att")
	det, _ := w.NewStep("det")
	w.AddDependency("det", "att")
	w.AddDependency("s", "det")
	w.AddDependency("iw2Step", "iwStep")
	d := &resource{real: "d"}
	dDetached := &resource{real: "dDetached"}
	dIWDetached := &resource{real: "dIWDetached"}
	dIWAttached := &resource{real: "dIWAttached"}
	disks[w].m = map[string]*resource{"d": d, "dDetached": dDetached, "dIWDetached": dIWDetached, "dIWAttached": dIWAttached}
	i := &resource{real: "i"}
	i2 := &resource{real: "i2"}
	i3 := &resource{real: "i3"}
	iPrevAtt := &resource{real: "iPrevAtt"}
	iIW := &resource{real: "iIW"}
	iIW2 := &resource{real: "iIW2"}
	instances[w].m = map[string]*resource{"i": i, "i2": i2, "i3": i3, "iPrevAtt": iPrevAtt, "iIW": iIW, "iIW2": iIW2}
	disks[w].attachments = map[*resource]map[*resource]*diskAttachment{
		dDetached:   {iPrevAtt: {mode: diskModeRW, attacher: att, detacher: det}},
		dIWDetached: {iIW: {mode: diskModeRW, detacher: iwSubStep}},
		dIWAttached: {iIW: {mode: diskModeRW}},
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

	// Prep for pretty.Compare (infinite recursion issue)
	s.w = nil
	att.w = nil
	det.w = nil
	iw.parent = nil
	iw2.parent = nil
	iwStep.w = nil
	iw2Step.w = nil
	iwSubStep.w = nil
	iw2SubStep.w = nil

	want := map[*resource]map[*resource]*diskAttachment{
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
	if diff := pretty.Compare(disks[w].attachments, want); diff != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diff)
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
	w.AddDependency("s", "att")
	w.AddDependency("det", "att")
	d := &resource{real: "d"}
	i := &resource{real: "i"}
	i2 := &resource{real: "i2"}
	i3 := &resource{real: "i3"}

	if err := disks[w].detachHelper(d, i, s); err == nil {
		t.Error("detaching i from d (which has no attachments) should have failed")
	}
	disks[w].attachments[d] = map[*resource]*diskAttachment{
		i:  {attacher: att},
		i2: {attacher: att, detacher: det},
	}

	tests := []struct {
		desc      string
		d, i      *resource
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

	// pretty.Compare prep
	s.w = nil
	bad.w = nil
	att.w = nil
	det.w = nil

	// Check state.
	want := map[*resource]map[*resource]*diskAttachment{
		d: {
			i:  {attacher: att, detacher: s},
			i2: disks[w].attachments[d][i2],
		},
	}
	if diff := pretty.Compare(disks[w].attachments, want); diff != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diff)
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
	w.AddDependency("s", "att")
	d := &resource{real: "d"}
	i := &resource{real: "i"}
	i2 := &resource{real: "i2"}
	disks[w].m = map[string]*resource{"d": d}
	instances[w].m = map[string]*resource{"i": i, "i2": i2}
	disks[w].attachments = map[*resource]map[*resource]*diskAttachment{
		disks[w].m["d"]: {
			instances[w].m["i"]:  {diskModeRW, att, nil},
			instances[w].m["i2"]: {diskModeRW, att, nil},
		},
	}

	errDetachHelper := func(d, i *resource, s *Step) error {
		return errorf("error")
	}

	tests := []struct {
		desc, d, i   string
		s            *Step
		detachHelper func(d, i *resource, s *Step) error
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

	// pretty.Compare prep
	s.w = nil
	att.w = nil

	// Check state.
	want := map[*resource]map[*resource]*diskAttachment{
		d: {
			i:  &diskAttachment{diskModeRW, att, s},
			i2: disks[w].attachments[d][i2], // Not modified.
		},
	}
	if diff := pretty.Compare(disks[w].attachments, want); diff != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diff)
	}
}

func TestDiskRegisterAllDetachments(t *testing.T) {
	// Test:
	// - detachHelper error
	// - normal case

	w := testWorkflow()
	s, _ := w.NewStep("s")
	att, _ := w.NewStep("att")
	w.AddDependency("s", "att")
	d := &resource{real: "d"}
	d2 := &resource{real: "d2"}
	i := &resource{real: "i"}
	instances[w].m["i"] = i
	disks[w].attachments[d] = map[*resource]*diskAttachment{i: {attacher: att}}
	disks[w].attachments[d2] = map[*resource]*diskAttachment{i: {attacher: att}}

	errDetachHelper := func(d, i *resource, s *Step) error {
		return errorf("error")
	}

	tests := []struct {
		desc, iName  string
		s            *Step
		detachHelper func(d, i *resource, s *Step) error
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

	// pretty.Compare prep
	s.w = nil
	att.w = nil

	// Check state.
	want := map[*resource]map[*resource]*diskAttachment{
		d:  {i: {attacher: att, detacher: s}},
		d2: {i: {attacher: att, detacher: s}},
	}
	if diff := pretty.Compare(disks[w].attachments, want); diff != "" {
		t.Errorf("attachments not modified as expected: (-got,+want)\n%s", diff)
	}
}
