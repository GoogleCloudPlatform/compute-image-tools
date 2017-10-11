# Daisy Examples
This is a collection of Daisy workflow examples. To run any of these examples,
you need to replace some values in the workflow configs. In particular, you
must at least change the workflow `Project`, `GCSPath`, and `OAuthPath` fields
(or override them using their respective flags). For `OAuthPath`, you may
remove it entirely if you wish to use the [application-default credentials](#https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login)
for the user running Daisy.

## SubWorkflows: subworkflow.wf.json and simple_vm* files.
This is a wrapper around the VM creation script. The parent workflow, 
`subworkflow.wf.json` uses `simple_vm.wf.json` as a SubWorkflow.

Take aways:
* The SubWorkflow step field `Path` is the relative local path to the workflow
  file being used. In this case, `simple_vm.wf.json`
