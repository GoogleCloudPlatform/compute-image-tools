# VM Creation

This is an example of creating a VM with a startup script. The script simply
prints "Hello, World!" then shuts down. The workflow has a wait step that waits
for the VM to stop.

To run this example, you need to replace some values in the workflow configs. In
particular, you must at least change the workflow `Project`, `GCSPath`, and
`OAuthPath` fields (or override them using their respective flags). For
`OAuthPath`, you may remove it entirely if you wish to use the
[application-default
credentials](#https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login)
for the user running Daisy.

### Take aways from this example:

* Dependencies: the step `run` waited for `setup` to complete and the step
  `wait` waited for `run` to complete.
* Sources: the created VM used the `startup` Source file. Note how `startup`
  references `simple_vm_startup.sh`, a local file. These local file paths are
  relative.
* Automatic cleanup: Daisy automatically deleted the disk and VM after the
  workflow terminated. (auto cleanup can be turned off for any resource)
* VM Serial Port streaming: the serial port was streamed to a subdirectory
  within `GCSPath`. Make sure to take a look.
