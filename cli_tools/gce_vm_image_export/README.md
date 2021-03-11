## Compute Engine VM Image Export

The `gce_vm_image_export` tool exports a VM image to Google Cloud Storage.
It uses Daisy to perform exports while adding additional logic to perform
export setup and clean-up, such as validating flags.

### Build
Download and install [Go](https://golang.org/doc/install). Then pull and 
install the `gce_vm_image_export` tool, this should place the binary in the 
[Go bin directory](https://golang.org/doc/code.html#GOPATH):

```
go get github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_export
```

### Flags

#### Required flags
+ `-client_id=CLIENT_ID` Identifies the client of the importer. For example: `gcloud` or
  `pantheon`.
+ `-destination_uri=DESTINATION_URI` The Google Cloud Storage URI destination for the exported
  virtual disk file. For example: gs://my-bucket/my-exported-image.vmdk.
+ `-source_image=SOURCE_IMAGE` An existing Compute Engine image URI from which to 
  export.

#### Optional flags  
+ `-format=FORMAT` Specify the format to export to, such as vmdk, vhdx, vpc, or qcow2.
+ `-project=PROJECT` Project to run in, overrides what is set in workflow.
+ `-network=NETWORK` Name of the network in your project to use for the image import. The network 
  must have access to Google Cloud Storage. If not specified, the  network named 'default' is used.
+ `-subnet=SUBNET` Name of the subnetwork in your project to use for the image import. If the 
  network resource is in legacy mode, do not provide this property. If the network is in auto subnet 
  mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this 
  field should be specified. Region or zone should be specified if this field is specified.
+ `-zone=ZONE` Zone of the image to import. The zone in which to do the work of
  importing the image. Overrides the default compute/zone property value for
  this command invocation.  
+ `-timeout=TIMEOUT` Maximum time a build can last before it is failed as "TIMEOUT". For example,
  specifying 2h will fail the process after 2 hours.
+ `-scratch_bucket_gcs_path=PATH` GCS scratch bucket to use, overrides default set in Daisy.
+ `-oauth=OAUTH_PATH` Path to oauth json file, overrides what is set in workflow.
+ `-compute_endpoint_override=ENDPOINT` Compute API endpoint to override default.
+ `-disable_gcs_logging` Do not stream logs to GCS
+ `-disable_cloud_logging` Do not stream logs to Cloud Logging
+ `-disable_stdout_logging` Do not display individual workflow logs on stdout
+ `-labels=[KEY=VALUE,...]` labels: List of label KEY=VALUE pairs to add. Keys must start with a
  lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and 
  numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.
+ `-compute_service_account` Compute service account to be used by exporter 
  Virtual Machine. When empty, the Compute Engine default service account is used.
+ `-client_version` Identifies the version of the client of the exporter
  
### Usage

```
gce_vm_image_export -client_id=CLIENT_ID -destination_uri=DESTINATION_URI
        -source_image=SOURCE_IMAGE [-format=FORMAT] [-project=PROJECT] [-network=NETWORK]
        [-subnet=SUBNET] [-zone=ZONE] [-timeout=TIMEOUT] [-scratch_bucket_gcs_path=PATH]
        [-oauth=OAUTH_PATH] [-compute_endpoint_override=ENDPOINT] [-disable_gcs_logging]
        [-disable_cloud_logging] [-disable_stdout_logging] [-labels=KEY=VALUE,...]
        [-compute_service_account=COMPUTE_SERVICE_ACCOUNT] [-client_version]
```
