$ErrorActionPreference = 'Stop'

$script:os_drive = ''
$script:components_dir = 'c:\components'

function Run-Command {
 [CmdletBinding(SupportsShouldProcess=$true)]
  param (
    [Parameter(Mandatory=$true, ValueFromPipelineByPropertyName=$true)]
      [string]$Executable,
    [Parameter(ValueFromRemainingArguments=$true,
               ValueFromPipelineByPropertyName=$true)]
      $Arguments = $null
  )
  Write-Output "Running $Executable with arguments $Arguments."
  $out = &$executable $arguments 2>&1 | Out-String
  $out.Trim()
}

function Get-MetadataValue {
  param (
    [parameter(Mandatory=$true)]
      [string]$key,
    [parameter(Mandatory=$false)]
      [string]$default
  )

  # Returns the provided metadata value for a given key.
  $url = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$key"
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
      Write-Host "Failed to retrieve value for $key."
      return $null
    }
  }
}

function Setup-ScriptRunner {
  $metadata_scripts = "${script:os_drive}\Program Files\Google\Compute Engine\metadata_scripts"
  New-Item "${metadata_scripts}\" -ItemType Directory | Out-Null
  Copy-Item "${script:components_dir}\GCEMetadataScripts.exe" "${metadata_scripts}\GCEMetadataScripts.exe" -Verbose
  Copy-Item "${script:components_dir}\run_startup_scripts.cmd" "${metadata_scripts}\run_startup_scripts.cmd" -Verbose
  # This file must be unicode with no trailing new line and exactly match the source.
  (Get-Content "${script:components_dir}\GCEStartup" | Out-String).TrimEnd() | Out-File -Encoding Unicode -NoNewline "${script:os_drive}\Windows\System32\Tasks\GCEStartup"

  Run-Command reg load 'HKLM\MountedSoftware' "${script:os_drive}\Windows\System32\config\SOFTWARE"
  $dst = 'HKLM:\MountedSoftware\Microsoft\Windows NT\CurrentVersion\Schedule\TaskCache'

  $acl = Get-ACL $dst
  $admin = New-Object System.Security.Principal.NTAccount('Builtin', 'Administrators')
  $acl.SetOwner($admin)
  $ace = New-Object System.Security.AccessControl.RegistryAccessRule(
    $admin,
    [System.Security.AccessControl.RegistryRights]'FullControl',
    [System.Security.AccessControl.InheritanceFlags]::ObjectInherit,
    [System.Security.AccessControl.PropagationFlags]::InheritOnly,
    [System.Security.AccessControl.AccessControlType]::Allow
  )
  $acl.AddAccessRule($ace)
  Set-Acl -Path $dst -AclObject $acl

  & reg import "${script:components_dir}\GCEStartup.reg"
  # Garbage collect before unmounting.
  [gc]::collect()

  Run-Command reg unload 'HKLM\MountedSoftware'
}

try {
  Write-Output 'Beginning translation bootstrap process.'

  $bcd_drive = ''
  Get-Disk 1 | Get-Partition | ForEach-Object {
    if (Test-Path "$($_.DriveLetter):\Windows") {
      $script:os_drive = "$($_.DriveLetter):"
    }
    elseif (Test-Path "$($_.DriveLetter):\Boot\BCD") {
      $bcd_drive = "$($_.DriveLetter):"
    }
  }
  if (!$bcd_drive) {
    $bcd_drive = $script:os_drive
  }
  if (!$script:os_drive) {
    $partitions = Get-Disk 1 | Get-Partition
    throw "No Windows folder found on any partition: $partitions"
  }

  $kernel32_ver = (Get-Command "${script:os_drive}\Windows\System32\kernel32.dll").Version
  $os_version = "$($kernel32_ver.Major).$($kernel32_ver.Minor)"

  $version = Get-MetadataValue -key 'version'
  if ($version -ne $os_version) {
    throw "Incorrect Windows version to translate, mounted image is $os_version, not $version"
  }

  $driver_dir = 'c:\drivers'
  New-Item $driver_dir -Type Directory | Out-Null
  New-Item $script:components_dir -Type Directory | Out-Null

  $daisy_sources = Get-MetadataValue -key 'daisy-sources-path'

  Write-Output 'Pulling components.'
  & 'gsutil' -m cp -r "${daisy_sources}/components/*" $script:components_dir

  Write-Output 'Pulling drivers.'
  & 'gsutil' -m cp -r "${daisy_sources}/drivers/*" $driver_dir

  Copy-Item "${driver_dir}\netkvmco.dll" "${script:os_drive}\Windows\System32\netkvmco.dll" -Verbose

  Write-Output 'Slipstreaming drivers.'
  Add-WindowsDriver -Path "${script:os_drive}\" -Driver $driver_dir -Recurse -Verbose

  Write-Output 'Setting up script runner.'
  Setup-ScriptRunner

  Write-Output 'Setting up cloud repo.'
  Run-Command 'C:\ProgramData\GooGet\googet.exe' -root "${script:os_drive}\ProgramData\GooGet" addrepo 'google-compute-engine-stable' 'https://packages.cloud.google.com/yuck/repos/google-compute-engine-stable'
  Write-Output 'Copying googet.'
  Copy-Item 'C:\ProgramData\GooGet\googet.exe' "${script:os_drive}\ProgramData\GooGet\googet.exe" -Force -Verbose

  Run-Command bcdboot "${script:os_drive}\Windows" /s $bcd_drive

  # Turn off startup animation which breaks headless installation.
  # See http://support.microsoft.com/kb/2955372/en-us
  Run-Command reg load 'HKLM\MountedSoftware' "${script:os_drive}\Windows\System32\config\SOFTWARE"
  Run-Command reg add 'HKLM\MountedSoftware\Microsoft\Windows\CurrentVersion\Authentication\LogonUI' /v 'AnimationDisabled' /t 'REG_DWORD' /d 1 /f
  Run-Command reg unload 'HKLM\MountedSoftware'

  Write-Output 'Translate bootstrap complete'
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "Message: $($_.Exception.Message)"
  Write-Output 'Translate bootstrap failed'
  exit 1
}
