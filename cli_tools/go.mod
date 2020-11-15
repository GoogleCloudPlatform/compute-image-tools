module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.65.0
	cloud.google.com/go/logging v1.1.0 // indirect
	cloud.google.com/go/storage v1.11.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200914170536-737b13cffca0
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200414213327-359251a2c860
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20200721210327-c9ab1b6aeb02
	github.com/aws/aws-sdk-go v1.34.22
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.4
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2
	github.com/google/logger v1.1.0
	github.com/google/uuid v1.1.2
	github.com/klauspost/compress v1.11.0 // indirect
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.0
	github.com/stretchr/testify v1.5.1
	github.com/vmware/govmomi v0.23.1
	golang.org/x/exp v0.0.0-20200331195152-e8c3332aa8e5 // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirecresource_labeler_testt
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sys v0.0.0-20200909081042-eff7692f9009
	golang.org/x/tools v0.0.0-20200914161755-17fc728d0d1e // indirect
	google.golang.org/api v0.31.0
	google.golang.org/genproto v0.0.0-20200911024640-645f7a48b24f // indirect
	google.golang.org/grpc v1.32.0 // indirect
	google.golang.org/protobuf v1.25.0
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go
