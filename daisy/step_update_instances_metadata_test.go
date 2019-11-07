package daisy

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestUpdateInstancesMetadataValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}

	tests := []struct {
		desc    string
		sm      *UpdateInstancesMetadata
		wantErr bool
	}{
		{"empty metadata case", &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{}}}, true},
		{"bad instance case", &UpdateInstancesMetadata{{Instance: "bad", Metadata: map[string]string{"key": "value"}}}, true},
		{"positive flow case", &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{"key": "value"}}}, false},
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

func computeMetataToMap(md compute.Metadata) map[string]string {
	mdMap := map[string]string{}
	for _, item := range md.Items {
		mdMap[item.Key] = *item.Value
	}
	return mdMap
}

func mapToComputeMetadata(md map[string]string) compute.Metadata {
	mdComp := compute.Metadata{}
	for k, v := range md {
		vCopy := v
		mdComp.Items = append(mdComp.Items, &compute.MetadataItems{Key: k, Value: &vCopy})
	}
	return mdComp
}

func TestUpdateInstancesMetadataRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}

	tests := []struct {
		desc             string
		originalMetadata map[string]string
		expectedMetadata map[string]string
		sm               *UpdateInstancesMetadata
		wantErr          bool
		getInstErr       error
		setMetaErr       error
	}{
		{"blank case", map[string]string{}, map[string]string{}, &UpdateInstancesMetadata{}, false, nil, nil},
		{"Add metadata case", map[string]string{"orig1": "value1"}, map[string]string{"orig1": "value1", "new1": "value2"}, &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{"new1": "value2"}}}, false, nil, nil},
		{"override metadata case", map[string]string{"key1": "value1"}, map[string]string{"key1": "value2"}, &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{"key1": "value2"}}}, false, nil, nil},
		{"get instance error case", map[string]string{}, map[string]string{}, &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{"key1": "value1"}}}, true, Errf("error"), nil},
		{"set metadata error case", map[string]string{}, map[string]string{"key1": "value1"}, &UpdateInstancesMetadata{{Instance: testInstance, Metadata: map[string]string{"key1": "value1"}}}, true, nil, Errf("error")},
	}
	for _, tt := range tests {
		originalCompMetadata := mapToComputeMetadata(tt.originalMetadata)
		instance := compute.Instance{Metadata: &originalCompMetadata}
		mockGetInstance := func(_ string, _ string, _ string) (*compute.Instance, error) { return &instance, tt.getInstErr }

		var gotM compute.Metadata
		mockSetInstanceMetadata := func(_ string, _ string, _ string, md *compute.Metadata) error { gotM = *md; return tt.setMetaErr }
		w.ComputeClient = &daisyCompute.TestClient{GetInstanceFn: mockGetInstance, SetInstanceMetadataFn: mockSetInstanceMetadata}
		err := tt.sm.run(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
		resMetadata := computeMetataToMap(gotM)
		if !reflect.DeepEqual(tt.expectedMetadata, resMetadata) {
			t.Errorf("%s: expected metadata %v, got %v", tt.desc, tt.expectedMetadata, resMetadata)
		}
	}
}
