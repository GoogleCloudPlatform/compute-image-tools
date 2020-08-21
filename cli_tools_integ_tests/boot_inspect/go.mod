module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_integ_test/boot_inspect

go 1.13

require (
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0
	github.com/google/uuid v1.1.2
	github.com/stretchr/testify v1.6.1
	google.golang.org/api v0.31.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../../daisy
