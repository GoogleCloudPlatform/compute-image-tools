# Daisy Examples
This is a collection of Daisy workflow examples. To run any of these examples,
you need to replace some values in the workflow configs. In particular, you
must at least change the workflow `Project`, `GCSPath`, and `OAuthPath` fields
(or override them using their respective flags). For `OAuthPath`, you may
remove it entirely if you wish to use the [application-default credentials](#https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login)
for the user running Daisy.

## Simple example - VM creation: simple_vm* files
This is an example of creating a VM with a startup script. The script simply
prints "Hello, World!" then shuts down. The workflow has a wait step that waits
for the VM to stop.

Take aways from this example:
* Dependencies: the step `run` waited for `setup` to complete and the step 
  `wait` waited for `run` to complete.
* Sources: the created VM used the `startup` Source file. Note how `startup`
  references `simple_vm_startup.sh`, a local file. These local filepaths are
  relative.
* Automatic cleanup: Daisy automatically deleted the disk and VM after the
  workflow terminated. (auto cleanup can be turned off for any resource)
* VM Serial Port streaming: the serial port was streamed to a subdirectory
  within `GCSPath`. Make sure to take a look.

## SubWorkflows: subworkflow.wf.json and simple_vm* files.
This is a wrapper around the VM creation script. The parent workflow, 
`subworkflow.wf.json` uses `simple_vm.wf.json` as a SubWorkflow.

Take aways:
* The SubWorkflow step field `Path` is the relative local path to the workflow
  file being used. In this case, `simple_vm.wf.json`

## Variables: vars.wf.json and simple_vm* files.
This is the same as `simple_vm.wf.json`, but the VM name has been changed from
`foo-instance` to `${vm-name}`. The `Vars` workflow field has also been created
with the variable `vm-name` set to `myvar`.

Take aways:
* The VM Name field evaluated`${vm-name}` as `myvar`.
* We also had to change the `wait` step's WaitForInstancesSignal VM Name field
  to `${vm-name}`
* Running Daisy with `-variables vm-name=differentvar` would override the
  hardcoded `myvar`.
