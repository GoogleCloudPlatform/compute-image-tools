module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.17

require (
	cloud.google.com/go/compute v1.2.0
	cloud.google.com/go/storage v1.14.0
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0-20220201175241-7409375050b9
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200406182414-bf9021434372
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0-20220126184140-b288db61775e
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20210202205636-8f5a30e8969f
	github.com/aws/aws-sdk-go v1.37.5
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.4.1
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/google/logger v1.1.0
	github.com/google/uuid v1.2.0
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.1
	github.com/stretchr/testify v1.7.0
	github.com/vmware/govmomi v0.24.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158
	google.golang.org/api v0.68.0
	google.golang.org/protobuf v1.27.1
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/iam v0.1.1 // indirect
	cloud.google.com/go/logging v1.2.0 // indirect
	cos.googlesource.com/cos/tools.git v0.0.0-20210104210903-4b3bc7d49b79 // indirect
	github.com/GoogleCloudPlatform/guest-logging-go v0.0.0-20200113214433-6cbb518174d4 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.7.0 // indirect
	go.chromium.org/luci v0.0.0-20210204234011-34a994fe5aec // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220208230804-65c12eb4c068 // indirect
	google.golang.org/grpc v1.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common
