## Compute Engine OVF Export

The `gce_ovf_export` tool exports a Google Compute Engine VM or a Google Compute
Engine machine image to a virtual appliance in OVF format. It supports exporting
OVF and OVA archives.

The following configurations of the OVF virtual appliance are exported:
+ Virtual Disks (represented by the DiskSection of the OVF format) 
+ CPU and Memory (represented by the ResourceAllocationSection of the OVF 
format). If the CPU/memory configuration are out of bounds of the supported range in Compute Engine,
export process will set the respective configurations to the max possible. 
+ Boot Disk (represented by the BootDeviceSection of the OVF format) 
+ Guest OS (represented by the OperatingSystemSection of the OVF format) 


### Build
Download and install [Go](https://golang.org/doc/install). Then pull and 
install the `gce_ovf_export` tool, this should place the binary in the 
[Go bin directory](https://golang.org/doc/code.html#GOPATH):

```
go get github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export
```

### Flags

#### Required flags
+ `-destination-uri` GCS path to the exported OVF package or OVA archive.

Exactly one of these must be specified:
+ `-instance-name` Name of the VM instances to export.
+ `-machine-image-name` Name of the machine image to export.

#### Optional flags
+ `-client-id` Identifies the client of the OVF exporter. For example: `gcloud` or
  `pantheon`.
+ `-ovf-format=OVF_FORMAT` One of: `ovf` or `ova`. Defaults to `ovf`. If `ova`
  is specified, exported OVF package will be packed as an OVA archive and
  individual files will be removed from GCS.  
+ `-disk-export-format=DISK_FORMAT` format for disks in OVF, such as vmdk, vhdx,
  vpc, or qcow2. Any format supported by qemu-img is supported by OVF export.
  Defaults to `vmdk`.
+ `--os=OS` Operating system to be set in OVF descriptor. Overrides the value
  detected by the exporter. Integer based on CIM operating system found at
  https://schemas.dmtf.org/wbem/cim-html/2.51.0/CIM_OperatingSystem.html
+ `-network-tier=NETWORK_TIER` Specifies the network tier that will be used to configure the 
  instance. NETWORK_TIER must be one of: PREMIUM, STANDARD. The default value is PREMIUM.
+ `-subnet=SUBNET` Name of the subnetwork in your project to use for the image export. If	the
  network resource is in legacy mode, do not provide this property. If the network is in auto subnet
  mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this
  field should be specified. Zone should be specified if this field is specified.
+ `-timeout=TIMEOUT; default="2h"` Maximum time an export can last before it is failed as TIMEOUT.
  For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for
  information on duration formats.
+ `-project=PROJECT` project to run in, overrides what is set in workflow
+ `-scratch-bucket-gcs-path=PATH` GCS scratch bucket to use, overrides what is set in workflow
+ `-oauth=OAUTH_PATH` path to oauth json file.
+ `-compute-endpoint-override=ENDPOINT` API endpoint to override default
+ `-disable-gcs-logging` do not stream logs to GCS
+ `-disable-cloud-logging` do not stream logs to Cloud Logging
+ `-disable-stdout-logging` do not display individual workflow logs on stdout
+ `-client-version` identifies the version of the client of the exporter

### Usage

Export a VM instance:
```
gce_ovf_export -destination-uri=GCS_PATH -instance-name=INSTANCE_NAME
[-client-id=CLIENT_ID] [-ovf-format=OVF_FORMAT]
[-disk-export-format=DISK_FORMAT] [--os] [-network=NETWORK] [-subnet=SUBNET]
[-timeout=TIMEOUT; default="2h"] [-project=PROJECT]
[-scratch-bucket-gcs-path=SCRATCH_BUCKET_PATH] [-oauth=OAUTH_FILE_PATH]
[-compute-endpoint-override=CE_ENDPOINT] [-disable-gcs-logging] 
[-disable-cloud-logging] [-disable-stdout-logging] [-client-version]

```

Export a machine image:
```
gce_ovf_export -destination-uri=GCS_PATH -machine-image-name=MACHINE_IMAGE 
[-client-id=CLIENT_ID] [-ovf-format=OVF_FORMAT] 
[-disk-export-format=DISK_FORMAT] [--os] [-network=NETWORK] [-subnet=SUBNET]
[-timeout=TIMEOUT; default="2h"] [-project=PROJECT]
[-scratch-bucket-gcs-path=SCRATCH_BUCKET_PATH] [-oauth=OAUTH_FILE_PATH]
[-compute-endpoint-override=CE_ENDPOINT] [-disable-gcs-logging] 
[-disable-cloud-logging] [-disable-stdout-logging] [-client-version]

