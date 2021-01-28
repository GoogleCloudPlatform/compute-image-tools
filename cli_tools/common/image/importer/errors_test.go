package importer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_createError(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		name           string
		fromUser       string
		detectedDistro string
		detectedMajor  string
		detectedMinor  string
		original       error
		wantErr        string
	}{
		{
			name:           "return original error - when detection matches actual",
			fromUser:       "centos-7",
			detectedDistro: "centos",
			detectedMajor:  "7",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when failure to parse user's input",
			fromUser:       "not-a-distro",
			detectedDistro: "centos",
			detectedMajor:  "7",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return new error - when detection doesn't match actual - and detection fully specified",
			fromUser:       "ubuntu-1804",
			detectedDistro: "centos",
			detectedMajor:  "7",
			detectedMinor:  "",
			original:       cause,
			wantErr: "\"centos-7\" was detected on your disk, but \"ubuntu-1804\" was specified. " +
				"Please verify and re-import",
		},
		{
			name:           "return original error - when detection empty",
			fromUser:       "ubuntu-1804",
			detectedDistro: "",
			detectedMajor:  "",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when detection empty",
			fromUser:       "ubuntu-1804",
			detectedDistro: "unknown",
			detectedMajor:  "",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when detected debian has major version 0",
			fromUser:       "debian-9",
			detectedDistro: "debian",
			detectedMajor:  "0",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when detected rhel has major version 0",
			fromUser:       "rhel-7",
			detectedDistro: "rhel",
			detectedMajor:  "0",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
		{
			name:           "return original error - when detected centos has major version 0",
			fromUser:       "centos-7",
			detectedDistro: "centos",
			detectedMajor:  "0",
			detectedMinor:  "",
			original:       cause,
			wantErr:        cause.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := customizeErrorToDetectionResults(tt.fromUser, tt.detectedDistro, tt.detectedMajor, tt.detectedMinor, tt.original)
			assert.EqualError(t, actual, tt.wantErr)
		})
	}
}
