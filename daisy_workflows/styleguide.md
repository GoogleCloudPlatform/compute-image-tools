# Daisy Workflow Styleguide
Recommendations from the GCE team about how to style daisy workflows.

## Workflow file naming
Use `.wf.json` for workflow names and prefer underscores in file names.

For example: `my_awesome_build.wf.json`.

## Workflow naming
Use dashed names for workflow names. `Name": "my-awesome-build"`

## Step naming
Use dashed names for steps such as `create-build-disk`. Don't use camel
case for step naming as this looks too much like the Steps themselves.

For example:

```json
"Steps": {
  "create-build-disk": {
    "CreateDisks": [
      {
        "Name": "disk-debian-build",
        "SourceImage": "projects/bct-prod-images/global/images/family/debian-8",
        "SizeGb": "10",
        "Type": "pd-ssd"
      }
    ]
  }
}
```

## Resource naming
Name resources with a prefix denoting the resource type. This will make it much
easier to reference these resources. Use dashes instead of camel case or
underscores for resource names. Short hand is acceptable as long as you are
consistent throughout. Obviously, if the resources are not ephermeral (if they
are to be used outside of the workflow), name them whatever suits the need.

* Disks:
`"Name": "disk-debian-build"`

* Instances:
`"Name": "instance-debian-builder"` or `"Name": "inst-debian-builder"`

* Images:
`"Name": "image-redhat-installer"` or `"Name": "img-redhat-installer"`

## Variable naming
Name variables with underscores and make them explicit. For example, don't name
a variable `image` instead use an explicit name `debian_image` or `debian_base_image`.

```json
"Vars": {
  "debian_base_image": "projects/bct-prod-images/global/images/family/debian-8"
}
```

## Subworkflow usage
TBD
