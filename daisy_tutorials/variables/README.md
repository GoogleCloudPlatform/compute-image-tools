# Variables

This is the same as [`simple_vm.wf.json`](../vm_creation/simple_vm.wf.json), but
the VM name has been changed from `foo-instance` to `${vm-name}`. The `Vars`
workflow field has also been created with the variable `vm-name` set to `myvar`.

To run this example, you need to replace some values in the workflow configs. In
particular, you must at least change the workflow `Project`, `GCSPath`, and
`OAuthPath` fields (or override them using their respective flags). For
`OAuthPath`, you may remove it entirely if you wish to use the
[application-default
credentials](#https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login)
for the user running Daisy.

### Take aways:

* The VM Name field evaluated`${vm-name}` as `myvar`.
* We also had to change the `wait` step's WaitForInstancesSignal VM Name field
  to `${vm-name}`
* Running Daisy with `-variables vm-name=differentvar` would override the
  hardcoded `myvar`.
