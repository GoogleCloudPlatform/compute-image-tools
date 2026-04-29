## Daisy sqlserver image build workflow
Builds a SQL Server derivative image. 
Proper install media must be provided.
Required vars:
+ `source_image` GCE image to base build on
+ `sql_server_media` GCS or local path to SQLServer installer media
+ `sql_server_config` GCS or local path to SQLServer config ini file

Optional vars:
+ `ssms_exe` GCS or local path to SQL Server Management Studio (SSMS) bootstrapper
+ `install_disk` Name of the GCE disk to use for the SQL install
+ `sbom_destination` The path to export the SBOM file, if it is generated
+ `sbom_util_gcs_root` The path where the sbomutil executable is stored

### SQL Server 2025 Standard workflow example
This workflow builds a SQL Server 2025 Standard Edition image on the latest Windows Server 2025 GCE image.
Different versions of SQL Server can be installed by replacing `sql_server_media` with the 
correct media. Be sure to use the correct `sql_server_config` year version to match that of 
the media used.
The image completed image will created in the project the Daisy workflow is run in. 
See daisy_workflows/image_build/sqlserver/sql-2025-standard-windows-2025-dc.wf.json for the workflow json definition.

### SQL Server Install Configuration
To build SQL Server images, we install SQL Server via configuration files (e.g. .ini file). To generate a configuration file, see 
https://learn.microsoft.com/en-us/sql/database-engine/install-windows/install-sql-server-using-a-configuration-file?view=sql-server-ver17 .
SQL Server ISOs for this install can be found in MS admin center.

### SQL Server Management Studio Installation
Microsoft currently uses a bootstrapper to download and install the latest SSMS (currently 22, as of 12/8/2025). 

SSMS media for this install can be found here:
https://learn.microsoft.com/en-us/ssms/install/install

### SQL Server installation script
The installation definitions of SQL products are defined in daisy_workflows/image_build/sqlserver/sql_install.ps1.
This file may need to be modified if installation workflow changes.
