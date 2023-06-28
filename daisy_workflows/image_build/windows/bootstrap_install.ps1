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
    $value = ($client.DownloadString($url)).Trim()
    Write-Host "Retrieved metadata for key $key with value $value."
    return $value
  }
  catch [System.Net.WebException] {
    if ($default) {
      Write-Host "Failed to retrieve value for $key, returning default of $default."
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
      Downloads the drivers from GCE to the local drivers directory.
  #>
  $gs_path = Get-MetadataValue -key 'drivers-path'
  if ($gs_path) {
    Write-Output "Downloading drivers from $gs_path."
    & 'gsutil' -m cp -r "${gs_path}/*" $script:driver_dir
    Write-Output 'Driver download complete.'
  }
  else {
    throw "Download-Drivers() Failed to find metadata key 'drivers-path'. Build cannot continue."
  }
}

function Download-Components {
  <#
    .SYNOPSIS
      Downloads the components from GCE to the local components directory.
  #>
  $gs_path = Get-MetadataValue -key 'components-path'
  if ($gs_path) {
    Write-Output "Downloading components from $gs_path."
    & 'gsutil' -m cp -r "${gs_path}/*" $script:components_dir
    Write-Output 'Components download complete.'
  }
  else {
    throw "Download-Components() Failed to find metadata key 'components-path'. Build cannot continue."
  }
}

function Download-Sbomutil {
  <#
    .SYNOPSIS
      Downloads sbomutil from GCE to the local components directory.
  #>
  $gs_path = Get-MetadataValue -key 'sbom-util-gcs-root'
  if (!$gs_path) {
    Write-Output "No metadata sbom-util-gcs-root set, skipping sbomutil download."
    return
  }

  $gs_path = "${gs_path}/windows"
  $latest = gsutil ls "${gs_path}" | Select -Last 1
  if (!$latest) {
    Write-Output "Could not determine sbomutil's latest release, skipping sbomutil download."
    return
  }

  # The variable $latest already has a backslash at the end, as a result of gsutil ls.
  Write-Output "Downloading sbomutil from $gs_path."
  & 'gsutil' -m cp "${latest}sbomutil.exe" "C:\sbomutil.exe"
  Write-Output 'Components download complete.'
}

function Generate-Sbom {
  <#
    .SYNOPSIS
      Generates sbom and upload the result to a gcs bucket.
  #>
  $gs_path = Get-MetadataValue -key 'sbom-destination'
  if (!$gs_path) {
    Write-Output "No metadata sbom-destination set, skipping sbom generation."
    return
  }

  if (!(Test-Path "C:\sbomutil.exe")) {
    Write-Output "Could not find sbomutil tool, skipping sbom generation."
    return
  }

  # Comp name is a short descriptor at the top of the sbom file for the software.
  $comp_name = Get-MetadataValue -key 'edition'

  Write-Output "Generating sbom."
  & "C:\sbomutil.exe" -archetype=windows-image -googet_path 'D:\ProgramData\GooGet' -extra_content="${script:sbom_dir}\" -comp_name="${comp_name}" -output image.sbom.json
  & 'gsutil' cp image.sbom.json $gs_path
  Write-Output "Sbom file uploaded to $gs_path."
}

function Format-InstallDiskUEFI {
  <#
    .SYNOPSIS
      Clears and initializes disk 1 as GPT. Formats the install disk as D: and system partition as S: for UEFI boot.
  #>
  Write-Host 'Formatting install disk for UEFI.'
  Set-Disk -Number 1 -IsOffline $false
  Clear-Disk -Number 1 -RemoveData -Confirm:$false -ErrorAction SilentlyContinue
  Initialize-Disk -Number 1 -PartitionStyle GPT

  Write-Host 'Creating FAT32 system partition of 100MB and assigning volume drive S.'
  New-Partition -DiskNumber 1 -Size 100MB -DriveLetter S | Format-Volume -FileSystem 'FAT32' -Confirm:$false
  Write-Host 'Creating NTFS Windows partition and assigning volume drive D.'
  New-Partition -DiskNumber 1 -UseMaximumSize -DriveLetter D | Format-Volume -FileSystem 'NTFS' -Confirm:$false
  Write-Host 'Formatting UEFI install disk complete.'
}

function Format-InstallDiskMBR {
  <#
    .SYNOPSIS
      Clears and initializes disk 1 as MBR. Formats disk as NFTS and assigns as D: for MBR boot.
  #>
  Write-Host 'Formatting install disk for MBR.'
  Set-Disk -Number 1 -IsOffline $false
  Clear-Disk -Number 1 -RemoveData -Confirm:$false -ErrorAction SilentlyContinue
  Initialize-Disk -Number 1 -PartitionStyle MBR
  Write-Host 'Creating NTFS Windows partition and assigning volume drive D.'
  New-Partition -DiskNumber 1 -UseMaximumSize -DriveLetter D -IsActive | Format-Volume -FileSystem 'NTFS' -Confirm:$false
  Write-Host 'Formatting MBR install disk complete.'
}

function Get-Wim {
  <#
    .SYNOPSIS
      Mount the ISO and return the absolute path of the install.wim.
  #>
  $iso = "${script:components_dir}\windows.iso"
  if (Test-Path $iso) {
    $mount_result = Mount-DiskImage -ImagePath $iso -PassThru
    $iso_drive = ($mount_result | Get-Volume).DriveLetter
  }
  else {
    throw "Get-Wim: Failed to find iso:'$iso'. Boostrapping cannot continue."
  }
  Write-Host "ISO mounted as $iso_drive"
  $wim = "${iso_drive}:\sources\install.wim"

  if (-not (Test-Path $wim)) {
    throw "Get-Wim: Failed to find wim:'$wim'. Boostrapping cannot continue."
  }
  return $wim
}

function Bootstrap-InstallDisk {
  <#
    .SYNOPSIS
      Apply the image and do all the offline modifications to prepare the system to boot.

    .DESCRIPTION
      The following steps are completed in Bootstrap-InstallDisk:
      1. Disable Defender for increased performance.
      2. Apply the specified OS edition WIM to the install disk.
      3. Apply the offline updates to the image applied to the install disk.
      4. Update the autounattend.xml file and apply it to the image.
      5. Copy the setupcomplete.cmd.
      6. Copy netkvmco.dll and add the drivers to the image.
      7. Setup bootloader.
      8. Disable login animation.
  #>
  $slipstream_max_attemps = 2
  $edition = Get-MetadataValue -key 'edition'

  # Disable Defender real time monitoring to greatly increase image
  # expansion and patch deployment speed.
  Set-MpPreference -DisableRealtimeMonitoring $true

  Write-Output "Applying $edition wim image to install disk."
  Expand-WindowsImage -ImagePath $(Get-Wim) -Name $edition -ApplyPath D:\

  if (Test-Path $script:updates_dir) {
    Write-Output 'Slipstreaming updates into image in alphabetical order.'
    Get-ChildItem $script:updates_dir | ForEach-Object {
      Write-Output "Slipstreaming '$($_.Name)' into image."
      $path = $_.FullName
      for ($i=1; $i -le $slipstream_max_attemps; $i++) {
        try {
          Add-WindowsPackage -Path D:\ -PackagePath $path -ErrorAction Stop
          Write-Output "Successfully slipstreamed update $($_.Name) on attempt $i."
          break
        }
        catch {
          Write-Output "Failed to slipstream update '$($_.Name)' on attempt $i of $slipstream_max_attemps attempts."
        }
      }
    }
    Write-Output 'Slipstreaming updates completed.'
  }

  Write-Output 'Populating Autounattend.xml file.'
  $autounattend_template = "${script:components_dir}\Autounattend-template.xml"
  $autounattend_file = "${script:components_dir}\Autounattend.xml"
  if (-not (Test-Path $autounattend_template)) {
    throw "Failed to find '${autounattend_template}'. Builder cannot continue."
  }

  $xml = [xml](Get-Content $autounattend_template)

  $product_key = Get-MetadataValue -key 'product-key'
  if ($product_key.length -gt 20) {
    Write-Output "Adding product key $product_key to Autounattend.xml."
    $new_element = $xml.CreateElement('ProductKey')
    $new_element.set_InnerText($product_key)
    $xml.unattend.settings[1].component[0].AppendChild($new_element)
  }
  $xml.Save($autounattend_file)
  Write-Output "autounattend-template.xml has been updated and saved to $autounattend_file."

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
  Copy-Item "${script:components_dir}\SetupComplete.cmd" -Destination "${scripts}\SetupComplete.cmd"

  Write-Output 'Copying netkvmco.dll.'
  Copy-Item $script:driver_dir\netkvmco.dll D:\Windows\System32\netkvmco.dll


  # Add-WindowsDriver only works for Windows 10 x86
  if ($script:x86) {
    Write-Output 'Slipstreaming drivers using DISM'
    DISM /Image:D: /Add-Driver /Driver:$script:driver_dir
  }
  else {
    Write-Output 'Slipstreaming drivers using Add-WindowsDriver'
    Add-WindowsDriver -Path D:\ -Driver $script:driver_dir -Recurse -Verbose
  }

  Write-Output 'Done applying image.'

  if (-not (Test-Path D:\Windows)) {
    throw 'Windows not installed!'
  }

  if ($script:uefi) {
    Write-Output 'Setting up UEFI bootloader.'
    & bcdboot D:\Windows /s S: /f UEFI
    Set-Partition -DriveLetter S -GptType '{c12a7328-f81f-11d2-ba4b-00a0c93ec93b}'
  }
  else {
    Write-Output 'Setting up MBR bootloader.'
    & bcdboot D:\Windows /s D: /f BIOS
  }

  Write-Output 'Disabling startup animation.'
  # See http://support.microsoft.com/kb/2955372/en-us
  reg load HKLM\MountedSoftware D:\Windows\System32\config\SOFTWARE
  reg add HKLM\MountedSoftware\Microsoft\Windows\CurrentVersion\Authentication\LogonUI /v AnimationDisabled /t REG_DWORD /d 1 /f
  reg unload HKLM\MountedSoftware
}

try {
  # Setup directories to store files which are added to the sbom.
  $script:sbom_dir = 'C:\sbomcomponents'
  $script:components_dir = "$script:sbom_dir\components"
  $script:updates_dir = "$script:components_dir\updates"
  $script:driver_dir = "$script:sbom_dir\drivers"
  New-Item $script:sbom_dir -Type directory
  New-Item $script:updates_dir -Type directory
  New-Item $script:driver_dir -Type directory

  $script:uefi = (Get-MetadataValue -key 'uefi-build').ToLower() -eq 'true'
  $script:x86 = (Get-MetadataValue -key 'x86-build').ToLower() -eq 'true'

  Write-Output 'Boostrapping Windows install disk.'
  Download-Components
  Download-Drivers
  Download-Sbomutil
  if ($script:uefi) {
    Format-InstallDiskUEFI
  }
  else {
    Format-InstallDiskMBR
  }
  Bootstrap-InstallDisk

  $repo = Get-MetadataValue -key 'google-cloud-repo'
  Write-Output "Setting up GooGet repo $repo."
  & 'C:\ProgramData\GooGet\googet.exe' -root 'D:\ProgramData\GooGet' addrepo "google-compute-engine-${repo}" "https://packages.cloud.google.com/yuck/repos/google-compute-engine-${repo}"

  Generate-Sbom

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
