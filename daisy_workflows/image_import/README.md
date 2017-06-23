# Disk and Image Imports

**Proof of concept only! Highly experimental!**

We use [qemu-img](http://www.qemu.org/documentation) to convert formats to a GCE compatible disk. Valid input formats are:

* raw
* qcow2
* qcow
* vmdk
* vdi
* vhd
* vhdx
* qed
* vpc

### image_import.wf.json

Imports a virtual disk file and converts it into a GCE image resource.

Variables:
* `source_disk_file`: A supported source virtual disk file, either local or in GCS.

Example Daisy invocation:
```shell
# Example importing a VMDK (using a credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables source_disk_file=image.vmdk \
      image_import.wf.json
```

### disk_import.wf.json

Import a virtual disk file and converts it into a given GCE disk resource. This
workflow can be included in other workflows to perform further actions on the
imported disk.

Variables:
* `imported_disk`: A GCE disk resource to convert the imported disk to. This GCE disk must already exist.
* `source_disk_file`: A supported source virtual disk file, either local or in GCS.

In the following example, we are creating a disk with Daisy and passing the Daisy internal name of the disk resource into the workflow. This is because the VM acting up on the disk needs to know what the real name of the disk resource is in order to correctly resize it.

```json
"Steps": {
  "create-import-disk": {
    "CreateDisks": [
      {
        "Name": "ubuntu14-import",
        "SizeGb": "10",
        "Type": "pd-ssd"
      }
    ]
  },
  "import-virtual-disk": {
    "IncludeWorkflow": {
      "Path": "./disk_import.wf.json",
      "Vars": {
        "source_disk_file": "${source_disk_file}",
        "imported_disk": "ubuntu14-import-${NAME}-${ID}"
      }
    },
    "Timeout": "60m"
  }
}
```

## Imports w/Translation

Translation workflows attempt to add GCE packages, remove known problematic
packages, assure networking it setup to boot correctly, and configure the bootloader
(assumed to be grub in Linux distros) to output logs to the serial console.
These are generic assumptions and will not work in every case.

Variables:
* `source_disk_file`: A supported source virtual disk file, either local or in GCS.
* `install_gce_packages`: True by default, if set to false, will not attempt to
  install packages for GCE.

### Workflows

* **centos/import_centos_6.wf.json**: imports and translates a CentOS 6 based virtual disk.
* **centos/import_centos_7.wf.json**: imports and translates a CentOS 7 based virtual disk.
* **debian/import_debian_8.wf.json**: imports and translates a Debian 8 Jessie based virtual disk.
* **debian/import_debian_9.wf.json**: imports and translates a Debian 9 Stretch based virtual disk.
* **rhel/import_rhel_6_byol.wf.json**: imports and translates a Red Hat Enterprise Linux 6 based virtual disk using your own Red Hat license.
* **rhel/import_rhel_6_licensed.wf.json**: imports and translates a Red Hat Enterprise Linux 6 based virtual disk and converts it to use a GCE based Red Hat cloud license. If you use the resulting image you will be charged for the license.
* **rhel/import_rhel_7_byol.wf.json**: imports and translates a Red Hat Enterprise Linux 7 based virtual disk using your own Red Hat license.
* **rhel/import_rhel_7_licensed.wf.json**: imports and translates a Red Hat Enterprise Linux 7 based virtual disk and converts it to use a GCE based Red Hat cloud license. If you use the resulting image you will be charged for the license.
* **ubuntu/import_ubuntu_1404.wf.json**: imports and translates an Ubuntu 14.04 Trusty based virtual disk.
* **ubuntu/import_ubuntu_1604.wf.json**: imports and translates an Ubuntu 16.04 Xenial based virtual disk.

Example Daisy invocation:
```shell
# Example importing an Ubuntu 14.04 VMDK (using a credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables source_disk_file=gs://my-vmdk-bucket/ubuntu-1404.vmdk \
      ubuntu/import_ubuntu_1404.wf.json
```
