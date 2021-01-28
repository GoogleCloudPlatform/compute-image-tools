# Daisy E2E Tests

This directory contains end to end tests for Daisy.
To run against your local checkout (requires access to resources in
compute-image-tools-test):

```bash
go run daisy/daisy_test_runner/main.go -projects=<my project> -zone=us-central1-c daisy_integration_tests/daisy_e2e.test.gotmpl
```

Prow runs these tests periodically against HEAD.

## Test Environment Details

* Periodic Tests run in the `compute-image-test-pool-xxx` projects and have permissions:
  * GCE read/write
  * GCS read on the `compute-image-tools-test-resources` bucket.
  * GCS read/write on the `compute-image-tools-test-sandbox` bucket.
