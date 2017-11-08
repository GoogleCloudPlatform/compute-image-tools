# What is Daisy?
Daisy is a solution for running complex, multi-step workflows on GCE.

[![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/compute-image-tools/daisy?status.svg)](https://godoc.org/github.com/GoogleCloudPlatform/compute-image-tools/daisy)

The current Daisy stepset includes support for creating/deleting GCE resources,
waiting for signals from GCE VMs, streaming GCE VM logs, uploading local files
to GCE and GCE VMs, and more.

For example, Daisy is used to create Google Official Guest OS images. The
workflow:
1. Creates a Debian 8 disk and another empty disk.
2. Creates and boots a VM with the two disks.
3. Runs and waits for a script on the VM.
4. Creates an image from the previously empty disk.
5. Automatically cleans up the VM and disks.

Other use-case examples:
* Workflows for importing external physical or virtual disks to GCE.
* GCE environment deployment.
* Ad hoc GCE testing environment deployment and test running.

## Table of contents
  * [Setup](#setup)
    * [Prebuilt binaries](#prebuilt-binaries)
    * [Daisy container](#daisy-container)
    * [Build from source](#build-from-source)
  * [Running Daisy](#running-daisy)
  * [Testing](#testing)
  * [Documentation](#documentation)

## Setup
### Prebuilt binaries
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

### Daisy container
Daisy containers are available at gcr.io/compute-image-tools/daisy. All the
workflows in `compute-image-tools/daisy_workflows` are put in the `workflows`
directory at the root of the container.
+ Built from the latest GitHub release: gcr.io/compute-image-tools/daisy:release
+ Built from the latest commit to the master branch: gcr.io/compute-image-tools/daisy:latest

Daisy containers built with the beta Compute api
+ Built from the latest GitHub release: gcr.io/compute-image-tools/daisy_beta:release
+ Built from the latest commit to the master branch: gcr.io/compute-image-tools/daisy_beta:latest

### Build from source
Daisy can be easily built from source with the [Golang SDK](https://golang.org)
```shell
go get github.com/GoogleCloudPlatform/compute-image-tools/daisy/daisy
```
This will place the Daisy binary in $GOPATH/bin.

## Running Daisy
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

For information about writing Daisy config files, see the [workflow
configuration file
specification](https://googlecloudplatform.github.io/compute-image-tools/daisy-workflow-config-spec.html).

# Testing
Infrastructure has been set up to perform presubmit testing on PRs and
periodic continuous integration tests against HEAD.

Presubmit checks unit tests, `golint`, `go fmt`, and `go vet` against PRs
with changes to Daisy source code. Unit test coverage is reported to
codecov.io, which posts coverage reports on the PR. Presubmit tests are
gated by repo owners. Repo owners have the following commands available on
a PR:
* `/go test`: runs unit tests and reports coverage.
* `/golint`: runs `golint`.
* `/go fmt`: runs `go fmt`.
* `/go vet`: runs `go vet`.
* `/ok-to-test`: gives Prow the go-ahead to run the entire suite automatically.
* `/retest`: reruns failed tests, only available after Prow reports failures.

Periodic tests run every 6 hours. Currently, periodic tests include the e2e
tests here: [../daisy_workflows/e2e_tests](../daisy_workflows/e2e_tests). You
can see the test results for the e2e tests in testgrid: [https://k8s-testgrid.appspot.com/google-gce-compute-image-tools#daisy-e2e].

# Documentation

Daisy documentation can be found
[here](https://googlecloudplatform.github.io/compute-image-tools/daisy.html).

