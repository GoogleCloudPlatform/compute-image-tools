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

  # https://support.microsoft.com/en-us/help/4072698/windows-server-guidance-to-protect-against-the-speculative-execution
  if (-not (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat')) {
    New-Item -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Type Directory | Out-Null
  }
  New-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\QualityCompat' -Name 'cadca5fe-87d3-4b96-b7fb-a231484277cc' -Value 0  -PropertyType DWORD -Force | Out-Null

  Write-Host 'Install-WindowsUpdates: Starting Windows update.'

  # In 2008R2 the initial search can fail with error 0x80244010. Retrying the search again generally resolves the issueis.
  $session = New-Object -ComObject 'Microsoft.Update.Session'
  $query = 'IsInstalled=0'
  $searcher = $session.CreateUpdateSearcher()
  $i = 1
  while ($i -lt 10) {
    try {
      Write-Host "Install-WindowsUpdates: Searching for updates, try $i."
      $updates = $searcher.Search($query).Updates

      # Skip Windows 7 optional language pack updates
      if ($pn -like 'Windows 7*') {
        if ($updates.Count -le 37 -and $updates.Count -ge 33) {
          Write-Host 'Install-WindowsUpdates: Windows 7 detected. Skipping ~35 language pack updates.'
          $query = 'IsInstalled=0 and AutoSelectOnWebsites=1'
          continue
        }
      }

      break
    } catch {
      Write-Host 'Install-WindowsUpdates: Update search failed.'
      $i++
      if ($i -ge 10) {
        Write-Host 'Install-WindowsUpdates: Reseting update server'
        Reset-WindowsUpdateServer | Out-Null
        Write-Host 'Install-WindowsUpdates: Searching for updates one last time.'
        $updates = $searcher.Search($query).Updates
      }
    }
  }

  if ($updates.Count -eq 0) {
    Write-Host 'Install-WindowsUpdates: No updates required!'
    return $false
  }

  # Windows 7 may enter a loop with a single update remaining
  if ($pn -like 'Windows 7*') {
    if ($updates.Count -eq 1) {
      Write-Host 'Install-WindowsUpdates: Windows 7 detected. Single update remaining. Displaying and continuing install.'
      foreach ($update in $updates) {
        Write-Host ($update.Description)
     }
     return $false
    }
  }

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

  Write-Host "Install-WindowsUpdates: Downloading and installing $($updates.Count) updates."
  foreach ($update in $updates) {
    Write-Host "Install-WindowsUpdates: Update - Title:$($update.Title), Description:$($update.Description)"
  }

  $downloader = $session.CreateUpdateDownloader()
  $downloader.Updates = $updates
  $download_result = $downloader.Download()
  Write-Host "Install-WindowsUpdates: Download complete. Result: $(Get-ResultCodeDescription $download_result.ResultCode). Installing updates."
  $installer = $session.CreateUpdateInstaller()
  $installer.Updates = $updates
  $installer.AllowSourcePrompts = $false
  $install_result = $installer.Install()
  Write-Host "Install-WindowsUpdates: Update installation completed. Result: $(Get-ResultCodeDescription $install_result.ResultCode)"
  return $true
}

function Get-ResultCodeDescription {
  <#
    .SYNOPSIS
      Returns the description of the Windows Update download/install ResultCode.

    .PARAMETER $ResultCode
      The ResultCode to convert.

    .RETURNS
      The human readable description of the result code.
  #>
  param (
    [Parameter(Mandatory=$true)] [int]$ResultCode
  )
  $Result = switch ($ResultCode) {
    0 { 'Not Started' }
    1 { 'In Progress' }
    2 { 'Succeeded' }
    3 { 'SucceededWithErrors' }
    4 { 'Failed' }
    5 { 'Aborted' }
    default { "Unknown, ResultCode: $ResultCode" }
  }
  return $Result
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
  if ($sql_server_config -like '*core*') {
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

function Enable-MicrosoftUpdate {
  $service_manager = New-Object -ComObject 'Microsoft.Update.ServiceManager'
  $service_manager.AddService2("7971f918-a847-4430-9279-4a52d1efe18d",7,"")
}

try {
  if (!(Test-Path 'D:\')) {
    $sysprep = 'c:\Windows\System32\Sysprep'
    Remove-Item "${sysprep}\Panther\*" -Recurse -Force -ErrorAction Continue
    Remove-Item "${sysprep}\Sysprep_succeeded.tag" -Recurse -Force -ErrorAction Continue
    Format-ScratchDisk
    Install-SqlServer
    Install-SSMS
    Enable-MicrosoftUpdate
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
  Write-Host 'Exception caught in script:'
  Write-Host $_.InvocationInfo.PositionMessage
  Write-Host "SQL build failed: $($_.Exception.Message)"
  exit 1
}
