$gs_path = "gs://gce-windows-drivers-public/release/gvnic-gq"
$destination = "$env:TEMP\gvnic-gq"

Write-Output "Downloading drivers from $gs_path to $destination"
If (test-path -PathType container $destination) {
    Remove-Item -Path $destination -Recurse -Force
}
New-Item -ItemType Directory -Path $destination
& 'gsutil' cp "${gs_path}/*" $destination
Write-Output 'Driver download complete.'

Write-Output 'Removing all instances of gvnic driver'
Get-WindowsDriver -Path D:\ | ForEach-Object {
    if ($_.OriginalFileName -Match 'gvnic.inf') {
        Write-Output $_.OriginalFileName
        Remove-WindowsDriver -Path D:\ -Driver $_.OriginalFileName
    }
}

Write-Output 'Installing GVNIC GQ driver using Add-WindowsDriver'
Add-WindowsDriver -Path D:\ -Driver $destination -Recurse -Verbose

Stop-Computer -ComputerName localhost -Force