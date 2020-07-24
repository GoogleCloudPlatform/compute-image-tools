`inspection` is a Python library that finds boot-related properties of a disk.
It focuses on systems that are runnable on GCE.

## Usage from Daisy:

Execute the workflow `inspect-disk.wf.json`. There is a single variable, `pd_uri`,
which is the disk to inspect. Results are written to the serial
console in Daisy's key-value format.

Example:

```shell script
pd="projects/edens-test/zones/us-central1-b/disks/test-the-image-opensuse-15-1-pl9hv"
daisy -project edens-test \
      -zone=us-central1-b \
      -variables="pd_uri=$pd" \
      inspect-disk.wf.json

# <..running..>

"Status: <serial-output key:'architecture' value:'x64'>"
"Status: <serial-output key:'distro' value:'opensuse'>"
"Status: <serial-output key:'major' value:'15'>"
"Status: <serial-output key:'minor' value:'1'>"
```
