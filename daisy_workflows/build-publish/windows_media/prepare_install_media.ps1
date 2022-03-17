$ErrorActionPreference = 'Stop'

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

function Copy-FromGCS {
  param (
    [parameter(Mandatory=$true)]
      [string]$key,
    [parameter(Mandatory=$true)]
      [string]$dst
  )

  $gs_path = Get-MetadataValue -key $key
  if ($gs_path) {
    Write-Host "BuildStatus: Downloading from ${gs_path} to ${dst}."
    & 'gsutil' -m cp -r "${gs_path}/*" $dst
    Write-Host 'BuildStatus: Download complete.'
  }
  else {
    throw "Failed to find key ${key}. Builder cannot continue."
  }
}

function Format-MediaDisk {
  Write-Host 'BuildStatus: Formatting media disk.'
  Set-Disk -Number 1 -IsOffline $false
  Clear-Disk -Number 1 -RemoveData -Confirm:$false -ErrorAction SilentlyContinue
  Initialize-Disk -Number 1 -PartitionStyle GPT
  New-Partition -DiskNumber 1 -UseMaximumSize -DriveLetter D | Format-Volume -FileSystem 'NTFS' -Confirm:$false
  Write-Host 'BuildStatus: Formatting media disk complete.'
}

function Extract-Iso {
  param (
    [parameter(Mandatory=$true)]
      [string]$iso,
    [parameter(Mandatory=$true)]
      [string]$dst
  )

  Write-Host "BuildStatus: extracting iso ${iso} to ${dst}"
  $mount_result = Mount-DiskImage -ImagePath $iso -PassThru
  $iso_drive = ($mount_result | Get-Volume).DriveLetter
  Copy-Item "${iso_drive}:\*" $dst -Exclude @('autorun.inf', 'NanoServer') -Recurse -Force
  Write-Host "BuildStatus: finished extracting ${iso}"
}

$init_block = {
  function Update-Wim {
    param (
      [parameter(Mandatory=$true)]
        [string]$media_dir,
      [parameter(Mandatory=$true)]
        [string]$updates_dir
    )

    $name = Split-Path $media_dir -Leaf
    $wim = "${media_dir}\sources\install.wim"
    Set-ItemProperty -Path $wim -Name IsReadOnly -Value $false
    Write-Host "${name}: Modifying image ${wim}"
    Get-WindowsImage -Imagepath $wim | ForEach-Object {
      $image_name = $_.ImageName
      if ($image_name -notlike '*Datacenter*') {
        Write-Host "${name}: Removing ${image_name}"
        Remove-WindowsImage -ImagePath $wim -Name $image_name
      }
      else {
        $mount_path = 'c:\' + $name
        New-Item $mount_path -Type Directory
        Write-Host "${name}: Mounting ${wim} '${image_name}' at $mount_path"
        Mount-WindowsImage -ImagePath $wim -Name $image_name -Path $mount_path
        Add-Updates $name $updates_dir $mount_path
        Disable-StartupAnimation $name $mount_path

        Write-Host "${name}: Saving image ${wim}"
        Dismount-WindowsImage -Save -Path $mount_path
        Remove-Item $mount_path -Force -Recurse
      }
    }
    Set-ItemProperty -Path $wim -Name IsReadOnly -Value $true
  }

  function Add-Updates {
    param (
      [parameter(Mandatory=$true)]
        [string]$name,
      [parameter(Mandatory=$true)]
        [string]$updates_dir,
      [parameter(Mandatory=$true)]
        [string]$mount_path
    )
    Write-Host "${name}: Slipstreaming updates into image."
    Get-ChildItem $updates_dir | ForEach-Object {
      Write-Host "${name}: Adding '$($_.Name)' to image."
      $path = $_.FullName
      try {
        Add-WindowsPackage -Path $mount_path -PackagePath $path -ErrorAction Stop
        Write-Host "${name}: Successfully added $($_.Name)."
      }
      catch {
        Write-Host "${name}: Slipstreaming update failed, trying once more..."
        Add-WindowsPackage -Path $mount_path -PackagePath $path
      }
    }
  }

  function Disable-StartupAnimation {
    param (
      [parameter(Mandatory=$true)]
        [string]$name,
      [parameter(Mandatory=$true)]
        [string]$mount_path
    )
    Write-Host "${name}: Disabling startup animation."
    # See http://support.microsoft.com/kb/2955372/en-us
    $key = "HKLM\${name}"
    reg load $key "${mount_path}\Windows\System32\config\SOFTWARE"
    reg add "${key}\Microsoft\Windows\CurrentVersion\Authentication\LogonUI" /v AnimationDisabled /t REG_DWORD /d 1 /f
    reg unload $key
  }
}

try {
  # Setup builder directories.
  $builder_dir = 'C:\builder'
  $iso_dir = "$builder_dir\iso"
  $updates_dir = "$builder_dir\updates"
  $components_dir = "$builder_dir\components"
  New-Item $builder_dir -Type Directory
  New-Item $iso_dir -Type Directory
  New-Item $updates_dir -Type Directory
  New-Item $components_dir -Type Directory

  Copy-FromGCS 'updates-path' $updates_dir
  Copy-FromGCS 'iso-path' $iso_dir
  Copy-FromGCS 'components-path' $components_dir

  Format-MediaDisk

  # Copy common files.
  $dst = 'D:\common'
  New-Item $dst -Type Directory
  Copy-Item "${components_dir}\common\*" $dst -Recurse -ErrorAction Continue

  $jobs = @()
  Get-ChildItem $iso_dir | ForEach-Object {
    $name = ($_.Name).trimend('.iso')
    $dst = "D:\${name}"
    New-Item $dst -Type Directory
    Extract-Iso $_.Fullname $dst
    Copy-Item "${components_dir}\${name}\*" $dst -Recurse -ErrorAction Continue
    $script = {
      $args = @($input)[0]
      Update-Wim -media_dir $args[0] -updates_dir $args[1]
    }
    if (Test-Path "${updates_dir}\${name}") {
      $job = Start-Job -ScriptBlock $script -InitializationScript $init_block -InputObject @($dst, "${updates_dir}\${name}")
      $jobs += $job.Id
    }
  }

  Write-Host 'BuildStatus: Waiting for jobs to complete'
  Receive-Job -Id $jobs -Wait | ForEach-Object {
    Write-Host "BuildStatus: $_"
  }

  Write-Host 'BuildCompleted'
}
catch {
  Write-Host 'Exception caught in script:'
  Write-Host $_.InvocationInfo.PositionMessage
  Write-Host "BuildFailed: $($_.Exception.Message)"
  exit 1
}
