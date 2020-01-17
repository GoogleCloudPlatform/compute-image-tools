module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.51.0
	cloud.google.com/go/storage v1.5.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200114193104-06c8be9a6a7d
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200114194943-8842c1d85084
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20200113163233-44035fcbfdd9
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.4
	github.com/golang/mock v1.3.1
	github.com/google/logger v1.0.1
	github.com/google/uuid v1.1.1
	github.com/klauspost/pgzip v1.2.1
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.0
	github.com/stretchr/testify v1.4.0
	github.com/vmware/govmomi v0.22.1
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50
	google.golang.org/api v0.15.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
