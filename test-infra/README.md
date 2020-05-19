# GoogleCloudPlatform/compute-image-tools testing infrastructure

## Prow and Gubenator

We use [Prow](https://github.com/kubernetes/test-infra/tree/master/prow)
to run periodic, postsubmit, and presubmit (PRs) tests on packages in this.
repo. Our Prow GKE cluster runs in the GCP project, `compute-image-tools-test`.

We publish test results to
[Gubenator](https://github.com/kubernetes/test-infra/tree/master/gubernator).
Specifically, we publish test data/logs
to the publicly readable GCS bucket, `compute-image-tools-test`.

Links:
* Prow dashboard http://35.244.198.104/
* Testgrid dashboard for periodic Daisy e2e tests
https://k8s-testgrid.appspot.com/google-gce-compute-image-tools#ci-daisy-e2e
* Gubenator dashboard https://k8s-gubernator.appspot.com/

## Layout of test-infra/

| Path | Description |
| --- | --- |
| `prow/` | Configuration for the Prow cluster. |
| `prow/config.yaml` | Configuration describing Prow events and the associated Prow job containers. |
| `prow/plugins.yaml` | Configuration for Prow plugins. |
| `prowjobs/` | Prow job containers. |
| `prowjobs/daisy-e2e/` | Runs workflows in `daisy_workflows` against the latest (master:HEAD) Daisy binary. |
| `prowjobs/gce-image-import-export-tests/` | Runs image importer/exporter tests against the latest (master:HEAD) Image Importer/Exporter binary. |
| `prowjobs/gce-ovf-import-tests/` | Runs OVF importer tests against the latest (master:HEAD) OVF Importer binary. |
| `prowjobs/gocheck/` | Runs `go fmt`, `golint`, `go vet` against Go code in the repo. |
| `prowjobs/osconfig-tests/` | Runs OS Config tests. |
| `prowjobs/flake8/` | Runs `flake8` against python code in the repo. |
| `prowjobs/unittests/` | Runs all scripts within the repo with the filename `unittests.sh`. Each script is run within its own directory. Publishes code coverage results to Codecov. |
| `prowjobs/wrapper/` | Imported by other Prow jobs. Contains a wrapper binary that manages test log/artifact uploads. |

## Prow job wrapper binary

The container in `prowjobs/wrapper/` is not a Prow job itself. It contains a
binary which is imported by other other Prow job containers at container build
time.

The wrapper binary is wraps the container entry point. It handles logging to
the GCS bucket we use for Gubenator. The wrapper logs start, finish, and build
logs. It also uploads all artifacts found in the artifacts directory.
The artifacts directory is set with the environment variable `${ARTIFACTS}`.

## Prow job: test-runner

test-runner runs the latest daisy_test_runner binary against the test
template.

## Prow job: daisy-e2e

daisy-e2e invokes our latest Daisy binary against the
`compute-image-tools-test` GCP project. It runs the workflows matching
`daisy_worklows/e2e_tests/*.wf.json`.
Each matching workflow is run as a test case.

## Prow job: gce-image-import-export-tests

gce-image-import-export-tests invokes our latest Image Importer/Exporter
 binary against the `compute-image-tools-test` GCP project. It runs the
 tests defined in `cli_tools_e2e_test/gce_image_import_export`.

## Prow job: gce-ovf-import-tests

gce-ovf-import-tests invokes our latest OVF Importer binary against the
`compute-image-tools-test` GCP project. It runs the tests defined in
`cli_tools_e2e_test/gce_ovf_import`.

### Periodic runs and testgrid

This job is run periodically and results are uploaded to Gubenator and testgrid.
Testgrid is managed by the Kubernetes. Configuration changes require a pull
request. See
https://github.com/kubernetes/test-infra/pull/5044
for an example.

## Prow job: gocheck

Runs `go fmt`, `go vet`, and `golint`, checking for proper Go style and
formatting of ALL Go code within the repo.

## Prow job: flake8

Runs `flake8` checking for proper python style of ALL python code within the 
repo.

## Prow job: unittests

The unittests Prow job runs all scripts with the name `unittests.sh` in the
repo. It is up to each script to run the unit tests for the code its testing.
The script returns a nonzero status to indicate test failures.

### Test artifacts

Artifacts, such as unit test reports, produced by a `unittests.sh` script
should be published in a directory, `artifacts`, in the same directory as the
script. After a script terminates, artifacts are moved from its own `artifacts`
directory to a subdirectory of `/artifacts` in preparation to be uploaded by
the wrapper.

### Coverage reports

Two environment variables, `${GOCOVPATH}` and `${PYCOVPATH}`, are available to
the `unittests.sh` scripts. These are paths to Go and Python code coverage
reports which will be uploaded to codecov. Code coverage output must be
APPENDED to these filepaths.
