module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.21

require (
	cloud.google.com/go/compute v1.2.0
	cloud.google.com/go/storage v1.14.0
	github.com/GoogleCloudPlatform/compute-daisy v0.0.0-20220223233810-60345cd7065c
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0-20220201175241-7409375050b9
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0-20220126184140-b288db61775e
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.3.0
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/sys v0.14.0
	google.golang.org/api v0.67.0
	google.golang.org/protobuf v1.31.0
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/iam v0.2.0 // indirect
	cloud.google.com/go/logging v1.4.0 // indirect
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200406182414-bf9021434372 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmware/govmomi v0.24.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/oauth2 v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220207164111-0872dc986b00 // indirect
	google.golang.org/grpc v1.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common
