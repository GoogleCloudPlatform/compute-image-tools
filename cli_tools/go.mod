module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/logging v1.4.0 // indirect
	cloud.google.com/go/storage v1.15.0
	cos.googlesource.com/cos/tools.git v0.0.0-20210329212435-a349a79f950d // indirect
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20210421223824-28138fd8e1a6
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200416045929-22b14b6b7c19
	github.com/GoogleCloudPlatform/compute-image-tools/proto/go v0.0.0
	github.com/GoogleCloudPlatform/guest-logging-go v0.0.0-20210408162703-fce8c8cf6383 // indirect
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20210421222842-d1bc27bfac9b
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/aws/aws-sdk-go v1.38.23
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.5
	github.com/go-playground/validator/v10 v10.5.0
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.5.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/google/logger v1.1.0
	github.com/google/uuid v1.2.0
	github.com/klauspost/compress v1.12.1 // indirect
	github.com/klauspost/pgzip v1.2.5
	github.com/kylelemons/godebug v1.1.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/minio/highwayhash v1.0.2
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/vmware/govmomi v0.25.0
	go.chromium.org/luci v0.0.0-20210421223051-ca2642d21a83 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210421223729-2554d15bf5f7 // indirect
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210421221651-33663a62ff08
	google.golang.org/api v0.45.0
	google.golang.org/genproto v0.0.0-20210421164718-3947dc264843 // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/proto/go => ../proto/go

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
