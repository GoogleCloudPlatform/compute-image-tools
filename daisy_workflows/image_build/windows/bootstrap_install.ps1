$ErrorActionPreference = 'Stop'

function Get-MetadataValue {
  <#
    .SYNOPSIS
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

function Download-Drivers {
  <#
    .SYNOPSIS
      Downloads a drivers from GCE to the local builder directory.
  #>
  $gs_path = Get-MetadataValue -key 'drivers-path'
  if ($gs_path) {
    Write-Output "Downloading components from $gs_path."
    & 'gsutil' -m cp -r "${gs_path}/*" $script:driver_dir
    Write-Output 'Download complete.'
  }
  else {
    throw "Failed to find key 'drivers-path'. Builder cannot continue."
  }
}

function Download-Components {
  <#
    .SYNOPSIS
      Downloads a builder component from GCE to the local builder directory.
  #>
  $gs_path = Get-MetadataValue -key 'components-path'
  if ($gs_path) {
    Write-Output "Downloading components from $gs_path."
    & 'gsutil' -m cp -r "${gs_path}/*" $script:components_dir
    Write-Output 'Download complete.'
  }
  else {
    throw "Failed to find key 'components-path'. Builder cannot continue."
  }
}

function Format-InstallDisk {
  <#
    .SYNOPSIS
      Clears then formats the install disk and assigns as D:
  #>
  Write-Output 'Formatting install disk.'
  Set-Disk -Number 1 -IsOffline $false
  Clear-Disk -Number 1 -RemoveData -Confirm:$false -ErrorAction SilentlyContinue
  Initialize-Disk -Number 1 -PartitionStyle MBR
  New-Partition -DiskNumber 1 -UseMaximumSize -DriveLetter D -IsActive |
    Format-Volume -FileSystem 'NTFS' -Confirm:$false
  Write-Output 'Formatting install disk complete.'
}

function Get-Wim {
  $iso = "${script:components_dir}\windows.iso"
  if (Test-Path $iso) {
    $mount_result = Mount-DiskImage -ImagePath $iso -PassThru
    $iso_drive = ($mount_result | Get-Volume).DriveLetter
  }
  else {
    throw "Failed to find iso:'${iso}'. Builder cannot continue."
  }

  return "${iso_drive}:\sources\install.wim"
}

function Bootstrap-InstallDisk {
  <#
    .SYNOPSIS
      Mount the ISO and image the disk.
  #>

  $edition = Get-MetadataValue -key 'edition'

  # Disable Defender real time monitoring to greatly increase image
  # expansion and patch deployment speed.
  Set-MpPreference -DisableRealtimeMonitoring $true

  Write-Output 'Applying wim image to install disk.'
  Expand-WindowsImage -ImagePath $(Get-Wim) -Name $edition -ApplyPath D:\

  if (Test-Path $script:updates_dir) {
    # Add-WindowsPackage will install in alphabetical order.
    Write-Output 'Slipstreaming updates into image.'
    Get-ChildItem $script:updates_dir | ForEach-Object {
      Write-Output "Adding '$($_.Name)' to image."
      $path = $_.FullName
      try {
        Add-WindowsPackage -Path D:\ -PackagePath $path -ErrorAction Stop
        Write-Output "Successfully added $($_.Name)."
      }
      catch {
        Write-Output 'Slipstreaming update failed, trying once more...'
        Add-WindowsPackage -Path D:\ -PackagePath $path
      }
    }
  }

  Write-Output 'Populating Unattend file.'
  $autounattend_template = "${script:components_dir}\Autounattend-template.xml"
  $autounattend_file = "${script:components_dir}\Autounattend.xml"
  if (-not (Test-Path $autounattend_template)) {
    throw "Failed to find '${autounattend_template}'. Builder cannot continue."
  }

  $xml = [xml](Get-Content $autounattend_template)
  if ($product_key_string.length -gt 0) {
    Write-Output "Product key string $product_key_string."
    $new_element = $xml.CreateElement('ProductKey')
    $new_element.set_InnerText($product_key_string)
    $xml.unattend.settings[1].component[0].AppendChild($new_element)
  }
  $xml.Save($autounattend_file)

  $panther = 'D:\Windows\Panther'
  if (Test-Path $panther) {
    Remove-Item $panther -Recurse -Force
  }
  New-Item  $panther -Type Directory | Out-Null
  Copy-Item $autounattend_file "${panther}\unattend.xml"

  Write-Output 'Applying Unattend settings.'
  Use-WindowsUnattend -Path D:\ -UnattendPath $autounattend_file | Out-Null

  Write-Output 'Copying SetupComplete.cmd.'
  $scripts = 'D:\Windows\Setup\Scripts'
  New-Item  $scripts -Type Directory | Out-Null
  Copy-Item "${script:components_dir}\SetupComplete.cmd" $scripts

  Write-Output 'Copying netkvmco.dll.'
  Copy-Item $script:driver_dir\netkvmco.dll D:\Windows\System32\netkvmco.dll

  Write-Output 'Slipstreaming drivers.'
  Add-WindowsDriver -Path D:\ -Driver $script:driver_dir -Recurse -Verbose

  Write-Output 'Done applying image.'

  if (-not (Test-Path D:\Windows)) {
    throw 'Windows not installed!'
  }

  Write-Output 'Setting up bootloader.'
  & bcdboot D:\Windows /s D:

  Write-Output 'Disabling startup animation.'
  # See http://support.microsoft.com/kb/2955372/en-us
  reg load HKLM\MountedSoftware D:\Windows\System32\config\SOFTWARE
  reg add HKLM\MountedSoftware\Microsoft\Windows\CurrentVersion\Authentication\LogonUI /v AnimationDisabled /t REG_DWORD /d 1 /f
  reg unload HKLM\MountedSoftware
}

try {
  # Setup builder directories.
  $script:builder_dir = 'C:\builder'
  $script:components_dir = "$script:builder_dir\components"
  $script:updates_dir = "$script:components_dir\updates"
  $script:driver_dir = "$script:components_dir\drivers"
  New-Item $script:builder_dir -Type directory
  New-Item $script:driver_dir -Type directory

  Write-Output 'Boostrapping Windows install disk.'
  Download-Components
  Download-Drivers
  Format-InstallDisk
  Bootstrap-InstallDisk

  Write-Output 'Setting up repo.'
  $repo = Get-MetadataValue -key 'google-cloud-repo'
  & 'C:\ProgramData\GooGet\googet.exe' -root 'D:\ProgramData\GooGet' addrepo "google-compute-engine-${repo}" "https://packages.cloud.google.com/yuck/repos/google-compute-engine-${repo}"

  Write-Output 'Bootstrapping Complete, shutting down...'
  Stop-Computer
}
catch {
  Write-Output 'Exception caught in script:'
  Write-Output $_.InvocationInfo.PositionMessage
  Write-Output "Message: $($_.Exception.Message)"
  Write-Output 'Windows build failed.'
  exit 1
}
