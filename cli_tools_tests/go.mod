module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests

go 1.13

require (
	cloud.google.com/go/storage v1.11.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0
	github.com/aws/aws-sdk-go v1.34.22
	github.com/google/uuid v1.1.2
	github.com/stretchr/testify v1.5.1
	google.golang.org/api v0.31.0
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy

replace github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils => ../go/e2e_test_utils
