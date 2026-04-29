$script:port = $null

function Write-StreamLog {
    param (
        [string]$log,
        [String]$setupActIdentifier,
        [hashtable]$lastMaxOffsets,
        [hashtable]$readers
    )

    try {
        #case where the setup log dir isnt found, will fail
        # creating a reader cause the path deosnt exist
        if ( -not $(Test-Path -Path $(Split-Path $log -Parent)) ) {
            continue
        }

        $readers[$log] = New-Object System.IO.StreamReader(New-Object IO.FileStream($log,
            [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read,
            [IO.FileShare]::ReadWrite))
        if ( -not ($lastMaxOffsets.ContainsKey($log)) ) {
            $script:port.WriteLine("Tracking $setupLogs[$log]")
        }
        else {
            $readers[$log].BaseStream.Position = $lastMaxOffsets[$log]
        }
        while ( $null -ne ($line = $readers[$log].ReadLine()) ) {
            $script:port.WriteLine($setupActIdentifier + ': ' + $line)
        }

        # Update the last max offset
        $lastMaxOffsets[$log] = $readers[$log].BaseStream.Position
        $readers[$log].Close()

    }
    catch {
        $lastMaxOffsets.remove($log)
    }
}

function Write-StreamLogs {
    param (
        [string[]]$logs,
        [String[]]$setupActIdentifiers
    )

    $lastMaxOffsets= @{}
    $readers = @{}

    while ($true) {
        Start-Sleep -Milliseconds 100

        for ($i=0; $i -lt $logs.count; ++$i) {
            Write-StreamLog -log $logs[$i] -setupActIdentifier $setupActIdentifiers[$i] -lastMaxOffsets $lastMaxOffsets -readers $readers
        }
    }
}

$script:port = New-Object System.IO.Ports.SerialPort COM3,115200,None,8,one
$script:port.open()

try {
    $winBTPath = "${env:systemdrive}\`$WINDOWS.~BT\Sources\Panther"
    $setupLogs = @( "${env:windir}\setupact.log", "$env:windir\setuperr.log",
    "$winBTPath\setupact.log", "$winBTPath\setuperr.log")
    $setupActIdentifiers= @( '$WINDIR setupact$', '$WINDIR setuperr$',
    '$WINDOWS.~BT setupact$', '$WINDOWS.~BT setuperr$')

    Write-StreamLogs -logs $setupLogs -setupActIdentifiers $setupActIdentifiers
} finally {
    $script:port.close()
}

