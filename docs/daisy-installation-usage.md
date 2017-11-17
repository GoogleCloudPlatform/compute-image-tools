# Installation
## Prebuilt binaries
Prebuilt Daisy binaries are available for Windows, macOS, and Linux distros.
Two versions are available, one built with the v1 (stable) Compute api, and the
other with the beta Compute API.

Built from the latest GitHub release (all 64bit):

+ [Windows](https://storage.googleapis.com/compute-image-tools/release/windows/daisy.exe)
+ [Windows beta](https://storage.googleapis.com/compute-image-tools/release/windows/daisy_beta.exe)
+ [macOS](https://storage.googleapis.com/compute-image-tools/release/darwin/daisy)
+ [macOS beta](https://storage.googleapis.com/compute-image-tools/release/darwin/daisy_beta)
+ [Linux](https://storage.googleapis.com/compute-image-tools/release/linux/daisy)
+ [Linux beta](https://storage.googleapis.com/compute-image-tools/release/linux/daisy_beta)

Built from the latest commit to the master branch (all 64bit):

+ [Windows](https://storage.googleapis.com/compute-image-tools/latest/windows/daisy.exe)
+ [Windows beta](https://storage.googleapis.com/compute-image-tools/latest/windows/daisy_beta.exe)
+ [macOS](https://storage.googleapis.com/compute-image-tools/latest/darwin/daisy)
+ [macOS beta](https://storage.googleapis.com/compute-image-tools/latest/darwin/daisy_beta)
+ [Linux](https://storage.googleapis.com/compute-image-tools/latest/linux/daisy)
+ [Linux beta](https://storage.googleapis.com/compute-image-tools/latest/linux/daisy_beta)

## Daisy container
Daisy containers are available at gcr.io/compute-image-tools/daisy. All the
workflows in `compute-image-tools/daisy_workflows` are put in the `workflows`
directory at the root of the container.

+ Built from the latest GitHub release: gcr.io/compute-image-tools/daisy:release
+ Built from the latest commit to the master branch: gcr.io/compute-image-tools/daisy:latest

Daisy containers built with the beta Compute api

+ Built from the latest GitHub release: gcr.io/compute-image-tools/daisy_beta:release
+ Built from the latest commit to the master branch: gcr.io/compute-image-tools/daisy_beta:latest

## Build from source
Daisy can be easily built from source with the [Golang SDK](https://golang.org)
```shell
go get github.com/GoogleCloudPlatform/compute-image-tools/daisy/daisy
```
This will place the Daisy binary in `$GOPATH/bin`.

# Usage
The basic use case for Daisy looks like:
```shell
daisy [path to workflow config file]
```

Workflow variables can be set using the  `-variables` flag or the
`-var:VARNAME` flag. The `-variables` flag takes a comma separated list
of `key=value` pairs. Both of these examples set the workflow variables
`foo=bar` and `baz=gaz`:
```shell
daisy -variables foo=bar,baz=gaz wf.json
```

```shell
daisy -var:foo bar -var:baz gaz wf.json
```

For additional information about Daisy flags, use `daisy -h`.

# What Next?

For information on how to write Daisy workflow files, see the [workflow config
file specification](daisy-workflow-config-spec.md).
