module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.15.0
	cos.googlesource.com/cos/tools.git v0.0.0-20210503174137-420f9145f047 // indirect
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20210506201125-d5d70336aaff
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/GoogleCloudPlatform/guest-logging-go v0.0.0-20210506184106-4c72f3089987 // indirect
  github.com/GoogleCloudPlatform/osconfig v0.0.0-20210430160431-53ca2c974ef2
	github.com/aws/aws-sdk-go v1.38.37
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.6.0
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang/glog v0.0.0-20210429001901-424d2337a529 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/google/logger v1.1.1
	github.com/google/uuid v1.2.0
	github.com/klauspost/compress v1.12.2 // indirect
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/minio/highwayhash v1.0.2
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/vmware/govmomi v0.25.0
  golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
  golang.org/x/sys v0.0.0-20210510120138-977fb7262007
	google.golang.org/api v0.46.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
