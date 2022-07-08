<#
    .SYNOPSIS
        Common system configuration functions.
#>

$script:verbose = $true

function Invoke-ExternalCommand {
    param (
        [string] $cmd,
        [string] $errorStr,
        [Parameter(ValueFromRemainingArguments = $true)]$args
    )

    if ($script:verbose) {
        Write-LogInfo "Executing $cmd $args "
    }

    $output = & $cmd $args
    if (0 -ne $LastExitCode) {
        throw $errorStr + "Output: $output"
    }

    if ($script:verbose) {
        Write-LogInfo $output
    }
}

function Enable-EmsAccess {
    Write-LogInfo 'Enabling EMS access'

    $errorStr = 'failed to bcedit'
    Invoke-ExternalCommand -cmd 'bcdedit' -errorStr $errorStr /emssettings EMSPORT:2 EMSBAUDRATE:115200
    Invoke-ExternalCommand -cmd 'bcdedit' -errorStr $errorStr /ems on
}

function Config-NetSvc {
    $svchost_path = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Svchost'
    $netsvcs = (Get-ItemProperty -Path $svchost_path).netsvcs
    $netsvcs += 'sacsvr'
    Set-ItemProperty -Path $svchost_path -Name netsvcs -Value $netsvcs -Type MultiString
}

function Update-GooglePackages {
    Write-LogInfo 'Updating Google drivers and packages'
    $packages = @(
        'google-compute-engine-windows',
        'google-compute-engine-metadata-scripts',
        'google-compute-engine-driver-netkvm',
        'google-compute-engine-driver-pvpanic',
        'google-compute-engine-driver-vioscsi',
        'google-compute-engine-sysprep',
        'google-compute-engine-vss'
    )

    $packages | ForEach-Object {
        $output = & googet -noconfirm install $_
        if ($null -ne $output) {
            Write-LogInfo $output
        }
    }

}

function Convert-MinutestoMilliseconds {
    param (
        [Parameter(Mandatory = $True)] $timeInMilliseconds
    )

    return $timeInMilliseconds * 1000 * 60
}

function Config-Net {
    Write-LogInfo 'Restoring TCP timeout and'
    $TcpParams = 'HKLM:\System\CurrentControlSet\Services\Tcpip\Parameters'
    $timeInMilliseconds = Convert-MinutestoMilliseconds 5
    New-ItemProperty -Force -Path $TcpParams -Name 'KeepAliveTime' -Value $timeInMilliseconds -PropertyType DWord | Out-Null
}

function Update-WindowsLicense {
    Write-LogInfo 'Refreshing Windows license'
    Invoke-ExternalCommand -cmd "${env:ProgramFiles}\Google\Compute Engine\sysprep\activate_instance.ps1" -errorStr 'Failed to refresh Windows license'
}

function Config-Sys {
    Enable-EmsAccess

    Config-NetSvc

    Config-Net

    Update-WindowsLicense
}

function Config-Time {
    Write-LogInfo 'Strating W32time and resync'
    $output = $(Start-Service W32time)
    if ($null -ne $output) {
        Write-LogInfo $output
    }
    Invoke-ExternalCommand 'w32tm' -errorStr 'failed to start resync' /resync | Out-Default
}

function Test-UpgradeBlockingRegistryValues {
    $path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run'
    $value = Get-ItemProperty -Path $path
    if ( -not ([string]::IsNullOrEmpty($value)) ) {
        Write-LogWarning "Got blocking registry value, key: $path value: $value"
    }

    $path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\RunOnce'
    $value = Get-ItemProperty -Path $path
    if ( -not ([string]::IsNullOrEmpty($value)) ) {
        Write-LogWarning "Got blocking registry value, key: $path value: $value"
    }

    $path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending'
    if (Test-Path $path) {
        Log-IfUpgradeBlockingRegistryValueFound -path $path -out 'true'
    }

    $path = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired'
    if (Test-Path $path) {
        Log-IfUpgradeBlockingRegistryValueFound -path $path -out 'true'
    }

    $path = 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\'
    $value = Get-ItemProperty -Path $path -Name PendingFileRenameOperations -ErrorAction SilentlyContinue
    if ( -not ([string]::IsNullOrEmpty($value)) ) {
        Write-LogWarning "Got blocking registry value, key: $path\PendingFileRenameOperations value: true"
    }
}

function Restore-PreUpgradeConfiguration {
    Test-UpgradeBlockingRegistryValues

    Config-Sys

    Test-UpgradeBlockingRegistryValues
}

function Restore-PostUpgradeConfiguration {
    Config-Sys

    Config-Time

    & route add 169.254.169.254 mask 255.255.255.255 0.0.0.0 -p | Out-Default

    Update-GooglePackages
}

Export-ModuleMember -Function Restore-PreUpgradeConfiguration
Export-ModuleMember -Function Restore-PostUpgradeConfiguration
