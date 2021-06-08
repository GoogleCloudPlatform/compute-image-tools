package importer

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"

	"github.com/stretchr/testify/assert"
)

func Test_customizeErrorToDetectionResults(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		name           string
		fromUser       string
		detectedDistro distro.Release
		original       error
		wantErr        string
	}{
		{
			name:           "return original error - when detection matches actual",
			fromUser:       "centos-7",
			detectedDistro: distro.FromGcloudOSArgumentMustParse("centos-7"),
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when failure to parse user's input",
			fromUser:       "not-a-distro",
			detectedDistro: distro.FromGcloudOSArgumentMustParse("centos-7"),
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return new error - when detection doesn't match actual - and detection fully specified",
			fromUser:       "ubuntu-1804",
			detectedDistro: distro.FromGcloudOSArgumentMustParse("centos-7"),
			original:       cause,
			wantErr: "\"centos-7\" was detected on your disk, but \"ubuntu-1804\" was specified. " +
				"Please verify and re-import",
		},
		{
			name:           "return original error - when detection empty",
			fromUser:       "ubuntu-1804",
			detectedDistro: nil,
			original:       cause,
			wantErr:        cause.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := customizeErrorToDetectionResults(tt.fromUser, tt.detectedDistro, tt.original)
			assert.EqualError(t, actual, tt.wantErr)
		})
	}
}
