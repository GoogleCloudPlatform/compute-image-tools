# Container images for compute-image-tools testing infrastructure

These directories contain source code for containers used by Prow. An example
process for updating the unittests container image is as follows:

1. Make changes.
1. In the unittests dir: `docker build --tag gcr.io/compute-image-tools-test/unittests:latest .`
1. Manually test the image before pushing:

* `docker run -e "REPO_OWNER=GoogleCloudPlatform" -e "REPO_NAME=compute-image-tools" -e "PULL_NUMBER=264" -v ~/codecovtoken:/etc/codecov/token gcr.io/compute-image-tools-test/unittests:latest`

1. Push the image to Google Container Registry:

* `gcloud docker -- push gcr.io/compute-image-tools-test/unittests:latest`
