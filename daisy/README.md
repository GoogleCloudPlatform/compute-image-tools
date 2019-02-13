# What is Daisy?
Daisy is a solution for running multi-step workflows on GCE.

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

# [Documentation](https://googlecloudplatform.github.io/compute-image-tools/daisy.html)

Daisy documentation can be found
[here](https://googlecloudplatform.github.io/compute-image-tools/daisy.html).

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
tests here: [../test-infra/prow/config.yaml](../test-infra/prow/config.yaml). You
can see the test results for the e2e tests in testgrid: [https://k8s-testgrid.appspot.com/google-gce-compute-image-tools#ci-daisy-e2e].

