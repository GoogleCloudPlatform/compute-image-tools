# Daisy E2E Tests
This directory contains end to end tests for Daisy. These tests are run 
periodically against HEAD.

* Files matching `*.wf.json` are individual test cases.
  * If you have a sub wf that you do not want to run as a test case, you need
    to use a different file extension, e.g. `thing.subwf.json`, or put it in a
    subdirectory, e.g. `subdir/thing.wf.json`.
* Test cases run in parallel except in cases of ordered test cases.
* Ordered test cases:
  * Share a prefix.
  * Use non-negative numeric suffixes (before the `.wf.json`). If there
    is a test case without a suffix, it runs first.
  * e.g. `foo.wf.json`, `foo0.wf.json`, `foo1.wf.json`, and `foo014.wf.json`:
   `foo.wf.json`  will run, then `foo0.wf.json`, then `foo1.wf.json`, then
   finally `foo014.wf.json`.

## Test Environment Details
* Tests run in the GCP project `compute-image-tools-test` and have permissions:
  * GCE read/write
  * GCS read on the `compute-image-tools-test-resources` bucket.
  * GCS read/write on the `compute-image-tools-test-sandbox` bucket.
* Test logs are written to the `compute-image-tools-test` GCS bucket for Gubenator and
  testgrid to pick up.
* Defaults are provided for workflow `Project` and `Zone` fields.
  * `Project` is `compute-image-tools-test`
  * `Zone` is variable.
* The following args are passed to the test workflows:
  * `test-id`: The ID of this test run. Useful for sharing resources between
    test cases (would probably need to be ordered test cases, since parallel)
    test cases would cause race conditions).
