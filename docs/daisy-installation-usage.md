# Installation
## Prebuilt binaries
Prebuilt Daisy binaries are available for Windows, macOS, and Linux distros.

Built from the latest GitHub release (all 64bit):

+ [Windows](https://storage.googleapis.com/compute-image-tools/release/windows/daisy.exe)
+ [macOS](https://storage.googleapis.com/compute-image-tools/release/darwin/daisy)
+ [Linux](https://storage.googleapis.com/compute-image-tools/release/linux/daisy)

Built from the latest commit to the master branch (all 64bit):

+ [Windows](https://storage.googleapis.com/compute-image-tools/latest/windows/daisy.exe)
+ [macOS](https://storage.googleapis.com/compute-image-tools/latest/darwin/daisy)
+ [Linux](https://storage.googleapis.com/compute-image-tools/latest/linux/daisy)

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
go get github.com/GoogleCloudPlatform/compute-image-tools/daisy
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

# Logging

Daisy will send logs to [Cloud Logging](https://cloud.google.com/logging/) if
available. If the API is disabled or the account Daisy is running is under does
not have permission to write log entries (the `logging.logEntries.create`
permission), Daisy will still send the logs to GCS and stdout by default.

- To disable sending logs to GCS, call Daisy with the flag `-disable_gcs_logging`
- To disable sending logs to Cloud Logging,  call Daisy with the flag `-disable_cloud_logging`
- To disable sending logs to stdout, call Daisy with the flag `-disable_stdout_logging`

# What Next?

For information on how to write Daisy workflow files, see the [workflow config
file specification](daisy-workflow-config-spec.md).
