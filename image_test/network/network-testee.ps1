# Copyright 2018 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Signalize wait-for-instance that instance is ready
Write-Host "BOOTED"

# Enable serving the port 80 for http server
netsh advfirewall firewall add rule name="http" protocol=TCP dir=in localport=80 action=allow

# Run the webserver that prints the hostname and os
$hostname = [System.Text.Encoding]::UTF8.GetBytes($(hostname))
$os = [System.Text.Encoding]::UTF8.GetBytes("windows")
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://+:80/")
Write-Host "Listening on port 80"
$listener.Start()
while ($listener.IsListening) {
    $context = $listener.GetContext()
    Write-Host "Received", $context.Request.RawUrl, "request from", $context.Request.RemoteEndPoint
    $buffer = switch ($context.Request.RawUrl) {
        "/os"       { $os }
        "/hostname" { $hostname}
        default     { continue }
    }
    $context.Response.ContentLength64 = $buffer.Length
    $context.Response.OutputStream.Write($buffer, 0, $buffer.Length)
    $context.Response.Close()
}
