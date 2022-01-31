module github.com/GoogleCloudPlatform/compute-image-tools/common

go 1.13

require (
	cloud.google.com/go/storage v1.10.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy latest
	google.golang.org/api v0.31.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
