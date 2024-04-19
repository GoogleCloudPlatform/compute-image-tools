module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.21

require (
	cloud.google.com/go/compute/metadata v0.2.3
	cloud.google.com/go/storage v1.30.1
	github.com/GoogleCloudPlatform/compute-daisy v0.0.0-20220223233810-60345cd7065c
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0-20220201175241-7409375050b9
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0-20220126184140-b288db61775e
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.1
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.15.5
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.4.0
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/sys v0.18.0
	google.golang.org/api v0.149.0
	google.golang.org/protobuf v1.31.0
)

require (
	cloud.google.com/go v0.110.8 // indirect
	cloud.google.com/go/compute v1.23.2 // indirect
	cloud.google.com/go/iam v1.1.4 // indirect
	cloud.google.com/go/logging v1.8.1 // indirect
	cloud.google.com/go/longrunning v0.5.2 // indirect
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200406182414-bf9021434372 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/vmware/govmomi v0.24.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.13.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/grpc v1.59.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common
