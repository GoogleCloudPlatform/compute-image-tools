### Windows ISO to Google Compute Image Build

The currently supported versions of Windows Server are listed below. Additional
versions will be added in the future.

| Windows Server | Servicing Stack Update (SSU) | Cumulative Update (CU) | Additional Recommended Update |
| ---| ---| --- | --- |
| 2016 |[KB45090](http://download.windowsupdate.com/c/msdownload/update/software/secu/2019/07/windows10.0-kb4509091-x64_1e1039047044a1f632562cd65df3ce6b77804018.msu) |[KB4507460](http://download.windowsupdate.com/c/msdownload/update/software/secu/2019/07/windows10.0-kb4507460-x64_6d645b0420de6481c8cbad52885f70f7369cb662.msu) |

`Table last updated 2019-07-16`

- Download Servicing Stack Update and Cumulative Update files
- Rename files so Service Stack Update installs first. **NOTE:** SSU is
  typically a higher KB number than the CU.
```shell
$ curl -O http://download.windowsupdate.com/c/msdownload/update/software/secu/2019/07/windows10.0-kb4507460-x64_6d645b0420de6481c8cbad52885f70f7369cb662.msu
$ curl -O http://download.windowsupdate.com/c/msdownload/update/software/secu/2019/07/windows10.0-kb4509091-x64_1e1039047044a1f632562cd65df3ce6b77804018.msu
$ mv windows10.0-kb4509091-x64_1e1039047044a1f632562cd65df3ce6b77804018.msu first-windows10.0-kb4509091-x64_1e1039047044a1f632562cd65df3ce6b77804018.msu
```
- Move these files into a subdirectory, optionally named `updates`

- Run build workflow
```shell
$ daisy -project my_project -zone us-west1-c \
-var:media="SW_DVD9_Win_Server_STD_CORE_2016_64Bit_English_-4_DC_STD_MLF_X21-70526.ISO" \
-var:updates=updates/ windows-server-2016-dc.wf.json
```
| Variable Name | Description |
| --- | --- |
| project | Project to allocate resources from during build [Project docs](https://cloud.google.com/resource-manager/docs/creating-managing-projects) |
| zone | Zone to use for GCE build instance [Zone docs](https://cloud.google.com/compute/docs/regions-zones/) |
| media | Absolute path to or GCS resource name of ISO file |
| updates | (Optional) Directory containing updates to be included in install |

- Create a test instance (Optional)
```shell
$ gcloud compute instances create instance_name --async --machine-type n1-standard-8 \
--project my_project --zone us-east1-d --image windows-server-2016-dc-v1
```
[Cloud
SDK Reference for compute instances create](https://cloud.google.com/sdk/gcloud/reference/compute/instances/create)
