## Compute Engine One-step Image Import

The `gce_onestep_image_import` tool imports a VM image from other cloud providers to Google Compute Engine
image. It uses Daisy to perform imports while adding additional logic to perform
import setup and clean-up, such as creating a temporary bucket, validating
flags etc.  

### Build
Download and install [Go](https://golang.org/doc/install). Then pull and 
install the `gce_onestep_image_import` tool, this should place the binary in the 
[Go bin directory](https://golang.org/doc/code.html#GOPATH):

```
go get github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_onestep_image_import
```

### Flags

#### Required flags
+ `-image_name=IMAGE_NAME` Name of the disk image to create.
+ `-client_id=CLIENT_ID` Identifies the client of the importer. For example: `gcloud` or
  `pantheon`.
+ `-os=OS` Specifies the OS of the image being imported.
  This must be specified if cloud provider is specified.
  OS must be one of: centos-6, centos-7, debian-8, debian-9, rhel-6, rhel-6-byol, rhel-7, 
  rhel-7-byol, ubuntu-1404, ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol,
  windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, windows-2016,
  windows-2016-byol, windows-7-byol.
  
To import from AWS, all of these must be specified:
+ `-aws_access_key_id=AWS_ACCESS_KEY_ID` The access key ID for an AWS credential.
  This credential is associated with an IAM user or role.
  This IAM user must have permissions to import images.
+ `-aws_secret_access_key=AWS_SECRET_ACCESS_KEY` The secret access key for an AWS credential.
  This credential is associated with an IAM user or role.
  This IAM user must have permissions to import images.
+ `-aws_session_token=AWS_SESSION_TOKEN The session token for an AWS credential.
  This credential is associated with an IAM user or role.
  This IAM user must have permissions to import images.
+ `-aws_region=AWS_REGION` The AWS region for the image that you want to import.

To import from AWS, exactly one of the groups must be specified:

+ To import from an exported image file in S3:
    + `-aws_source_ami_file_path=AWS_SOURCE_AMI_FILE_PATH` The S3 resource path of
      the exported image file.

+ To import from a VM instance:
    + `-aws_ami_id=AWS_AMI_ID` The AWS AMI ID of the image to import.
    + `-aws_ami_export_location=AWS_AMI_EXPORT_LOCATION` The AWS S3 Bucket location
      where you want to export the image.

#### Optional flags
+ `-no_guest_environment` Google Guest Environment will not be installed on the image.
+ `-family=FAMILY` Family to set for the translated image.
+ `-description=DESCRIPTION` Description to set for the translated image.
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
+ `-project=PROJECT` Project to run in, overrides what is set in workflow.
+ `-scratch_bucket_gcs_path=PATH` GCS scratch bucket to use, overrides default set in Daisy.
+ `-oauth=OAUTH_PATH` Path to oauth json file, overrides what is set in workflow.
+ `-compute_endpoint_override=ENDPOINT` Compute API endpoint to override default.
+ `-disable_gcs_logging` Do not stream logs to GCS
+ `-disable_cloud_logging` Do not stream logs to Cloud Logging
+ `-disable_stdout_logging` Do not display individual workflow logs on stdout
+ `-kms-key=KMS_KEY_ID` ID of the key or fully qualified identifier for the key. This flag
  must be specified if any of the other arguments below are specified.
+ `-kms-keyring=KMS_KEYRING` The KMS keyring of the key.
+ `-kms-location=KMS_LOCATION` The Cloud location for the key.
+ `-kms-project=KMS_PROJECT` The Cloud project for the key
+ `-no_external_ip` Set if VPC does not allow external IPs
+ `-labels=[KEY=VALUE,...]` labels: List of label KEY=VALUE pairs to add. Keys must start with a
  lowercase character and contain only hyphens (-), underscores (_), lowercase characters, and 
  numbers. Values must contain only hyphens (-), underscores (_), lowercase characters, and numbers.
+ `-storage_location` Location for the imported image which can be any GCS location. If the location
  parameter is not included, images are created in the multi-region associated with the source disk,
  image, snapshot or GCS bucket.  

### Usage

```
gce_onestep_image_import -image_name=IMAGE_NAME -client_id=CLIENT_ID -os=OS
        -aws_access_key_id=AWS_ACCESS_KEY_ID -aws_secret_access_key=AWS_SECRET_ACCESS_KEY
        -aws_session_token=AWS_SESSION_TOKEN -aws_region=AWS_REGION
        (-aws_source_ami_file_path=AWS_SOURCE_AMI_FILE_PATH |
         -aws_ami_id=AWS_AMI_ID -aws_ami_export_location=AWS_AMI_EXPORT_LOCATION)
         [-no-guest-environment] [-family=FAMILY] [-description=DESCRIPTION] [-network=NETWORK]
        [-subnet=SUBNET] [-zone=ZONE] [-timeout=TIMEOUT] [-project=PROJECT]
        [-scratch_bucket_gcs_path=PATH] [-oauth=OAUTH_PATH] 
        [-compute_endpoint_override=ENDPOINT] [-disable_gcs_logging]
        [-disable_cloud_logging] [-disable_stdout_logging]
        [-kms-key=KMS_KEY -kms-keyring=KMS_KEYRING -kms-location=KMS_LOCATION
        -kms-project=KMS_PROJECT] [-labels=KEY=VALUE,...]
```
