# Importing Virtual Disks into Google Compute Engine (GCE)


# Introduction

This document provides instructions for using Daisy to import VMWare VMDKs and other virtual disks.

Some basic concepts to start with:

*   **Virtual Disk**: Virtual disk is a file that encapsulates the content of a virtualized disk in a virtualization environment. Virtual Disks are critical components of virtual machines for holding boot media and data. Virtualization platforms (e.g. VMWare, Hyper-v, KVM, etc.) each have their own format for virtual disks.
*   **Persistent Disk**: Compute Engine Persistent Disk is a Compute Engine resource that is equivalent to disk drives in physical computers and virtualdisks in a virtualization environment.
*   **GCE Image**: Image is an immutable representation of a Persistent Disk and is used for creating multiple disks from one single templatized version.

# Installation

**NOTE:** before attempting a virtual disk import, take a look at the [known compatibility issues](#compatibility-and-known-limitations) and our [compatibility precheck tool](#compatibility-precheck-tool) below.

It is recommended that you run Daisy in a GCE VM instance that is configured with required API scopes and workflow files pre-installed. Executing the following command will create a new VM instance and install required files.

```
gcloud compute instances create daisy-control --image-project debian-cloud --image-family debian-9 --scopes=compute-rw,storage-full --metadata startup-script-url=gs://compute-image-tools/daisy_import_alpha.sh
```

# Importing Virtual Disks using Daisy

### Step 1: Upload Virtual Disk to Google Cloud Storage

Upload your virtual disk files to Google Cloud Storage (GCS) by following instructions at https://cloud.google.com/storage/docs/object-basics

### Step 2: Connect to the VM instance that has Daisy installed

Connect to the VM that was created in the Installation section. You can connect to the VM by running the following command.

```
gcloud compute ssh daisy-control
```

### Step 3: Convert Virtual Disk to Compute Engine Image

Now inside the VM, run the workflow titled `import_image.wf.json` to convert the virtual disk file in GCS to a Compute Engine Image.

```
daisy -var:source_disk_file=YOUR_VIRTUAL_DISK_FILE -var:image_name=YOUR-IMAGE-NAME /daisy/image_import/import_image.wf.json
```
Where, `YOUR_VIRTUAL_DISK_FILE` is the virtual disk file that you uploaded to GCS in the previous step. You must specify the full GCS path to the file.

`YOUR_IMAGE_NAME` is the name of your destination image.

Following is an example that converts `my_server.vmdk` present in gs://my-awesome-bucket

```
daisy -var:source_disk_file=gs://my-awesome-bucket/my_Server1.vmdk -var:image_name=my-server-import /daisy/image_import/import_image.wf.json

[Daisy] Running workflow "import-image"
[import-image]: 2017/06/29 21:51:12 Logs will be streamed to gs://my-awesome-bucket/daisy-import-image-20170629-21:51:12-sdgxl/logs/daisy.log
[import-image]: 2017/06/29 21:51:12 Validating workflow
[import-image]: 2017/06/29 21:51:12 Validating step "setup-disks"
[import-image]: 2017/06/29 21:51:13 Validating step "import-virtual-disk"
[import-image]: 2017/06/29 21:51:13 Validating step "wait-for-signal"
[import-image]: 2017/06/29 21:51:13 Validating step "create-image"
[import-image]: 2017/06/29 21:51:14 Validation Complete
[import-image]: 2017/06/29 21:51:14 Uploading sources
[import-image]: 2017/06/29 21:51:15 Running workflow
[import-image]: 2017/06/29 21:51:15 Running step "setup-disks" (CreateDisks)
[import-image]: 2017/06/29 21:51:16 CreateDisks: creating disk "disk-importer-import-image-sdgxl".
[import-image]: 2017/06/29 21:51:16 CreateDisks: creating disk "disk-import-import-image-sdgxl".
[import-image]: 2017/06/29 21:51:19 Step "setup-disks" (CreateDisks) successfully finished.
[import-image]: 2017/06/29 21:51:20 Running step "import-virtual-disk" (CreateInstances)
[import-image]: 2017/06/29 21:51:20 CreateInstances: creating instance "inst-importer-import-image-sdgxl".
[import-image]: 2017/06/29 21:51:33 Step "import-virtual-disk" (CreateInstances) successfully finished.
[import-image]: 2017/06/29 21:51:33 Running step "wait-for-signal" (WaitForInstancesSignal)
[import-image]: 2017/06/29 21:51:33 CreateInstances: streaming instance "inst-importer-import-image-sdgxl" serial port 1 output to gs://my-awesome-bucket/daisy-import-image-20170629-21:51:12-sdgxl/logs/inst-importer-import-image-sdgxl-serial-port1.log.
[import-image]: 2017/06/29 21:51:33 WaitForInstancesSignal: watching serial port 1, SuccessMatch: "ImportSuccess:", FailureMatch: "ImportFailed:".
[import-image]: 2017/06/29 22:16:08 WaitForInstancesSignal: SuccessMatch found for instance "inst-importer-import-image-90hqq"
[import-image]: 2017/06/29 22:16:08 Step "wait-for-signal" (WaitForInstancesSignal) successfully finished.
[import-image]: 2017/06/29 22:16:08 Running step "create-image" (CreateImages)
[import-image]: 2017/06/29 22:16:09 CreateImages: creating image "my_server_import".
[import-image]: 2017/06/29 22:16:58 Step "create-image" (CreateImages) successfully finished.
[import-image]: 2017/06/29 22:16:59 Workflow "import-image" cleaning up (this may take up to 2 minutes.

[Daisy] Workflow "import-image" finished
[Daisy] All workflows completed successfully.
```

### Step 4: Create a bootable disk using the imported image

After you have created a Compute Engine Image from your virtual disk, next step is to make the disk bootable. While Compute Engine can boot most disks as-is, running the following workflows will ensure the disks have the right drivers and integration software to ensure that you are able to start an instance using those disks and connect to it using SSH (or RDP in case of Windows).

```
daisy -var:source_image=projects/<YOUR-PROJECT-NAME>/global/images/YOUR-IMPORTED-IMAGE \\
-var:translate_workflow=OS_SPECIFIC_WORKFLOW -var:image_name=YOUR-TRANSLATED-IMAGE-NAME \\
/daisy/image_import/import_from_image.wf.json
```

Where, `YOUR_IMPORTED_IMAGE` is the GCE image that was created in step 2. The `source_image` field must be specified using the partial URL format for resources: `projects/<project-name>/global/images/<image-name>`

`YOUR-TRANSLATED-IMAGE-NAME` is the name of your new translated image.

`OS_SPECIFIC_WORKFLOW` is the name of the conversion workflow to run. The following sample workflows are provided. You can also create your own custom workflows.

### Workflows to use for open source or Google provided licensing

<table>
  <tr>
    <td>Source Operating System
    </td>
    <td>Workflow File
    </td>
  </tr>
  <tr>
    <td>Debian 8
    </td>
    <td>/daisy/debian/translate_debian_8.wf.json
    </td>
  </tr>
  <tr>
    <td>Debian 9
    </td>
    <td>/daisy/debian/translate_debian_9.wf.json
    </td>
  </tr>
  <tr>
    <td>CentOS 6
    </td>
    <td>/daisy/enterprise_linux//translate_centos_6.wf.json
    </td>
  </tr>
  <tr>
    <td>CentOS 7
    </td>
    <td>/daisy/enterprise_linux//translate_centos_7.wf.json
    </td>
  </tr>
  <tr>
    <td>RHEL 6
    </td>
    <td>/daisy/enterprise_linux/translate_rhel_6_licensed.wf.json
    </td>
  </tr>
  <tr>
    <td>RHEL 7
    </td>
    <td>/daisy/enterprise_linux/translate_rhel_7_licensed.wf.json
    </td>
  </tr>
  <tr>
    <td>Ubuntu 14.04
    </td>
    <td>/daisy/ubuntu/translate_ubuntu_1404.wf.json
    </td>
  </tr>
  <tr>
    <td>Ubuntu 16.04
    </td>
    <td>/daisy/ubuntu/translate_ubuntu_1604.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2008 R2
    </td>
    <td>/daisy/windows/translate_windows_2008_r2.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2012 R2
    </td>
    <td>/daisy/windows/translate_windows_2012_r2.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2016
    </td>
    <td>/daisy/windows/translate_windows_2016.wf.json
    </td>
  </tr>
</table>

### Workflows to use if customer supplies licensing (BYOL)

<table>
  <tr>
    <td>Source Operating System
    </td>
    <td>Workflow File
    </td>
  </tr>
  <tr>
    <td>RHEL 6 BYOL
    </td>
    <td>/daisy/enterprise_linux/translate_rhel_6_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>RHEL 7 BYOL
    </td>
    <td>/daisy/enterprise_linux/translate_rhel_7_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2008 R2 BYOL
    </td>
    <td>/daisy/windows/translate_windows_2008_r2_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2012 BYOL
    </td>
    <td>/daisy/windows/translate_windows_2012_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2012 R2 BYOL
    </td>
    <td>/daisy/windows/translate_windows_2012_r2_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2016 BYOL
    </td>
    <td>/daisy/windows/translate_windows_2016_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows Server 2019 BYOL
    </td>
    <td>/daisy/windows/translate_windows_2019_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows 7 x64 BYOL
    </td>
    <td>/daisy/windows/translate_windows_7_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows 8.1 x64 BYOL
    </td>
    <td>/daisy/windows/translate_windows_81_byol.wf.json
    </td>
  </tr>
  <tr>
    <td>Windows 10 x64 BYOL
    </td>
    <td>/daisy/windows/translate_windows_10_byol.wf.json
    </td>
  </tr>
</table>

For example,

```
$ daisy -var:source_image=projectsmy-awesome-projectglobal/images/my-server-import -var:translate_workflow=/daisy/image_import/ubuntu/translate_ubuntu_1604.wf.json -var:image_name=my-new-ubuntu-1604-image /daisy/import_from_image.wf.json
```

# Compatibility and Known Limitations

*   Networking: Import workflow sets the interface to DHCP. If that fails, or if there are other interfaces set with firewalls, special routing, VPN's, or other non-standard configurations, networking may fail and while the resulting instance may boot, you may not be able to access it.

Not every VM image will be importable to GCE. Some VMs will have issues after
import. Below is a list of known compatibility requirements and issues:

### Windows

| Name | Severity | Description |
|---|---|---|
| OS Version | Required | We support the following OS versions: Windows Server 2008 R2, 2012 R2, 2016, or 2019 and Windows 7, 8, or 10. |
| OS Disk | Required | The disk containing the OS must be bootable and must be MBR. |
| Multiple Disks | Warning  | Image import cannot directly handle multiple disk scenarios. Additional disks must be imported and attached separately. |
| Powershell (Windows) | Warning | Warn if Powershell Version < 3. Powershell versions older than 3.0 can cause issues with GCE startup and shutdown scripts. |

### Linux

| Name | Severity | Description |
|---|---|---|
| OS Version | Required | We support the following OS versions: RHEL/CentOS/OEL 6 or 7; Debian 9; Ubuntu 14.04 or 16.04. |
| OS Disk | Required | The disk containing the OS must be bootable. The disk must be MBR and have GRUB installed. |
| Multiple Disks | Warning  | Image import cannot directly handle multiple disk scenarios. Additional disks must be imported and attached separately. |
| SSH | Warning | Warn if SSH is not running on port 22. GCE provides SSH clients via the Cloud Console and the gcloud CLI. These clients connect on port 22 and will not work if you have a different SSH configuration. |

### Compatibility Precheck Tool
Image import has a long runtime, can fail due to incompatibilities, and can
cause unexpected behavior post-import. As such, you may find it useful to run
our [precheck tool](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/cli_tools/import_precheck/)
to check for the known issues listed above.

# Advanced Topics

### Installing Daisy on your computer

[Pre-built binaries](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#prebuilt-binaries) are available.

Or, you can run one of the following commands to install Daisy and workflow files on a Debian or Ubuntu based system.

```
sudo curl https://storage.googleapis.com/compute-image-tools/daisy_import_alpha.sh | sudo bash
```

Or

```
apt-get update
apt-get -y install git

mkdir /daisy
git clone https://github.com/GoogleCloudPlatform/compute-image-tools.git /tmp/compute-image-tools

cp -R /tmp/compute-image-tools/daisy_workflows/* /daisy/
chmod -R u+rwX,g+rX,o+rX /daisy

wget https://storage.googleapis.com/compute-image-tools/release/linux/daisy -O /usr/bin/daisy

chmod 755 /usr/bin/daisy

echo "Daisy is installed and import workflows available in /daisy."
```

### Configuring Permissions for Daisy

In order to complete most workflows, specifically import workflows, Daisy needs to be to access Compute and Storage APIs. There are three options to ensure Daisy has the right level of access.


*   **Option A**:  Create Compute Engine instances with appropriate API scope and run Daisy in the instance.  Following API scopes are required.
        *   https://www.googleapis.com/auth/devstorage.read_write
        *   https://www.googleapis.com/auth/compute

`gcloud compute --project "YOUR-PROJECT" instances create "test-import-vm" --scopes=compute-rw,storage-full --image-project debian-cloud --image-family debian-9`

*   **Option B**:  In the computer where you have Daisy instance, run 'gcloud auth application-default login' and login as a user that has privileges to create and remove virtual machines in Compute Engine.

`gcloud auth application-default login`

### Advanced Daisy Parameters

Refer to https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#running-daisy for the full list of Daisy parameters.

### Custom Daisy Workflows

Daisy documentation (https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#workflow-config-overview) provides an overview of how to create and modify Daisy workflows. You can also refer to sample image import workflows at https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy_workflows/image_import
