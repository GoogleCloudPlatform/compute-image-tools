# Passing Data to Instances

There are three ways data can be made accessible to a machine created by
the CreateInstances step:

1. Short strings and flags can be passed using the `Metadata` field.
2. A single script can be run using the `StartupScript` field.
3. Files can be retrieved from the workflow's `Sources` field.

## Retrieving Metadata

Metadata can be thought of as "Cloud environment variables". Metadata is a
simple mapping between a key string and a value string. You can supply metadata
to your instance using the `Metadata` field of the `CreateInstances` step, like
so:

    {
      ...
      "Steps": {
        "my-step": {
          "CreateInstances": [ {
              "Name": "my-instance",
              "Disks": [ ... ],
              "Metadata": {
                "key1": "value1",
                "key2": "value2"
              }
          } ]
        }
      }
      ...
    }

Information passed to instances using the `Metadata` field can be retrieved by
querying (i.e. sending an HTTP GET request with something like `wget` or `curl`
to)
`http://metadata.google.internal/computeMetadata/v1/instance/attributes/{key}`
from the instance. You must use the special header `Metadata-Flavor: Google`.
See [here](https://cloud.google.com/compute/docs/storing-retrieving-metadata)
for documentation on working with metadata.

## Working with Sources

The "Sources" field in your Daisy workflow can contain links to local files, or
files in a Google Cloud Storage bucket using the URL scheme
`gs://bucket-name/file-name`. Everything listed in Sources is copied to a
scratch bucket created by Daisy. Daisy sets the special metadata key
`daisy-sources-path` to a `gs://` path to the Sources bucket.


Once you have retrieved the path to the Daisy Sources bucket, you can use
[`gsutil`](https://cloud.google.com/storage/docs/object-basics#download) (which
should come pre-installed on all Compute Engine VMs) to download files to the
instance.

An example:

Given a Daisy workflow like this:

    {
      ...
      "Sources": {
        "file.txt": "gs://my-bucket/some-file.txt",
        "startup-script.ps1": "./dostuff.ps1"
      },
      ...
      "Steps": {
        ...
        "create-my-instance": {
          "CreateInstances": [ {
            ...
            "StartupScript": "startup-script.ps1",
            ...
          } ]
        },
        ...
      }
    }

The script `startup-script.ps1` can do the following to download `file.txt` from
the Sources bucket:

    $client = New-Object Net.WebClient
    $client.Headers.Add('Metadata-Flavor', 'Google')
    $url_to_query = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/daisy-sources-path'
    $daisy_sources_path = ($client.DownloadString($url_to_query)).Trim()
    & gsutil cp "${daisy_sources_path}/file.txt" file.txt

This approach can be used to quickly and reliably download other scripts,
installers, and even very large files. Any instance created by Daisy will
automatically be given access to the appropriate Sources bucket.

