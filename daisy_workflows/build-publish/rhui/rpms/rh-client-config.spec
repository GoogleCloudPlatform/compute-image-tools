Name:		%{_name}
Version:	4.0
Release:	1
Summary:	RHUI config for GCE

Group:		Applications/Internet
License:	GPLv2
URL:		  http://redhat.com
Source0:	%{name}-%{version}.tar.gz
BuildRoot:	%{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildArch:	noarch
Requires: yum
Requires: crontabs

%description
Red Hat Update Infrastructure for Google Compute Engine instances


%prep
%setup -q


%build


%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT

# Repo file and mirrorlist
mkdir -p $RPM_BUILD_ROOT/etc/yum.repos.d
cp $RPM_BUILD_DIR/%{name}-%{version}/rh-cloud.repo $RPM_BUILD_ROOT/etc/yum.repos.d

# GPG keys
%if 0%{?_gpg_keys:1}
    mkdir -p $RPM_BUILD_ROOT/etc/pki/rpm-gpg
    cp $RPM_BUILD_DIR/%{name}-%{version}/RPM-GPG-KEY* $RPM_BUILD_ROOT/etc/pki/rpm-gpg/
%endif

# Client entitlement cert
mkdir -p $RPM_BUILD_ROOT/etc/pki/rhui/product
cp $RPM_BUILD_DIR/%{name}-%{version}/content.crt $RPM_BUILD_ROOT/etc/pki/rhui/product
cp $RPM_BUILD_DIR/%{name}-%{version}/key.pem $RPM_BUILD_ROOT/etc/pki/rhui
# GCE doesn't use self-signed certs for the RHUI endpoint.
#cp $RPM_BUILD_DIR/%{name}-%{version}/ca.crt $RPM_BUILD_ROOT/etc/pki/rhui

# rhui-set-release tool
mkdir -p $RPM_BUILD_ROOT/%{_bindir}
cp $RPM_BUILD_DIR/%{name}-%{version}/rhui-set-release $RPM_BUILD_ROOT/%{_bindir}

# google-rhui-client-update cron job
cp $RPM_BUILD_DIR/%{name}-%{version}/google-rhui-client-package-update $RPM_BUILD_ROOT/etc/cron.daily

%post
if [ "$1" = "1" ]; then  # 'install', not 'upgrade'
  # Disable RHN and subscription-manager plugin
  for f in /etc/yum/pluginconf.d/{subscription-manager.conf,rhnplugin.conf}; do
    if [ -f $f ]; then
      grep -iPq "enabled\s*=\s*(0|false|off)" $f || sed -i.save -e 's/^enabled.*/enabled = 0/g' $f || :
    fi
  done
fi


%clean
rm -rf $RPM_BUILD_ROOT


%files
%defattr(-,root,root,-)
%doc
# This also comes from /etc/rhui/templates, from the rhui-tools rpm
%{_bindir}/rhui-set-release
%attr(600,root,root) %config  /etc/pki/rhui/product/content.crt
# GCE doesn't use self-signed certs for the RHUI endpoint.
#%attr(600,root,root) %config  /etc/pki/rhui/ca.crt
%attr(600,root,root) %config  /etc/pki/rhui/key.pem
%config  /etc/yum.repos.d/rh-cloud.repo
%if 0%{?_gpg_keys:1}
    %config  /etc/pki/rpm-gpg/RPM-GPG-KEY-*
%endif
%attr(755,root,root) %config  /etc/cron.daily/google-rhui-client-package-update

%changelog
* Fri Feb 10 2023 Google Compute Engine <images--associates@google.com>
- Updated client RPMs for RHUIv4 migration.
