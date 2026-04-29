<#
    .SYNOPSIS
        Common logging functions.
#>

$script:port = New-Object System.IO.Ports.SerialPort COM3,115200,None,8,one
$script:bOpenedPort = $false
$script:setupDirPath = "${env:systemdrive}\Windows.setup"
$script:setupLogPath = "$script:setupDirPath\setup.log"

function Open-LogPort {
    if (-not $script:bOpenedPort) {
        $script:port.open()
        $script:bOpenedPort = $true
    }
}

function Close-LogPort {
    if ($bOpenedPort) {
        $script:port.close()
        $script:bOpenedPort = $false
    }
}

function Format-LogMsg {
    param (
        [Parameter(Mandatory = $True)] [ValidateNotNull()] $str
    )

    $formatted = $($str | Out-String).TrimEnd()
    $logMsg = "$((Get-Date).ToUniversalTime().ToString("yyyy/MM/dd HH:mm:ss.fffZ")) $formatted"
    return $logMsg
}

function Log-FormattedMsg {
    param (
        [Parameter(Mandatory = $True)] [ValidateNotNull()] $msg
    )
    Open-LogPort
    $script:port.WriteLine($msg)
    Close-LogPort

    # Create log dir if it doesnt exist
    if ( -not (Test-Path $script:setupLogPath) ) {
        New-Item -ItemType Directory -Force -Path $script:setupDirPath | Out-Null
        New-Item -ItemType File -Force -Path $script:setupLogPath | Out-Null
    }

    Add-Content $script:setupLogPath $msg
}

function Write-LogInfo {
    param (
        [Parameter(Mandatory = $True)] [ValidateNotNull()] $str
    )

    $formattedMsg = Format-LogMsg "Replatform Info: $str."
    Log-FormattedMsg $formattedMsg

    Write-Host $formattedMsg
}

function Write-LogWarning {
    param (
        [Parameter(Mandatory = $True)] [ValidateNotNull()] $str
    )

    $formattedMsg = Format-LogMsg "Replatform Warning: $str."
    Log-FormattedMsg $formattedMsg

    Write-Host $formattedMsg -ForegroundColor Yellow
}

function Write-LogError {
    param (
        [Parameter(Mandatory = $True)] [ValidateNotNull()] $str
    )

    $formattedMsg = Format-LogMsg "Replatform Error: $str."
    Log-FormattedMsg $formattedMsg

    Write-Host $formattedMsg -ForegroundColor Red
}

Export-ModuleMember -Function Write-LogInfo
Export-ModuleMember -Function Write-LogWarning
Export-ModuleMember -Function Write-LogError
Export-ModuleMember -Function Close-LogPort
