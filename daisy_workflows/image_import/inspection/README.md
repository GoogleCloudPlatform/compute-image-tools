`inspection` is a Python library that finds boot-related properties of a disk.
It focuses on systems that are runnable on Google Compute Engine.

## Usage from Daisy:

Execute the workflow `inspect-disk.wf.json`. There is a single variable, `pd_uri`,
which is the disk to inspect. Results are written to the serial
console in Daisy's key-value format.

Example:

```shell script
pd="projects/your-project/zones/us-central1-b/disks/test-the-image-opensuse-15-1-pl9hv"
daisy -project your-project \
      -zone=us-central1-b \
      -variables="pd_uri=$pd" \
      inspect-disk.wf.json

# <..running..>

"Status: <serial-output key:'architecture' value:'x64'>"
"Status: <serial-output key:'distro' value:'opensuse'>"
"Status: <serial-output key:'major' value:'15'>"
"Status: <serial-output key:'minor' value:'1'>"
```

## Running tests
From the ./inspection directory:
```shell script
python3 -m tests.linux_tests
python3 -m tests.model_tests
```