package daisy

import (
	"context"
	"fmt"
	"testing"
)

func TestUpdateInstancesMetadataValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}

	testMetadata := map[string]string{"key": "value"}

	tests := []struct {
		desc    string
		sm     *UpdateInstancesMetadata
		wantErr bool
	}{
		{"empty metadata case", &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{}}}, true},
		{"bad instance case", &UpdateInstancesMetadata{{Instance: "bad", Metadata: testMetadata}}, true},
		{"positive flow case", &UpdateInstancesMetadata{{Instance: testInstance, Metadata: testMetadata}}, false},
	}
	for _, tt := range tests {
		err := tt.sm.validate(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}
