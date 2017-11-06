# Reusing Workflows

Instead of requiring you to copy-paste large sections of workflows to perform
similar tasks, Daisy provides two different step types for workflow reuse:
`IncludeWorkflow` and `SubWorkflow`. Both step types will cause Daisy to run the
workflow they specify, but they differ slightly in how they share (or do not
share) resources.

### IncludeWorkflow

The parent workflow of a workflow included using `IncludeWorkflow` has access to
the child workflow's resources, and vice versa. Sources and named resources
(e.g. disks, instances) present in either workflow can be referenced by name
from either workflow. Two children with a common parent can also access each
other's resources. It is up to you to ensure there are no naming collisions
amongst included workflows and their parents.

`IncludeWorkflow` will run the steps of the included workflow in parallel with
the parent workflow's steps, according to the dependency rules of both the
parent and included workflows. A workflow included with `IncludeWorkflow` will
behave largely as if the included workflow had been copy-pasted into the parent
workflow.

See also the [documentation](daisy-workflow-config-spec.md#type-includeworkflow)
for IncludeWorkflow.

### SubWorkflow

`SubWorkflow` runs the subworkflow specified as one step. Subworkflows do not
have access to their parent's resources. This means that any workflow can be
used as a subworkflow without fear of resource name collisions. Additionally,
the subworkflow inherits its parent's `Project`, `Zone`, `GCSPath`, and
`OAuthPath` fields. Even if these fields are set in the subworkflow, they will
be overwritten with the parent's values.

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


