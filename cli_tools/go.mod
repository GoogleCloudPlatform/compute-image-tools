module github.com/GoogleCloudPlatform/compute-image-tools/cli_tools

go 1.13

require (
	cloud.google.com/go v0.53.0
	cloud.google.com/go/storage v1.5.0
	github.com/GoogleCloudPlatform/compute-image-tools/daisy v0.0.0-20200219213738-41e8aea99239
	github.com/GoogleCloudPlatform/compute-image-tools/mocks v0.0.0-20200206014411-426b6301f679
	github.com/GoogleCloudPlatform/osconfig v0.0.0-20200211005319-080372593330
	github.com/dustin/go-humanize v1.0.0
	github.com/go-ole/go-ole v1.2.4
	github.com/golang/mock v1.4.0
	github.com/google/logger v1.0.1
	github.com/google/uuid v1.1.1
	github.com/klauspost/compress v1.10.1 // indirect
	github.com/klauspost/pgzip v1.2.1
	github.com/kylelemons/godebug v1.1.0
	github.com/minio/highwayhash v1.0.0
	github.com/stretchr/testify v1.4.0
	github.com/vmware/govmomi v0.22.2
	go.opencensus.io v0.22.3 // indirect
	golang.org/x/exp v0.0.0-20200213203834-85f925bdd4d0 // indirect
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/net v0.0.0-20200219183655-46282727080f // indirecresource_labeler_testt
	golang.org/x/sys v0.0.0-20200219091948-cb0a6d8edb6c
	golang.org/x/tools v0.0.0-20200220155224-947cbf191135 // indirect
	google.golang.org/api v0.17.0
	google.golang.org/genproto v0.0.0-20200218151345-dad8c97a84f5 // indirect
	google.golang.org/grpc v1.27.1 // indirect
)

replace github.com/GoogleCloudPlatform/compute-image-tools/daisy => ../daisy
