<#
    .SYNOPSIS
        Apply GCE-specific settings and upgrade drivers. This script should be run:

        (1) Before an in-place upgrade to ensure critical settings are applied and
            drivers are up to date.
        (2) After an in-place upgrade to re-apply critical settings which might have
            been lost or overwritten during the upgrade.

#>
[CmdletBinding()]
  param (
    [parameter(Mandatory = $False)]
    [String]$ProductKey = 'D764K-2NDRG-47T6Q-P8T8W-YP6DF',

    [parameter(Mandatory = $False)]
    [String]$SetupExePath
  )

function Find-ImageIndex {
    param (
        [parameter(Mandatory=$true)]
        [string[]]$image_names,

        [parameter(Mandatory=$true)]
        [PSObject[]]$image_list
    )

    foreach ($image in $images) {
        foreach ($image_name in $image_names) {
            if ($image.ImageName -eq $image_name) {
                return $image.ImageIndex
            }
        }
    }
    $msg = 'No image is found from the installation media: '+$image_names[0]
    throw $msg
}

$script:UPGRADABLE_WIN_VERSION_MIN = New-Object -TypeName System.Version -ArgumentList 6,2,0,0           # Win 2012
$script:UPGRADABLE_WIN_VERSION_MAX = New-Object -TypeName System.Version -ArgumentList 10,0,26099,0  # The last 2022 build is not known yet

$script:ACCEPTABLE_WIN_VERSIONS_MIN = New-Object -TypeName System.Version -ArgumentList 10,0,26100,0   # Win 2025
$script:ACCEPTABLE_WIN_VERSIONS_MAX = New-Object -TypeName System.Version -ArgumentList 10,0,65535,65535 # The last 2025 build is not known yet

$script:currLocation = Split-Path -parent $MyInvocation.MyCommand.Definition

Import-Module -Name $script:currLocation\..\common\config.psm1
Import-Module -Name $script:currLocation\..\common\logging.psm1

$errorCode = 0

try {
    $script:winVersion = [System.Environment]::OSVersion.Version
    $script:installationType = Get-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows NT\CurrentVersion' -Name 'InstallationType'
    $images = Get-WindowsImage -ImagePath ./sources/install.wim
    if ($script:installationType.InstallationType -eq 'Server Core') {
        $script:imageIndex = Find-ImageIndex -image_name $('Windows Server 2025 SERVERDATACENTERCORE', 'Windows Server 2025 Datacenter') -image_list $images
    }
    elseif ($script:installationType.InstallationType -eq 'Server') {
        $script:imageIndex = Find-ImageIndex -image_name $('Windows Server 2025 SERVERDATACENTER', 'Windows Server 2025 Datacenter (Desktop Experience)') -image_list $images
    }
    else {
        $msg = 'Unexpected installation type is detected: '+$script:installationType.InstallationType
        throw $msg
    }
    $msg = $script:installationType.InstallationType+' installation type is chosen.'
    Write-LogInfo $msg

    $script:winVersion = [System.Environment]::OSVersion.Version

    if (!$SetupExePath) {
        $SetupExePath = "${script:currLocation}\setup.exe"
    }

    if ( -not (Test-Path $SetupExePath -PathType Leaf) ) {
        throw "$SetupExePath doesn't exist"
    }

    $setupArgs = @(
      '/imageindex', $script:imageIndex,
      '/auto', 'upgrade',
      '/eula', 'accept',
      '/telemetry', 'disable',
      '/pkey', $ProductKey,
      '/Compat', 'IgnoreWarning',
      '/noreboot'
    )

    if (([System.Environment]::OSVersion.Version -ge $script:UPGRADABLE_WIN_VERSION_MIN) -and
        ([System.Environment]::OSVersion.Version -le $script:UPGRADABLE_WIN_VERSION_MAX)) {
        # Running upgradable OS version.

        # Start setup.exe but don't wait for it to complete. We will poll for the BCD entry.
        $process = Start-Process -FilePath $SetupExePath -ArgumentList $setupArgs -PassThru
        Write-LogInfo "Waiting for 'Windows Setup' BCD entry to be created by setup.exe (PID: $($process.Id))"

        $timeout = New-TimeSpan -Minutes 60
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

        while ($stopwatch.Elapsed -lt $timeout) {
          $bcdOutput = (bcdedit /enum all | Out-String) -split '(?ms)^Windows Boot Loader\s*--+\s*'
          $setupEntry = $bcdOutput | Where-Object { $_ -match 'description\s+Windows Setup' }

          if ($setupEntry) {
            break
          }
          if (-not (Get-Process -Id $process.Id -ErrorAction SilentlyContinue)) {
            throw "setup.exe process (PID: $($process.Id)) exited unexpectedly before creating the BCD entry."
          }
          Start-Sleep -Seconds 60
        }
        $stopwatch.Stop()
        if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) {
          Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
    elseif (([System.Environment]::OSVersion.Version -ge $script:ACCEPTABLE_WIN_VERSIONS_MIN) -and
          ([System.Environment]::OSVersion.Version -le $script:ACCEPTABLE_WIN_VERSIONS_MAX)) {
        # Running acceptable OS version.

        Write-LogInfo 'Starting post-upgrade script'
        Restore-PostUpgradeConfiguration

        Write-LogInfo 'Finished post-upgrade configurations successfully'
    }
    else {
        # Running some unsupported version.
        throw "This version of Windows ($script:winVersion) can not be upgraded to Windows 2025"
    }
}
catch {
    Write-LogError $_
    $errorCode = 1
    Close-LogPort
}

Write-LogInfo 'Done with the Windows upgrade process (Setup.exe)'

try {
    # Step 2. Identify the Boot Sequence.
    Write-LogInfo 'Reading the Windows Boot Manager (bootmgr) configuration...'
    $bootmgrConfig = bcdedit.exe /enum '{bootmgr}'

    # Find the 'bootsequence' identifier using a regular expression.
    $regex = 'bootsequence\s+({[a-fA-F0-9\-]+})'
    $match = $bootmgrConfig | Select-String -Pattern $regex

    # Check if the 'bootsequence' identifier was found.
    if ($match) {
        # Extract the captured GUID from the match.
        $identifier = $match.Matches[0].Groups[1].Value
        Write-LogInfo "Successfully found the Windows Setup identifier: $identifier"

        # Step 3. Modify the BCD Entry.
        # 0x15000075 is an enum that tells winload.efi to use a previously calculated TSC frequency
        # value to bypass the current bug in the winload.efi file.
        Write-LogInfo "Executing: bcdedit.exe /set $identifier allowedinmemorysettings 0x15000075"
        bcdedit.exe /set $identifier allowedinmemorysettings 0x15000075

        Write-LogInfo 'BCD entry updated successfully.'

        Write-LogInfo 'Rebooting the VM to complete the upgrade'
        shutdown /r
    }
    else {
        # If no 'bootsequence' is found, the one-time boot task is likely not set.
        Write-Warning "Could not find a 'bootsequence' entry in the Windows Boot Manager."
    }
}
catch {
  # Catch and display any errors that occur during execution.
  Write-Error "An unexpected error occurred: $_"
}

exit $errorCode
