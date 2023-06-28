## Daisy sqlserver image build workflow
Builds a SQL Server derivative image. 
Proper install media must be provided.

Required vars:
+ `source_image` GCE image to base build on
+ `sql_server_media` GCS or local path to SQLServer installer media
+ `sql_server_config` GCS or local path to SQLServer config ini file

Optional vars:
+ `ssms_exe` GCS or local path to SSMS installer for SQL 2016 installs
+ `install_disk` Name of the GCE disk to use for the SQL install
+ `sbom_destination` The path to export the SBOM file, if it is generated
+ `sbom_util_gcs_root` The path where the sbomutil executable is stored

### SQL 2016 Express workflow example
This workflow builds a SQl Server 2016 Express image on the latest Server 2016 GCE image.
Different versions of SQL Server can be installed by replacing `sql_server_media` with the 
correct media. Be sure to use the correct `sql_server_config` year version to match that of 
the media used.
The image completed image will created in the project the Daisy workflow is run in

SQl Server media for this install can be found here:
https://www.microsoft.com/en-us/sql-server/sql-server-editions-express

SSMS media for this install can be found here:
https://docs.microsoft.com/en-us/sql/ssms/download-sql-server-management-studio-ssms

```json
{
  "Name": "sql-2016-express-windows-2016-dc-image-build",
  "Project": "MYPROJECT",
  "Zone": "MYZONE",
  "GCSPath": "gs://MYBUCKET/daisy/${USERNAME}",
  "Vars": {
    "install_disk": "disk-install"
  },
  "Steps": {
    "build-sql-image": {
      "TimeOut": "70m",
      "IncludeWorkflow": {
        "Path": "sqlserver.wf.json",
        "Vars": {
          "sql_server_config": "./configs/sql_server_2016_config.ini",
          "sql_server_media": "gs://PATH/TO/SQLEXPRADV_x64_ENU.exe",
          "source_image": "projects/windows-cloud/global/images/family/windows-2016",
          "install_disk": "${install_disk}",
          "ssms_exe": "gs://PATH/TO/SSMS-Setup-ENU.exe",
          "sbom_destination": "gs://EXPORT_PATH",
          "sbom_util_gcs_root": "${sbom_util_gcs_root}"
        }
      }
    },
    "create-image": {
      "CreateImages": [
        {
          "Name": "sql-2016-express-windows-2016-dc-v${DATE}",
          "SourceDisk": "${install_disk}",
          "Description": "Microsoft, SQL Server 2016 Express, on Windows Server 2016, x64 built on ${DATE}",
          "GuestOsFeatures": [{"Type":"VIRTIO_SCSI_MULTIQUEUE"}, {"Type":"WINDOWS"}],
          "Family": "sql-exp-2016-win-2016",
          "NoCleanup": true,
          "ExactName": true
        }
      ]
    }
  },
  "Dependencies": {
    "create-image": ["build-sql-image"]
  }
}
```
 
