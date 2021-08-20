#  Copyright 2021 Google Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# Ensure the network interface is configured to use DHCP.
# This script assumes there is only one network adapter.

$adapter = Get-NetAdapter | ? {$_.Status -eq "up"}
$interface = $adapter | Get-NetIPInterface -AddressFamily IPv4

If ($interface.Dhcp -eq "Disabled") {
  Write-Host "Translate: $($interface.InterfaceAlias) does not have DHCP enabled. Enabling DHCP."
  # Remove existing gateway
  If (($interface | Get-NetIPConfiguration).Ipv4DefaultGateway) {
    Write-Host "Translate: Removing IPv4 default gateway for $($interface.InterfaceAlias)."
    $interface | Remove-NetRoute -Confirm:$false
  }

  # Enable DHCP
  Write-Host "Translate: Enabling DHCP for $($interface.InterfaceAlias)."
  $interface | Set-NetIPInterface -DHCP Enabled

  # Configure the DNS Servers automatically
  Write-Host "Translate: Configure the DNS to use DNS servers from DHCP for $($interface.InterfaceAlias)."
  $interface | Set-DnsClientServerAddress -ResetServerAddresses

  # Restart the network adapter
  Write-Host "Translate: Restarting network adapter: $($adapter.Name)."
  $adapter | Restart-NetAdapter

  # Give the network time to initialize.
  & ping 127.0.0.1 -n 30
} else {
  Write-Host "Translate: $($interface.InterfaceAlias) is configured for DHCP."
}

# Log DNS Server information
Write-Host 'DNS client configuration:'
$interface | Get-DnsClientServerAddress

# Test connection to packages.cloud.google.com and log results.
Write-Host 'Testing connection to packages.cloud.google.com:'
Test-NetConnection -ComputerName packages.cloud.google.com -Port 443
Test-NetConnection -ComputerName packages.cloud.google.com -Port 80
