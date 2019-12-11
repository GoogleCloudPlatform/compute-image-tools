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
        "SourceImage": "projects/debian-cloud/global/images/family/debian-9",
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
consistent throughout. Obviously, if the resources are not ephemeral (if they
are to be used outside of the workflow), name them whatever suits the need.

* Disks:
`"Name": "disk-debian-build"`

* Instances:
`"Name": "instance-debian-builder"` or `"Name": "inst-debian-builder"`

* Images:
`"Name": "image-redhat-installer"` or `"Name": "img-redhat-installer"`

## Variable naming
Name variables all lowercase with underscores and make them explicit. For
example, don't name a variable `image` instead use an explicit name
`debian_image` or `debian_base_image`. Upper case variable names are reserved
for Daisy autovars.

```json
"Vars": {
  "debian_base_image": "projects/debian-cloud/global/images/family/debian-9"
}
```

## Variable annotations
Any variable can include a description, which is useful in and of itself as an
annotation. Required variables should always include a description. The
description will be printed to the user in case they forget to define the
required variable. Optional variables do not explicitly need a description but
it is helpful to annotate the purpose of the variable for users of the workflow.

```json
"Vars": {
  "image_size_gb": "10",
  "image_name": {"Required": true, "Description": "The name of the resulting
  image being created."},
  "debian_base_image": {"Value": "projects/debian-cloud/global/images/family/debian-9",
                        "Description": "The Debian base image family to build
                        an image from."}
}
```


## Subworkflow usage
TBD
