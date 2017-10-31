# Reusing Workflows

Instead of requiring you to copy-paste large sections of workflows to perform
similar tasks, Daisy provides two different step types for workflow reuse:
`IncludeWorkflow` and `SubWorkflow`. Both step types will cause Daisy to run the
workflow they specify, but how Daisy does this is different:

### IncludeWorkflow

The parent workflow of a workflow included using `IncludeWorkflow` has access to
the child workflow's resources, and vice versa. Disks, instances, steps, etc.
present in either workflow can be referenced by name from either workflow. Two
children with a common parent can also access each other's resources. It is up
to you to ensure there are no naming collisions amongst included workflows and
their parents.

`IncludeWorkflow` will run the steps of the included workflow in parallel with
the parent workflow's steps, according to the dependency rules of both the
parent and included workflows. A workflow included with `IncludeWorkflow` will
behave largely as if the included workflow had been copy-pasted into the parent
workflow.

An `IncludeWorkflow` step is considered "done" (for the purposes of
dependencies) as soon as the child workflow has been read. A step which only
depends on an `IncludeWorkflow` step is likely to run before all of the steps in
the included workflow have completed. Steps of the parent workflow which should
depend on a step in the child workflow must depend on *that step* within the
child workflow, and not on the `IncludeWorkflow` step of the parent. Note that
if the `IncludeWorkflow` step depends on other steps (that is, it is not run at
the very beginning of the parent workflow), the child workflow will not be
included before those steps it depends on have completed, so any task which
makes reference to the child workflow's resources should depend on the
`IncludeWorkflow` task which includes that child.

See also the [documentation](daisy-workflow-config-spec.md#type-includeworkflow)
for IncludeWorkflow.

### SubWorkflow

`SubWorkflow` runs the subworkflow specified as one step. Subworkflows do not
have access to their parent's resources. This means that any workflow can be
used as a subworkflow without fear of resource name collisions. Additionally,
the subworkflow inherits its parent's `Project`, `Zone`, `GCSPath`, and
`OAuthPath` fields. Even if these fields are set in the subworkflow, they will
be overwritten with the parent's values.

Unlike `IncludeWorkflow`, which is considered complete as soon as the included
workflow has been read and its steps scheduled, `SubWorkflow` treats the entire
subworkflow as a step, and is not considered complete until the entire
subworkflow has finished. This has important implications for dependencies: a
step which depends on an `IncludeWorkflow` step will run in parallel to that
included workflow (if not otherwise restricted by other dependencies) while a
step which depends on a `SubWorkflow` step will not run until that subworkflow
has completed.

See also the [documentation](daisy-workflow-config-spec.md#type-subworkflow) for
SubWorkflow.

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


