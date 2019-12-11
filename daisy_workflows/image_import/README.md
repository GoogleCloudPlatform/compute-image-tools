# Disk and Image Imports

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

### import_image.wf.json

Imports a virtual disk file and converts it into a GCE image resource.

Variables:
* `source_disk_file`: A supported source virtual disk file, either local or in GCS.
* `image_name`: The name of the imported image, will default to "imported-image-${ID}".
* `importer_instance_disk_size`: The size of the importer instance disk, additional disk space
is unused for the import but a larger size increases PD write speed. See 
[Compute Engine storage documentation](https://cloud.google.com/compute/docs/disks)

Example Daisy invocation:
```shell
# Example importing a VMDK (using a credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:source_disk_file ./image.vmdk \
      import_image.wf.json
```

## Image Translation

Translation workflows attempt to add GCE packages, remove known problematic
packages, assure networking it setup to boot correctly, and configure the bootloader
(assumed to be grub in Linux distros) to output logs to the serial console.
These are generic assumptions and will not work in every case.

Variables:
* `source_image`: The source GCE image to translate.
* `install_gce_packages`: True by default, if set to false, will not attempt to install packages for GCE.
* `image_name`: The name of the translated image, will default to "$DISTRO-$VER-${ID}".
* *WINDOWS ONLY* `sysprep`: False by default, whether to run sysprep before capturing the image.".

### Workflows

* **debian/translate_debian_8.wf.json**: translates a Debian 8 Jessie based virtual disk.
* **debian/translate_debian_9.wf.json**: translates a Debian 9 Stretch based virtual disk.
* **enterprise_linux//translate_centos_6.wf.json**: translates a CentOS 6 based virtual disk.
* **enterprise_linux//translate_centos_7.wf.json**: translates a CentOS 7 based virtual disk.
* **enterprise_linux/translate_rhel_6_byol.wf.json**: translates a Red Hat Enterprise Linux 6 based virtual disk using your own Red Hat license.
* **enterprise_linux/translate_rhel_6_licensed.wf.json**: translates a Red Hat Enterprise Linux 6 based virtual disk and converts it to use a GCE based Red Hat cloud license. If you use the resulting image you will be charged for the license.
* **enterprise_linux/translate_rhel_7_byol.wf.json**: translates a Red Hat Enterprise Linux 7 based virtual disk using your own Red Hat license.
* **enterprise_linux/translate_rhel_7_licensed.wf.json**: translates a Red Hat Enterprise Linux 7 based virtual disk and converts it to use a GCE based Red Hat cloud license. If you use the resulting image you will be charged for the license.
* **suse/translate_suse.wf.json**: translates a openSUSE Leap based virtual disk.
* **freebsd/translate_freebsd.wf.json**: translates a FreeBSD based virtual disk.
* **ubuntu/translate_ubuntu_1404.wf.json**: translates an Ubuntu 14.04 Trusty based virtual disk.
* **ubuntu/translate_ubuntu_1604.wf.json**: translates an Ubuntu 16.04 Xenial based virtual disk.

If you are using a Google provided license for Windows, but want to import a
custom disk image, use the following import workflows:
* **windows/translate_windows_2008_r2.wf.json**: Windows Server 2008r2
* **windows/translate_windows_2012.wf.json**: Windows Server 2012
* **windows/translate_windows_2012_r2.wf.json**: Windows Server 2012r2
* **windows/translate_windows_2016.wf.json**: Windows Server 2016
* **windows/translate_windows_2019.wf.json**: Windows Server 2019

If you are bringing a license and disk image for Windows, use the following import workflows:
* **windows/translate_windows_2008_r2_byol.wf.json**: Windows Server 2008r2
* **windows/translate_windows_2012_byol.wf.json**: Windows Server 2012
* **windows/translate_windows_2012_r2_byol.wf.json**: Windows Server 2012r2
* **windows/translate_windows_2016_byol.wf.json**: Windows Server 2016
* **windows/translate_windows_2019_byol.wf.json**: Windows Server 2019
* **windows/translate_windows_7_byol.wf.json**: Windows 7 x64
* **windows/translate_windows_8-1_x64_byol.wf.json**: Windows 8.1 x64
* **windows/translate_windows_10_byol.wf.json**: Windows 10 x64

Example Daisy invocation:
```shell
# Example translating an Ubuntu 14.04 VMDK (using a credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:source_image projects/my-project/global/images/ubuntu-1404-xy23f \
      ubuntu/translate_ubuntu_1404.wf.json
```

Example of importing a FreeBSD disk from a bucket to a translated image
```shell
daisy -project my-project \
      -zone us-central1-b \
      -variables image_name=my-imported-freebsd,translate_workflow=freebsd/translate_freebsd.wf.json,source_disk_file=gs://bucket/my-freebsd-disk.qcow2
      import_and_translate.wf.json
```
