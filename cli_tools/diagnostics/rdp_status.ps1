$RDPTCPpath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server\Winstations\RDP-Tcp'
$RDPServerPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'

$items= @(
  @{
    description="Verify that the Ethernet adapter is active, expect [Enabled]:";
    cmd="netsh interface show interface"
  },
  @{
    description="Verify that DHCP is enabled and IP configuration is correct, expect [DHCP Enabled: Yes]:";
    cmd="netsh interface ipv4 show addresses"
  },
  @{
    description="Check the Remote Desktop Service status, expect [Running]:";
    cmd="(Get-Service -Name TermService).Status"
  },
  @{
    description="Verify that Remote Desktop Session Host is enabled for multi-user connection, expect [installed]:";
    cmd="(Get-WindowsFeature -Name RDS-RD-Server).InstallState"
  },
  @{
    description="Check that Remote Connections are enabled, expect [fDenyTSConnections: 0]:";
    cmd="Write-Host $RDPServerPath'\fDenyTSConnections: '(Get-ItemProperty -Path '$RDPServerPath' -Name fDenyTSConnections -ErrorAction SilentlyContinue).fDenyTSConnections"
  },
  @{
    description="Ensure that the Windows firewall has Remote Desktop Connections enabled, expect [Enabled:Yes]:";
    cmd="netsh advfirewall firewall show rule name='Remote Desktop - User Mode (TCP-In)'"
  },
  @{
    description="Check what port number is configured for RDP connections, expect [default: 3389]:";
    cmd="Write-Host $RDPTCPpath'\PortNumber: '(Get-ItemProperty -Path '$RDPTCPpath' -Name PortNumber -ErrorAction SilentlyContinue).PortNumber"
  },
  @{
    description="Ensure that connected user account has permissions for remote connections, expect [target local/domain username in resulting list]:";
    cmd="net localgroup 'Remote Desktop Users'"
  },
  @{
    description="Verify that MTU size is no greater than 1460, expect [MTU <= 1460, Interface:Ethernet]:";
    cmd="netsh interface ipv4 show subinterfaces"
  },
  @{
    description="Verify that client-server seecurity negotiation is set to default value, expect [SecurityLayer REG_DWORD 0x0, 0x1, 0x2] depending on Security Layer configuration:";
    cmd="reg query 'HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp' /v SecurityLayer"
  },
  @{
    description="Verify that Network-level UserAuthentication is set to default value, expect [UserAuthentication REG_DWORD 0x0 or 0x1]:";
    cmd="reg query 'HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp' /v UserAuthentication"
  }
)

Write-Host "If you see any unexpected values, please go to troubleshooting guide:`nhttps://cloud.google.com/compute/docs/troubleshooting/troubleshooting-rdp`n"
foreach($item in $items){
  Write-Host $item["description"]
  Invoke-Expression $item["cmd"]
  Write-Host
}
