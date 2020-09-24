module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/image_import

go 1.13

require (
	cloud.google.com/go/storage v1.12.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common v0.0.0
	github.com/google/uuid v1.1.2
	github.com/stretchr/testify v1.6.1
	google.golang.org/api v0.32.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_e2e_test/common => ../common
