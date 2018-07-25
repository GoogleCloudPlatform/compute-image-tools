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

# Force the dist to be el7 to avoid el7.centos.
%if 0%{?rhel} == 7
  %define dist .el7
%endif

Name: google-compute-engine-inventory
Version: 1.1.0
Release: 1%{?dist}
Summary: Google Compute Engine inventory agent
License: ASL 2.0
Url: https://github.com/GoogleCloudPlatform/compute-image-tools

%if 0%{?el7}
BuildRequires: systemd
%endif

%if 0%{?el7}
Requires: systemd
%endif

%description
Google Compute Engine inventory agent

%prep
cp %{_topdir}/google-compute-engine-inventory.service %{_topdir}/google-compute-engine-inventory.conf %{_topdir}/BUILD

%build
CGO_ENABLED=0 GOPATH=%{_topdir}/BUILD go get -ldflags="-s -w" -v github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_inventory_agent/...

%install
mkdir -p %{buildroot}%{_bindir}
cp bin/gce_inventory_agent %{buildroot}%{_bindir}

%if 0%{?el6}
cp google-compute-engine-inventory.conf %{buildroot}/etc/init/
%endif

%if 0%{?el7}
mkdir -p %{buildroot}%{_unitdir}
cp google-compute-engine-inventory.service %{buildroot}%{_unitdir}
%endif

%files
%if 0%{?el6}
/etc/init/google-compute-engine-inventory.conf
%endif
%if 0%{?el7}
%{_unitdir}/google-compute-engine-inventory.service
%endif

%attr(0755,root,root) %{_bindir}/gce_inventory_agent

%post
%if 0%{?el6}
if [ $1 -eq 2 ]; then
  stop -q -n google-compute-engine-inventory
  start -q -n google-compute-engine-inventory
fi
%endif
%if 0%{?el7}
%systemd_post google-compute-engine-inventory.service
if [ $1 -eq 2 ]; then
  systemctl reload-or-restart google-compute-engine-inventory.service
fi
%endif


%preun
if [ $1 -eq 0 ]; then
%if 0%{?el6}
  stop -q -n google-compute-engine-inventory
%endif
%if 0%{?el7}
  %systemd_preun google-compute-engine-inventory.service
%endif
fi
