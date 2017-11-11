# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

$ErrorActionPreference = 'Stop'

function Format-ScratchDisk {
  <#
    .SYNOPSIS
      Clears then formats the scratch disk and assigns as D:
  #>
  Write-Host 'Formatting scratch disk.'
  Set-Disk -Number 1 -IsOffline $false
  Initialize-Disk -Number 1 -PartitionStyle MBR
  New-Partition -DiskNumber 1 -UseMaximumSize -DriveLetter D -IsActive |
    Format-Volume -FileSystem 'NTFS' -Confirm:$false
  New-Item $log_path -Type Directory
  Write-Host 'Formatting scratch disk complete.'
}

function Get-MetadataValue {
  <#
    .SYNOPSIS
      Returns a value for a given metadata key.

    .DESCRIPTION
      Attempt to retrieve the value for a given metadata key.
      Returns null if not found.

    .PARAMETER $key
      The metadata key to retrieve.

    .PARAMETER $default
      The value to return if the key is not found.

    .RETURNS
      The value for the key or null.
  #>
  param (
    [parameter(Mandatory=$true)]
      [string]$key,
    [parameter(Mandatory=$false)]
      [string]$default
  )

  # Returns the provided metadata value for a given key.
  $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/${key}"
  try {
    $client = New-Object Net.WebClient
    $client.Headers.Add('Metadata-Flavor', 'Google')
    return ($client.DownloadString($url)).Trim()
  }
  catch [System.Net.WebException] {
    if ($default) {
      return $default
    }
    else {
      Write-Host "Failed to retrieve value for ${key}."
      return $null
    }
  }
}

