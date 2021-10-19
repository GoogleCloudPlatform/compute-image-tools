e2e tests: Invoke gcloud and the wrapper binaries using their public APIs.

## Overview

Each directory adjacent to this README is a test suite. At the time of writing, there are four test suites:

```
gce_image_import_export
gce_ovf_export
gce_ovf_import
gce_windows_upgrade
```

We build, deploy, and invoke the e2e tests using docker images, with one image per test suite. The docker image
contains:

1. Tests, as a compiled go binary
2. One or more wrappers, as a compiled go binary.
3. gcloud

The Dockerfiles that generate these images are in the root of this repository. At the time of writing, they are:

```
gce_image_import_export_tests.Dockerfile
gce_ovf_export_tests.Dockerfile
gce_ovf_import_tests.Dockerfile
gce_windows_upgrade_tests.Dockerfile
```

## Building

Build e2e Dockerfiles using `docker build -f <docker file name>`.

This example builds the tests associated with `gce_image_import_export_tests.Dockerfile`, and stores the image in a
tag `gce_image_import_export_tests`.

From the root of this repository, execute:

```shell
gcloud auth login
gcloud auth configure-docker
docker build  -f gce_image_import_export_tests.Dockerfile  . -t e2e
```

Notes:

1. The first two gcloud commands ensure that your local docker installation can read
   the `gcr.io/compute-image-tools-test/wrapper-with-gcloud:latest` image.

Use the same syntax to build other e2e Dockerfiles.

## Running

Run e2e docker images using `docker run <tag name>`.

This example runs a subset of the tests from the image built in the prior step.

```shell
gcloud auth application-default login
docker run --env GOOGLE_APPLICATION_CREDENTIALS= \
           --env CLOUDSDK_CONFIG=/root/.config/gcloud \
           -v $HOME/.config/gcloud:/root/.config/gcloud \
           gce_image_import_export_tests \
           -test_project_id compute-image-test-pool-001 \
           -test_zone us-central1-a \
           -test_suite_filter=^ImageImport$ \
           -test_case_filter=ubuntu
```

Notes:

1. The first gcloud command activates
   [application default credentials](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login).
2. The `docker run` command erases the `GOOGLE_APPLICATION_CREDENTIALS`, and sets `CLOUDSDK_CONFIG`. This ensures that
   gcloud will use the credentials from step [1], which are mounted into a volume using `-v`. Prow
   uses `GOOGLE_APPLICATION_CREDENTIALS` to inject a service account; using `CLOUDSDK_CONFIG` ensures that you don't
   have to download those keys to your machine.
3. `-test_project_id ` and `-test_zone` are required. Instead of using the test pool project, you can run the tests in
   your own test project. This example uses the test pool, since the test suites assumes resources from that project
   will be present.
4. `-test_suite_filter` and `-test_case_filter` are optional. These specify which tests to execute; if empty, all tests
   are executed. See [launcher.go](../../go/e2e_test_utils/launcher.go) for the filter's implementation.

Beyond these flags, some test suites have further customization via a `-variables` flag. Search for `-variables` in the
[prow configuration](../../test-infra/prow/config.yaml) to see what's currently used.
