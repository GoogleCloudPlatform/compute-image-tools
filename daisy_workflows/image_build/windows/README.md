# Microsoft Windows Build

To build Microsoft Windows images for use in Google Compute Engine, the following resources are needed:

* Microsoft OS Volume license media in ISO format
* [PowerShell 7.0 or greater MSI installer](https://github.com/PowerShell/PowerShell#get-powershell)
* [Microsoft .NET Framework 4.8 offline installer for Windows](https://support.microsoft.com/en-us/help/4503548/microsoft-net-framework-4-8-offline-installer-for-windows)

The following resources are optional:

* Windows Updates to slipstream install in the installation process
* A Windows Server Update Services server.

## Slipstreaming Windows Update (Optional)

To reduce the time needed to build a Microsoft Windows image we can apply .msu
based updated to the OS. This process is more beneficial for older operating systems.
To do this the build process will install all of the .msu files from the specified updates directory or GCS location in numerical and alphanumeric order.

For Windows Server 2016 and newer operating systems, it is beneficial to install
at least the most recent Servicing Stack Update (SSU), Cumulative Update (CU),
and .NET framework updates.

It is best to install updates in the following order:
1. Microsoft .NET Framework Installation
1. Cumulative Update for .NET Framework
1. The OS's Servicing Stack Update
1. The OS's Cumulative Update

## How the workflow builds the image

1. All of the provided files are uploaded to a GSC bucket that will be used with the workflow.
1. A new installation disk of the specified size is created and attached to a Windows Server 2019 "bootstrap instance".
1. The WIM image from the media, the provided Windows updates, GCE drivers, and installation files are applied to the installation disk. This is all done in the [bootstrap_install.ps1](https://github.com/GoogleCloudPlatform/compute-image-tools/blob/master/daisy_workflows/image_build/windows/bootstrap_install.ps1).
1. The "bootstrap instance" is stopped and a new "install instance" is created that boots the installation disk.
1. The "install instance" is started and will run the [SetupComplete.cmd](https://github.com/GoogleCloudPlatform/compute-image-tools/blob/master/daisy_workflows/image_build/windows/components/SetupComplete.cmd) and then the [post_install.ps1](https://github.com/GoogleCloudPlatform/compute-image-tools/blob/master/daisy_workflows/image_build/windows/post_install.ps1).
1. The "install instance" is stopped and a GCE Image is created from the installation disk with the appropriate [on-demand licenses for Windows Server](https://cloud.google.com/compute/docs/instances/windows/ms-licensing) and features.

## Workflow Variables

The workflow files provide default values for many of the variables. When calling a workflow,
any required variable will need to be provided when calling daisy using the -var: flag. The OS specific workflow files simplify the image creation process by populating the OS specific variables to windows-build.wf.json and also creates a GCE image with the appropriate features and license.


| Variable Name | Description |
| --- | --- |
| project | Project to allocate resources from during build [Project docs](https://cloud.google.com/resource-manager/docs/creating-managing-projects) |
| zone | Zone to use for GCE build instance [Zone docs](https://cloud.google.com/compute/docs/regions-zones/) |
| media | Absolute path to or GCS resource name of the ISO file |
| pwsh | Absolute path to or GCS resource name of the PowerShell 7+ installer |
| dotnet48 | Absolute path to or GCS resource name of the Microsoft .NET Framework 4.8 offline installer |
| updates | (Optional) Directory or GCS location containing updates to be included in install |
| product_key | (Optional) Windows product key to use. Volume license media by default include the Generic Volume License Key. |

### Selecting a workflow

We have provided multiple workflows per operating system to provide different configurations of each image.

The workflow files use the following naming convention.

OperatingSystem-OperatingSystemEdition-BootType-LicenseType.wf.json

* OperatingSystem: The operating system name and version
* OperatingSystemEdition: The edition of the operating system
* BootType: bios or uefi
  * bios: BIOS boot with an MBR formatted boot disk.
  * uefi: UEFI boot with an GPT formatted boot disk. Supports Shielded VM features.
* LicenseType: byol or payg
  * byol - [bring your own license](https://cloud.google.com/compute/docs/nodes/bringing-your-own-licenses)
  * payg - [on-demand Windows Server license](https://cloud.google.com/compute/docs/instances/windows/ms-licensing#on-demand)

Here are some example of what each filename means:
* windows-server-2019-dc-uefi-payg.wf.json
  * Windows Server 2019 Data Center using UEFI with an GPT formatted boot disk that is
    using an [on-demand Windows Server license](https://cloud.google.com/compute/docs/instances/windows/ms-licensing#on-demand)
* windows-server-2019-dc-uefi-byol.wf.json
  * Windows Server 2019 Data Center using UEFI with an GPT formatted boot disk that is
    using a [bring your own license](https://cloud.google.com/compute/docs/nodes/bringing-your-own-licenses)
* windows-10-20h2-ent-x86-bios-byol.wf.json
  * Windows 10 Enterprise 20h2 x86 using BIOS with an MBR formatted boot disk that is
    using a [bring your own license](https://cloud.google.com/compute/docs/nodes/bringing-your-own-licenses)
* windows-10-20h2-ent-x64-uefi-byol.wf.json
  * Windows 10 Enterprise 20h2 x64 using using UEFI with an GPT formatted boot disk that is
    using a [bring your own license](https://cloud.google.com/compute/docs/nodes/bringing-your-own-licenses)


### Starting a build workflow

Below are some example of how to call daisy using the provided workflows and required variables.

#### Build a Windows Server 2016 Data Center edition with UEFI Support using local files
* The windows media will be in the same directory and is named WindowServer2016_DataCenter.ISO.
* The PowerShell 7+ installer will be in the same directory and is named PowerShell-7.0.3-win-x64.msi.
* The Microsoft .NET Framework 4.8 offline installer will be in the same directory and is named ndp48-x86-x64-allos-enu.exe.
* The windows updates for Windows Server 2016 are located in a subfolder called Updates.

```shell
$ daisy -project my_project -zone us-west1-c \
-var:media="WindowServer2016.ISO" \
-var:updates=updates/ \
-var:pwsh=PowerShell-7.0.3-win-x64.msi \
-var:dotnet48=ndp48-x86-x64-allos-enu.exe \
windows-server-2016-dc-uefi.wf.json
```

#### Build a Windows Server 2019 Data Center edition with UEFI Support using files in a GCS bucket
* The windows media is stores in a GCS.
* The PowerShell 7+ installer is stores in a GCS.
* The Microsoft .NET Framework 4.8 offline installer is stores in a GCS.
* The windows updates for Windows Server 2019 are stores in a GCS.

```shell
$ daisy -project my_project -zone us-west1-c \
-var:media="gs://example-build-resources/2019/WindowServer2019.ISO" \
-var:updates="gs://example-build-resources/2019/updates" \
-var:pwsh="gs://example-build-resources/PowerShell-7.0.3-win-x64.msi" \
-var:dotnet48="gs://example-build-resources/ndp48-x86-x64-allos-enu.exe" \
windows-server-2019-dc-uefi.wf.json
```

### Creating a instance from the newly created image (Optional)

In this example we'll be using an image Server 2019 image created on 12/09/2020
and it is named windows-server-2016-dc-v12092020 and stored in the the my_project project.

```shell
$ gcloud compute instances create instance_name --async --machine-type n1-standard-8 \
--project my_project --zone us-west1-c --image windows-server-2019-dc-v12092020
```
[Cloud
SDK Reference for compute instances create](https://cloud.google.com/sdk/gcloud/reference/compute/instances/create)
