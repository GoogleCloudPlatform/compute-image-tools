# Image Import into Google Compute Engine (GCE)

# Introduction
We provide tooling and instructions for migrating your existing virtual
machine images to GCE.

Tools:
* [Daisy](daisy.md)
* [Import Precheck](../import_precheck/README.md)

## Image Import Process
The image import process, also known as translation, imports virtual disks to
GCE disks. Once a disk is imported, additional provisioning and configuration
is usually needed to make the disk compatible with GCE. After importing,
provisioning, and configuring, a GCE image is created from the GCE disk and the
GCE disk is deleted.

The image import process is performed by our workflow manager tool,
[Daisy](daisy.md). We provide a workflow for the import process.

### Instructions
Please refer to the [userguide](daisy-import-userguide.md).

### Incompatibilities
Not every VM image will be importable to GCE. Some VMs will have issues after
import. Below is a list of known compatibility issues and requirements:

| Name | Severity | Description|
|-|-|-|
| OS Version | Required | We support the following OS versions: Windows Server 2008 R2, 2012 R2, or 2016; RHEL/CentOS/OEL 6 or 7; Debian 8 or 9; Ubuntu 14.04 or 16.04 |
| OS Disk | Required | The disk containing the OS must be bootable. The disk must be MBR and Linux distributions must have GRUB installed.|
| Multiple Disks | Warning  | Image import cannot handle multiple disk scenarios. Additional disks must be imported and attached separately.                                                                                          |
| SSH (Linux) | Warning | Warn if SSH is not running on port 22. GCE provides SSH clients via the Cloud Console and the gcloud CLI. These clients connect on port 22 and will not work if you have a different SSH configuration. |
| Powershell (Windows) | Warning | Warn if Powershell Version < 3. Powershell versions older than 3.0 can cause issues with GCE startup and shutdown scripts. |

## Image Import Precheck
Image import has a long runtime, can fail due to incompatibilities, and can
cause unexpected behavior post-import. As such, you may find it useful to run
our [precheck tool](../import_precheck/README.md) to check for the known issues
listed above. See [Incompatibilities](#incompatibilities) above.

There are binaries available for Windows and Linux.
