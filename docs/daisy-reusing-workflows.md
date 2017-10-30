# Reusing Workflows

Instead of requiring you to copy-paste large sections of workflows to perform
similar tasks, Daisy provides two different step types for workflow reuse:
`IncludeWorkflow` and `SubWorkflow`. Both step types will cause Daisy to run the
workflow they specify, but how Daisy does this is different:

### IncludeWorkflow

`IncludeWorkflow` will run the steps of the included workflow in parallel with
the parent workflow's steps. A workflow included with `IncludeWorkflow` will
behave largely as if the included workflow had been copy-pasted into the parent
workflow. The parent workflow will continue in parallel to the included
workflow, if permitted by both of their dependencies. Note that if the
`IncludeWorkflow` step depends on other steps, the workflow will not be included
before those steps it depends on have completed.

A parent workflow has access to its child workflow's resources, and vice versa.
Disks, instances, steps, etc. present in either workflow can be referenced by
name from either workflow. Two children with a common parent can access each
other's resources.

See also the
[documentation](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#type-includeworkflow)
for IncludeWorkflow.

### SubWorkflow

`SubWorkflow` runs the subworkflow specified as one step. Subworkflows do not
have access to their parent's resources. Additionally, the subworkflow inherits
its parent's `Project`, `Zone`, `GCSPath`, and `OAuthPath` fields. Even if these
values are set in the subworkflow, they will be overwritten by the parent's
values.

Unlike `IncludeWorkflow`, which is considered complete as soon as the included
workflow has been read and its steps scheduled, `SubWorkflow` treats the entire
subworkflow as a step, and is not considered complete until the entire
subworkflow has finished. This has important implications for dependencies: a
step which depends on an `IncludeWorkflow` step will run in parallel to that
included workflow (if not otherwise restricted by other dependencies) while a
step which depends on a `SubWorkflow` step will not run until that subworkflow
has completed.

## Using Vars

To allow parent workflows to control the behavior of their children, they can
set the `Vars` of a child workflow. Vars are represented by simple JSON objects
with three fields: `Value`, `Description`, and `Required`. Both
`IncludeWorkflow` and `SubWorkflow` tasks provide their own `Vars` field, which
allows the parent workflow to set the `Vars` of the child workflow. If a child
has a Var marked `Required` that is not set by the parent, an error will occur.

The typical use case for parent/child Vars is to set the `Required` and
`Description` of the child's Vars (but not the `Value`), and then pass values
from the parent. Here is an example of the child:

    {
      "Name": "child",
      "Vars": {
        "var1": { "Required": true, "Description": "passed from the parent" },
        "var2": { "Required": true, "Description": "also important" }
      },
      "Steps:" { ...  },
      "Dependencies": { ... }
    }

And the corresponding parent:

    {
      "Name": "parent",
      ...
      "Steps": {
        "include-step": {
          "IncludeWorkflow": {
            "Path": "path-to-child.wf.json",
            "Vars": {
              "var1": "value1",
              "var2": "value2"
            }
          }
        },
        ...
      },
      "Dependencies": { ... }
    }


