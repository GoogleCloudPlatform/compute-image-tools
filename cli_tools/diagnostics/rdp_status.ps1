$RDPTCPpath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server\Winstations\RDP-Tcp'
$RDPServerPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server'

$items= @(
  @{
    description="Check the Remote Desktop Service status, expect [Running]:";
    cmd="(Get-Service -Name TermService).Status"
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
  }
)

Write-Host "If you see any unexpected values, please go to troubleshooting guide:`nhttps://cloud.google.com/compute/docs/troubleshooting/troubleshooting-rdp`n"
foreach($item in $items){
  Write-Host $item["description"]
  Invoke-Expression $item["cmd"]
  Write-Host
}