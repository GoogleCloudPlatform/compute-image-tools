module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests

go 1.13

require (
	cloud.google.com/go/logging v1.4.2 // indirect
	cloud.google.com/go/storage v1.20.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/aws/aws-sdk-go v1.42.50
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/go-playground/validator/v10 v10.10.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/google/uuid v1.3.0
	github.com/stretchr/testify v1.7.0
	github.com/vmware/govmomi v0.27.3 // indirect
	golang.org/x/crypto v0.0.0-20220209195652-db638375bc3a // indirect
	google.golang.org/api v0.68.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy

replace github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils => ../go/e2e_test_utils

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go
