# Daisy E2E Tests

This directory contains end to end tests for Daisy. These tests are run
periodically against HEAD.

## Test Environment Details

* Tests run in the GCP project `compute-image-tools-test` and have permissions:
  * GCE read/write
  * GCS read on the `compute-image-tools-test-resources` bucket.
  * GCS read/write on the `compute-image-tools-test-sandbox` bucket.