function Install-WindowsUpdates {
  <#
    .SYNOPSIS
      Check for updates, returns true if restart is required.
  #>

  Write-Host 'Starting Windows update.'
  if (-not (Test-Connection download.microsoft.com -Count 1 -ErrorAction SilentlyContinue)) {
    throw 'Windows update server is not reachable. Cannot complete image build.'
  }

  # If MSFT_WUOperationsSession exists use that.
  $ci = New-CimInstance -Namespace root/Microsoft/Windows/WindowsUpdate -ClassName MSFT_WUOperationsSession -ErrorAction SilentlyContinue
  if ($ci) {
    $scan = $ci | Invoke-CimMethod -MethodName ScanForUpdates -Arguments @{SearchCriteria='IsInstalled=0';OnlineScan=$true}
    if ($scan.Updates.Count -eq 0) {
      Write-Host 'No updates to install'
      return $false
    }
    Write-Host "Downloading $($scan.Updates.Count) updates"
    $download = $ci | Invoke-CimMethod -MethodName DownloadUpdates -Arguments @{Updates=$scan.Updates}
    Write-Host "Download finished with HResult: $($download.HResult)"
    Write-Host "Installing $($scan.Updates.Count) updates"
    $install = $ci | Invoke-CimMethod -MethodName InstallUpdates -Arguments @{Updates=$scan.Updates}
    Write-Host "Install finished with HResult: $($install.HResult)"
    Write-Host 'Finished Windows update.'

    return (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired')
  }

  # In 2008 R2 the initial search can fail with error 0x80244010, something
  # to do with the number of trips the client is making to the WSUS server.
  # Trying the search again fixes this. Searching around the net didn't
  # yield any actual fixes other than trying again. It does seem to be a
  # somewhat common issue that has been around for years and has no other fix.
  # http://blogs.technet.com/b/sus/archive/2008/09/18/wsus-clients-fail-with-warning-syncserverupdatesinternal-failed-0x80244010.aspx
  $session = New-Object -ComObject 'Microsoft.Update.Session'
  $query = 'IsInstalled=0'
  $searcher = $session.CreateUpdateSearcher()
  $i = 1
  while ($i -lt 10) {
    try {
      Write-Host "Searching for updates, try $i."
      $updates = $searcher.Search($query).Updates
      break
    } catch {
      Write-Host 'Update search failed.'
      $i++
      if ($i -ge 10) {
        Write-Host 'Searching for updates one last time'
        $updates = $searcher.Search($query).Updates
      }
    }
  }

  if ($updates.Count -eq 0) {
    Write-Host 'No updates required!'
    return $false
  }
  else {
    foreach ($update in $updates) {
      if (-not ($update.EulaAccepted)) {
        Write-Host 'The following update required a EULA to be accepted:'
        Write-Host '----------------------------------------------------'
        Write-Host ($update.Description)
        Write-Host '----------------------------------------------------'
        Write-Host ($update.EulaText)
        Write-Host '----------------------------------------------------'
        $update.AcceptEula()
      }
    }
    $count = $updates.Count
    if ($count -eq 1) {
      # Sometimes we have a bug where we get stuck on one update. Let's
      # log what this one update is in case we are having trouble with it.
      Write-Host 'Downloading the following update:'
      Write-Host ($updates | Out-String)
    }
    else {
      Write-Host "Downloading $count updates."
    }
    $downloader = $session.CreateUpdateDownloader()
    $downloader.Updates = $updates
    $downloader.Download()
    Write-Host 'Download complete. Installing updates.'
    $installer = $session.CreateUpdateInstaller()
    $installer.Updates = $updates
    $installer.AllowSourcePrompts = $false
    $result = $installer.Install()
    $hresult = $result.HResult
    Write-Host "Install Finished with HResult: $hresult"
    Write-Host 'Finished Windows update.'
    return $result.RebootRequired
  }
}

function Install-SqlServer {
  Write-Host 'Beginning SQL Server install.'

  $sql_server_media = Get-MetadataValue -key 'sql-server-media'
  $sql_server_config = Get-MetadataValue -key 'sql-server-config'
  $gs_path = Get-MetadataValue -key 'daisy-sources-path'
  $sql_install = 'C:\sql_server_install'

  $sql_config_path = "${gs_path}/sql_config.ini"
  $sql_config = 'D:\sql_config.ini'
  & 'gsutil' -m cp $sql_config_path $sql_config

  if ($sql_server_config -like '*2012*' -or $sql_server_config -like '*2014*') {
    Write-Host 'Installing .Net 3.5'
    Install-WindowsFeature Net-Framework-Core
  }

  if ($sql_server_media -like '*.iso') {
    Write-Host 'Downloading SQL Server ISO'
    $iso = 'D:\sql_server.iso'
    & 'gsutil' -m cp "${gs_path}/sql_installer.media" $iso
    Write-Host 'Mount ISO'
    $mount_result = Mount-DiskImage -ImagePath $iso -PassThru
    $iso_drive = ($mount_result | Get-Volume).DriveLetter

    Write-Host 'Copying ISO contents'
    New-Item $sql_install -Type Directory
    Copy-Item "${iso_drive}:\*" $sql_install -Recurse

    Write-Host 'Unmounting ISO'
    Dismount-DiskImage -ImagePath $iso
  }
  elseif ($sql_server_media -like '*.exe') {
    Write-Host 'Downloading SQL Server exe'
    $exe = 'D:\sql_server.exe'
    & 'gsutil' -m cp "${gs_path}/sql_installer.media" $exe
    Start-Process $exe -ArgumentList @("/x:${sql_install}",'/u') -Wait
  }
  else {
    throw "Install media not iso or exe: ${sql_server_media}"
  }

  Write-Host 'Opening port 1433 for SQL Server'
  & netsh advfirewall firewall add rule name='SQL Server' dir=in action=allow protocol=TCP localport=1433

  Write-Host 'Installing SQL Server'
  & "${sql_install}\setup.exe" "/ConfigurationFile=${sql_config}"
  Write-Host 'Finished installing SQL Server'
}

function Install-SSMS {
  $sql_server_config = Get-MetadataValue -key 'sql-server-config'
  if ($sql_server_config -notlike '*2016*' -and $sql_server_config -notlike '*2017*' -or $sql_server_config -like '*core*') {
    Write-Host "Not installing SSMS for config ${sql_server_config}"
    return
  }
  Write-Host 'Installing SSMS'

  $gs_path = Get-MetadataValue -key 'daisy-sources-path'
  $ssms_exe = 'D:\SSMS-Setup-ENU.exe'
  & 'gsutil' -m cp "${gs_path}/SSMS-Setup-ENU.exe" $ssms_exe

  Start-Process $ssms_exe -ArgumentList @('/install','/quiet','/norestart') -Wait

  Write-Host 'Finished installing SSMS'
}

try {
  if (!(Test-Path 'D:\')) {
    $sysprep = 'c:\Windows\System32\Sysprep'
    Remove-Item "${sysprep}\Panther\*" -Recurse -Force -ErrorAction Continue
    Remove-Item "${sysprep}\Sysprep_succeeded.tag" -Recurse -Force -ErrorAction Continue
    Format-ScratchDisk
    Install-SqlServer
    Install-SSMS
  }

  $reboot_required = Install-WindowsUpdates
  if ($reboot_required) {
    Write-Host 'Reboot required.'
    Restart-Computer
    exit
  }

  if (-not (Test-Path 'C:\Program Files\Microsoft SQL Server')) {
    throw 'SQL Server is not installed.'
  }

  Write-Host 'Launching sysprep.'
  & 'C:\Program Files\Google\Compute Engine\sysprep\gcesysprep.bat' > "${log_path}\sysprep.txt" 2>&1
}
catch {
  Write-Host $_.Exception.Message
  Write-Host 'SQL image build failed'
  exit 1
}
