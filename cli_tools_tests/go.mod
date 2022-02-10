module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools_tests

go 1.17

require (
	cloud.google.com/go/storage v1.20.0
	github.com/GoogleCloudPlatform/compute-image-tools/cli_tools v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/common v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils v0.0.0
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/aws/aws-sdk-go v1.42.50
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/google/uuid v1.3.0
	github.com/stretchr/testify v1.7.0
	google.golang.org/api v0.68.0
	google.golang.org/protobuf v1.27.1
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/compute v1.2.0 // indirect
	cloud.google.com/go/iam v0.1.1 // indirect
	cloud.google.com/go/logging v1.4.2 // indirect
	github.com/cenkalti/backoff/v4 v4.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.10.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/minio/highwayhash v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmware/govmomi v0.27.3 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20220209195652-db638375bc3a // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220208230804-65c12eb4c068 // indirect
	google.golang.org/grpc v1.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/common => ../common

replace github.com/GoogleCloudPlatform/compute-image-tools/cli_tools => ../cli_tools

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy

replace github.com/GoogleCloudPlatform/compute-image-tools/go/e2e_test_utils => ../go/e2e_test_utils

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go
