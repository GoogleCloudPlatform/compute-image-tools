module github.com/GoogleCloudPlatform/compute-image-tools/gce_image_import_export_tests

go 1.13

require (
	cloud.google.com/go/storage v1.5.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0-20200131234344-9a5cb7b7d72d
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200214212030-cdab85f8f241
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0-20200128181915-c0775e429077
	google.golang.org/api v0.17.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
