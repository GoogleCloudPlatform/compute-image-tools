module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/logging v1.2.0 // indirect
	cloud.google.com/go/storage v1.14.0
	cos.googlesource.com/cos/tools.git v0.0.0-20210104210903-4b3bc7d49b79 // indirect
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20210519521119-aeddd2df9d23
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20210202205636-8f5a30e8969f
	github.com/aws/aws-sdk-go v1.37.5
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.1
	github.com/google/go-cmp v0.5.5
	github.com/google/logger v1.1.0
	github.com/google/uuid v1.2.0
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/minio/highwayhash v1.0.1
	github.com/stretchr/testify v1.6.1
	github.com/vmware/govmomi v0.24.0
	go.chromium.org/luci v0.0.0-20210204234011-34a994fe5aec // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57
	google.golang.org/api v0.44.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
