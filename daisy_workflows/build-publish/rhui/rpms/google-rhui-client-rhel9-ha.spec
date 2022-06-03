Name:		google-rhui-client-rhel9-ha
Version:	4.0
Release:	1
Summary:	RHUI config for GCE (HA repo)

Group:		Applications/Internet
License:	GPLv2
URL:		  http://redhat.com
Source0:	%{name}-%{version}.tar.gz
BuildRoot:	%{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildArch:	noarch
Requires: yum
Requires: google-rhui-client-rhel9

%description
Red Hat Update Infrastructure for Google Compute Engine instances
High Availability repo only


%prep
%setup -q


%build


%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT

# Repo file and mirrorlist
mkdir -p $RPM_BUILD_ROOT/etc/yum.repos.d
cp $RPM_BUILD_DIR/%{name}-%{version}/rh-cloud-ha.repo $RPM_BUILD_ROOT/etc/yum.repos.d/


%clean
rm -rf $RPM_BUILD_ROOT


%files
%defattr(-,root,root,-)
%config  /etc/yum.repos.d/rh-cloud-ha.repo

%changelog
* Fri Jun 03 2022 Liam Hopkins <liamh@google.com>
- Created new HA RPM
