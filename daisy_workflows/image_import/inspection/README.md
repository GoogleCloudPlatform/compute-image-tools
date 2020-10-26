`boot-inspect` is a Python library that finds boot-related properties of a
disk. It focuses on systems that are runnable on Google Compute Engine.

This package is under active development, and the APIs are not yet stabilized.

## Usage from Daisy:

Execute the workflow `boot-inspect.wf.json`. `pd_uri` is required, and is
the URI of the disk to inspect. Results are written to the serial
console in Daisy's key-value format.

Example:

```shell script
pd="projects/your-project/zones/us-central1-b/disks/test-the-image-opensuse-15-1-pl9hv"
daisy -project your-project \
      -zone=us-central1-b \
      -variables="pd_uri=$pd" \
      boot-inspect.wf.json

# <..running..>

"Status: <serial-output key:'architecture' value:'x64'>"
"Status: <serial-output key:'distro' value:'opensuse'>"
"Status: <serial-output key:'major' value:'15'>"
"Status: <serial-output key:'minor' value:'1'>"
```

## Run locally

The Python package installs a binary `boot-inspect` that can inspect
disk files (eg: vmdk) and mounted block devices (eg: /dev/sdb). Execute
`boot-inspect --help` for all options.

System-derived dependencies:
 - python3-guestfs
 - python3.5+

```shell script
pip3 install .
boot-inspect /images/ubuntu-16.04-server-cloudimg-amd64-disk1.vmdk 
{
    "device": "/images/ubuntu-16.04-server-cloudimg-amd64-disk1.vmdk",
    "os": {
        "distro": "ubuntu",
        "version": {
            "major": "16",
            "minor": "04"
        }
    },
    "architecture": "x64"
}
```

## Running unit tests

The safe and quick option is to run the prow job's container locally:

```shell script
cd $COMPUTE_IMAGE_TOOLS
rm -rf /tmp/artifacts && mkdir /tmp/artifacts
docker pull gcr.io/gcp-guest/pytest
docker run --volume $(pwd):/project:ro \
  --workdir /project \
  --volume /tmp/artifacts:/artifacts \
  --env ARTIFACTS=/artifacts \
  gcr.io/gcp-guest/pytest daisy_workflows/image_import/inspection
```

You'll have the same results as running on prow, and you'll
get test coverage across multiple Python interpreters. See
`/tmp/artifacts` for results.

For quick testing, you can execute the
[pytest](https://docs.pytest.org/en/stable/getting-started.html)
command from this directory. Be careful, though. Results may
differ from prow's depending on your local environment.
When in doubt, use the docker command above.
