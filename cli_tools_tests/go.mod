module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests

go 1.13

require (
	cloud.google.com/go/storage v1.14.0
	github.com/GoogleCloudPlatform/compute-daisy v0.0.0-20220223233810-60345cd7065c
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/aws/aws-sdk-go v1.37.5
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/google/uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	google.golang.org/api v0.67.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils => ../go/e2e_test_utils

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go
