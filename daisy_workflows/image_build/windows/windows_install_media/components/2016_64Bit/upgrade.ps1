<#
    .SYNOPSIS
        Apply GCE-specific settings and upgrade drivers. This script should be run:

        (1) Before an in-place upgrade to ensure critical settings are applied and
            drivers are up to date.
        (2) After an in-place upgrade to re-apply critical settings which might have
            been lost or overwritten during the upgrade.

    .PARAMETER unattendPath
        Custom unattend.xml file to use.
#>
[CmdletBinding()]
  param (
    [parameter(Mandatory = $False)]
    [String]$ProductKey = 'CB7KF-BWN84-R7R2Y-793K2-8XDDG',

    [parameter(Mandatory = $False)]
    [String]$SetupExePath
  )

function Find-ImageIndex {
    param (
        [parameter(Mandatory=$true)]
        [string]$image_name
    )

    $images = Get-WindowsImage -ImagePath ./sources/install.wim
    foreach ($image in $images) {
        if ($image.ImageName -eq $image_name) {
            return $image.ImageIndex
        }
    }
    $msg = 'No image is found from the installation media: '+$image_name
    throw $msg
}

$script:UPGRADABLE_WIN_VERSION_MIN = New-Object -TypeName System.Version -ArgumentList 6,2,0,0           # Win 2012
$script:UPGRADABLE_WIN_VERSION_MAX = New-Object -TypeName System.Version -ArgumentList 6,3,65535,65535   # Win 2012 R2

$script:ACCEPTABLE_WIN_VERSIONS_MIN = New-Object -TypeName System.Version -ArgumentList 10,0,0,0         # Win 2016
$script:ACCEPTABLE_WIN_VERSIONS_MAX = New-Object -TypeName System.Version -ArgumentList 10,0,14393,65535 # Win 2016, final build

$script:currLocation = Split-Path -parent $MyInvocation.MyCommand.Definition

Import-Module -Name $script:currLocation\..\common\config.psm1
Import-Module -Name $script:currLocation\..\common\logging.psm1

$errorCode = 0

try {
    $script:winVersion = [System.Environment]::OSVersion.Version
    $script:installationType = Get-ItemProperty -Path 'HKLM:\Software\Microsoft\Windows NT\CurrentVersion' -Name 'InstallationType'
    if ($script:installationType.InstallationType -eq 'Server Core') {
        $script:imageIndex = Find-ImageIndex -image_name $('Windows Server 2016 SERVERDATACENTERCORE', 'Windows Server 2016 Datacenter')
    }
    elseif ($script:installationType.InstallationType -eq 'Server') {
        $script:imageIndex = Find-ImageIndex -image_name $('Windows Server 2016 SERVERDATACENTER', 'Windows Server 2016 Datacenter (Desktop Experience)')
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

    if (([System.Environment]::OSVersion.Version -ge $script:UPGRADABLE_WIN_VERSION_MIN) -and
        ([System.Environment]::OSVersion.Version -le $script:UPGRADABLE_WIN_VERSION_MAX)) {
        # Running upgradable OS version.

        Write-LogInfo 'Starting pre-upgrade script'
        Restore-PreUpgradeConfiguration

        Write-LogInfo 'Starting Windows upgrade process (Setup.exe)'
        & $SetupExePath /imageindex $script:imageIndex /auto upgrade /telemetry disable /pkey $ProductKey  /Compat IgnoreWarning

        Write-LogInfo 'Starting Logging of Windows upgrade process (Setup.exe)'
        & ${script:currLocation}\..\common\setup-logger.ps1
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
        throw "This version of Windows ($script:winVersion) can not be upgraded to Windows 2016"
    }
}
catch {
    Write-LogError $_
    $errorCode = 1
    Close-LogPort
}

exit $errorCode
