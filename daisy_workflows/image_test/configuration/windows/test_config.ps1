function Test-Activation {
}

function Test-MTU {}

function Test-PowershellVersion {}

function Test-DotNetVersion {}

function Test-NTP {}

funcion Test-EMSEnabled {}

try {
  Test-MTU
  Test-PowershellVersion
  Test-DotNetVersion
  Test-NTP
  Test-EMSEnabled
  Test-Activation
catch {
  Write-Host TestFailed
  exit 1
}

Write-Host TestSuccess
