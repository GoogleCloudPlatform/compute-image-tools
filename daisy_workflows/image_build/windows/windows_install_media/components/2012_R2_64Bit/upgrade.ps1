<#
    .SYNOPSIS
        Apply GCE-specific settings and upgrade drivers. This script should be run:

        (1) Before an in-place upgrade to ensure critical settings are applied and
            drivers are up to date.
        (2) After an in-place upgrade to re-apply critical settings which might have
            been lost or overwritten during the upgrade.

    .PARAMETER UnattendPath
        Custom unattend.xml file to use.
#>
[CmdletBinding()]
  param (
    [parameter(Mandatory = $False)]
    [String]$UnattendPath,

    [parameter(Mandatory = $False)]
    [String]$SetupExePath
  )

$script:ACCEPTABLE_WIN_VERSION_MAJOR = 6
$script:UPGRADABLE_WIN_VERSION_MINOR = 1
$script:ACCEPTABLE_WIN_VERSION_MINORS= 1,3

$script:currLocation = Split-Path -parent $MyInvocation.MyCommand.Definition
$script:winVersion = $null

Import-Module -Name $script:currLocation\..\common\config.psm1
Import-Module -Name $script:currLocation\..\common\logging.psm1

function Exit-IfWinVersionIsntAcceptable {
    if ( $ACCEPTABLE_WIN_VERSION_MAJOR -ne $script:winVersion.major -or (-not $script:ACCEPTABLE_WIN_VERSION_MINORS -contains $script:winVersion.minor) ) {
        $script:ACCEPTABLE_WIN_VERSION_MINORS | ForEach-Object { $supportableVersionsStr += " $ACCEPTABLE_WIN_VERSION_MAJOR.$_" }
        $errorStr =    'This version of Windows ($script:winVersion) can not be upgraded.' +
            " Supported versions: $supportableVersionsStr"
        throw $errorStr
    }
}

function Get-UnattendFilePath {
    #Supported core SKUs:
    #12    Datacenter Server Core Edition, 13    Standard Server Core Edition, 14    Enterprise Server Core Edition
    $core_skus = 12,13,14
    $sku = (Get-WmiObject Win32_OperatingSystem).OperatingSystemSKU

    if ( $core_skus -contains $sku) {
        $unattendFileName = 'upgrade_unattend_core.xml'
    }
    else {
        $unattendFileName = 'upgrade_unattend.xml'
    }
    return "${script:currLocation}\unattend\${unattendFileName}"
}

$errorCode = 0

try {

    $script:winVersion = [System.Environment]::OSVersion.Version

    if ( $UPGRADABLE_WIN_VERSION_MINOR -eq $script:winVersion.minor ) {
        Write-LogInfo 'Starting pre-upgrade script'
    }
    else {
        Write-LogInfo 'Starting post-upgrade script'
    }

    Exit-IfWinVersionIsntAcceptable

    if (!$UnattendPath) {
        $UnattendPath = Get-UnattendFilePath
    }

    if (!$SetupExePath) {
        $SetupExePath = "${script:currLocation}\setup.exe"
    }

    if ( -not (Test-Path $SetupExePath -PathType Leaf) ) {
        throw "$SetupExePath doesn't exist"
    }

    if ( $UPGRADABLE_WIN_VERSION_MINOR -eq $script:winVersion.minor ) {
        Write-LogInfo 'Starting pre-upgrade script'
        Restore-PreUpgradeConfiguration

        Write-LogInfo 'Starting Windows upgrade process (Setup.exe)'
        & $SetupExePath /unattend:$UnattendPath /EMSPort:COM2 /emsbaudrate:115200

        Write-LogInfo 'Starting Logging of Windows upgrade process (Setup.exe)'
        & ${script:currLocation}\..\common\setup-logger.ps1
    }
    else {
        # Windows 2012 is running with post installation configuration
        Restore-PostUpgradeConfiguration
        Write-LogInfo 'Finished post-upgrade configurations successfully'
    }
}
catch {
    Write-LogError $_
    $errorCode = 1
    Close-LogPort
}

exit $errorCode
