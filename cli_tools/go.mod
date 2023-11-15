module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.17

require (
	cloud.google.com/go/compute v1.20.1
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/storage v1.31.0
	github.com/GoogleCloudPlatform/compute-daisy v0.0.0-20230630215637-031fb762c645
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0-20220201175241-7409375050b9
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0-20230629161555-92bb8f0f9211
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20210202205636-8f5a30e8969f
	github.com/aws/aws-sdk-go v1.37.5
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/dustin/go-humanize v1.0.1
	github.com/go-ole/go-ole v1.2.6
	github.com/go-playground/validator/v10 v10.14.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/google/go-cmp v0.5.9
	github.com/google/logger v1.1.0
	github.com/google/uuid v1.3.0
	github.com/klauspost/pgzip v1.2.6
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.2
	github.com/stretchr/testify v1.8.2
	github.com/vmware/govmomi v0.24.0
	golang.org/x/sync v0.3.0
	golang.org/x/sys v0.9.0
	google.golang.org/api v0.129.0
	google.golang.org/protobuf v1.31.0
)

require (
	cloud.google.com/go v0.110.3 // indirect
	cloud.google.com/go/iam v1.1.1 // indirect
	cloud.google.com/go/logging v1.7.0 // indirect
	cloud.google.com/go/longrunning v0.5.1 // indirect
	cos.googlesource.com/cos/tools.git v0.0.0-20210104210903-4b3bc7d49b79 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	github.com/klauspost/compress v1.16.6 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	go.chromium.org/luci v0.0.0-20210204234011-34a994fe5aec // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/net v0.11.0 // indirect
	golang.org/x/oauth2 v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/grpc v1.56.1 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common
