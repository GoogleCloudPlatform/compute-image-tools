## Compute Engine OVF Import

The `gce_ovf_import` tool imports a virtual appliance in OVF format created in VMware environments
to Google Compute Engine VM. It supports importing OVF and OVA archives.

The following configurations of the OVF virtual appliance are imported:
+ Virtual Disks (represented by the DiskSection of the OVF format) 
+ CPU and Memory (represented by the ResourceAllocationSection of the OVF format). If the 
CPU/memory configuration are out of bounds of the supported range in Compute Engine,
import process will set the respective configurations to the max possible. 
+ Boot Disk (represented by the BootDeviceSection of the OVF format) 
+ Guest OS (represented by the OperatingSystemSection of the OVF format) 


### Build
Download and install [Go](https://golang.org/doc/install). Then pull and 
install the `gce_ovf_import` tool, this should place the binary in the 
[Go bin directory](https://golang.org/doc/code.html#GOPATH):

```
go get github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import
```

### Flags

#### Required flags
+ `-instance-names` Name of the VM instances to create.
+ `-client-id` Identifies the client of the OVF importer. For example: `gcloud` or
  `pantheon`.
+ `-ovf-gcs-path` GCS path to OVF descriptor, OVA file or a directory with OVF package.
  
#### Optional flags
+ `-no-guest-environment` Google Guest Environment will not be installed on the image
+ `-can-ip-forward` If provided, allows the instances to send and receive packets with non-matching
  destination or source IP addresses.
+ `-deletion-protection` Enables deletion protection for the instance.
+ `-description=DESCRIPTION` Specifies a textual description of the instances.
+ `-labels=[KEY=VALUE,...]` List of label KEY=VALUE pairs to add. Keys must start with a lowercase
  character and contain only hyphens (-), underscores (_), lowercase characters, and numbers. Values
  must contain only hyphens (-), underscores (_), lowercase characters, and numbers.
+ `-machine-type=MACHINE_TYPE` Specifies the machine type used for the instance. To get a list of
  available machine types, run 'gcloud compute machine-types list'. If unspecified, the default type
  is n1-standard-1.
+ `-network=NETWORK` Name of the network in your project to use for the image import. The network
  must have access to Google Cloud Storage. If not specified, the network named default is used. If
  -subnet is also specified subnet must be a subnetwork of network specified by -network.
+ `-network-tier=NETWORK_TIER` Specifies the network tier that will be used to configure the 
  instance. NETWORK_TIER must be one of: PREMIUM, STANDARD. The default value is PREMIUM.
+ `-subnet=SUBNET` Name of the subnetwork in your project to use for the image import. If	the
  network resource is in legacy mode, do not provide this property. If the network is in auto subnet
  mode, providing the subnetwork is optional. If the network is in custom subnet mode, then this
  field should be specified. Zone should be specified if this field is specified.
+ `-private-network-ip=PRIVATE_NETWORK_IP` Specifies the RFC1918 IP to assign to the instance. The
  IP should be in the subnet or legacy network IP range.
+ `-no-external-ip` Specifies that VPC into which instances is being imported doesn't allow external
  IPs.
+ `-no-restart-on-failure` the instance will not be restarted if it’s terminated by Compute Engine.
  This does not affect terminations performed by the user.
+ `-os=OS` Specifies the OS of the image being imported. 
  OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, 
  rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol,
  windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016,
  windows-2016-byol, windows-7-byol, windows-2019, windows-2019-byol, windows-8-1-x64-byol.
+ `-shielded-integrity-monitoring` Enables monitoring and attestation of the boot integrity of the
  instance. The attestation is performed against the integrity policy baseline. This baseline is
  initially derived from the implicitly trusted boot image when the instance is created. This
  baseline can be updated by using -shielded-vm-learn-integrity-policy.
+ `-shielded-secure-boot` The instance will boot with secure boot enabled.
+ `-shielded-vtpm` The instance will boot with the TPM (Trusted Platform Module) enabled. A TPM is a
  hardware module that can be used for different security operations such as remote attestation,
  encryption and sealing of keys.
+ `-tags=TAG,[TAG,…]` Specifies a list of tags to apply to the instance. These tags allow network
  firewall rules and routes to be applied to specified VM instances. See
  `gcloud compute firewall-rules create` for more details.
+ `-zone=ZONE` Zone of the image to import. The zone in which to do the work of importing the image.
  Overrides the default compute/zone property value for this command invocation
+ `-boot-disk-kms-key=BOOT_DISK_KMS_KEY` The Cloud KMS (Key Management Service) cryptokey that will
  be used to protect the disk. The arguments in this group can be used to specify the attributes of
  this resource. ID of the key or fully qualified identifier for the key. This flag must be
  specified if any of the other arguments in this group are specified.
+ `-boot-disk-kms-keyring=BOOT_DISK_KMS_KEYRING` The KMS keyring of the key.
+ `-boot-disk-kms-location=BOOT_DISK_KMS_LOCATION` The Cloud location for the key.
+ `-boot-disk-kms-project=BOOT_DISK_KMS_PROJECT` The Cloud project for the key.
+ `-timeout=TIMEOUT; default="2h"` Maximum time a build can last before it is failed as TIMEOUT.
  For example, specifying 2h will fail the process after 2 hours. See `gcloud topic datetimes` for
  information on duration formats.
+ `-project=PROJECT` project to run in, overrides what is set in workflow
+ `-scratch-bucket-gcs-path=PATH` GCS scratch bucket to use, overrides what is set in workflow
+ `-oauth=OAUTH_PATH` path to oauth json file.
+ `-compute-endpoint-override=ENDPOINT` API endpoint to override default
+ `-disable-gcs-logging` do not stream logs to GCS
+ `-disable-cloud-logging` do not stream logs to Cloud Logging
+ `-disable-stdout-logging` do not display individual workflow logs on stdout
+ `-node-affinity-label` Node affinity label used to determine sole tenant node to schedule this instance on. Label is of the format: <key>,<operator>,<value>,<value2>... where <operator> can be one of: IN, NOT. For example: workload,IN,prod,test is a label with key 'workload' and values 'prod' and 'test'. This flag can be specified multiple times for multiple labels.
+ `-release-track` Release track of OVF import. One of: %s, %s or %s. Impacts which compute API release track is used by the import tool.

### Usage

```
gce_ovf_import -instance-names=INSTANCE_NAME -client-id=CLIENT_ID 
-source-uri=OVF_GCS_FILE_PATH
[-can-ip-forward]
[-custom-cpu=CUSTOM_CPU -custom-memory=CUSTOM_MEMORY]
[-deletion-protection] [-description=DESCRIPTION]
[-labels=[KEY=VALUE,…]]
[-machine-type=MACHINE_TYPE]
[-network=NETWORK] [-network-interface=[PROPERTY=VALUE,…]]
[-network-tier=NETWORK_TIER] 
[-subnet=SUBNET]
[-private-network-ip=PRIVATE_NETWORK_IP] 
[-no-external-ip]
[-no-restart-on-failure]
[-os=OS]
[-shielded-integrity-monitoring] [-shielded-secure-boot] [-shielded-vtpm]
[-tags=TAG,[TAG,…]] 
[-zone=ZONE] 
[-address=ADDRESS    | -no-address]
[-boot-disk-kms-key=KMS_KEY : -boot-disk-kms-keyring=KMS_KEYRING
 -boot-disk-kms-location=KMS_LOCATION -boot-disk-kms-project=KMS_PROJECT]
[-timeout=TIMEOUT; default="2h"] [-project=PROJECT]
[-scratch-bucket-gcs-path=SCRATCH_BUCKET_PATH] [-oauth=OAUTH_FILE_PATH]
[-compute-endpoint-override=CE_ENDPOINT] [-disable-gcs-logging] 
[-disable-cloud-logging] [-disable-stdout-logging] [-no-guest-environment]

```
