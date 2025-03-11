{{{$version := printf "%s.%s.%s" .major .minor .patch }}}

%if 0%{?with_debug}
%global _dwz_low_mem_die_limit 0
%else
%global debug_package %{nil}
%endif

%global app_name                ovirt-csi-driver
%global app_version             {{{$version}}}
%global oracle_release_version  1
%global _buildhost              build-ol%{?oraclelinux}-%{?_arch}.oracle.com

Name:           %{app_name}
Version:        %{app_version}
Release:        %{oracle_release_version}%{?dist}
Summary:        CSI driver for oVirt
License:        Apache 2.0
Group:          Development/Tools
Url:            https://github.com/oracle-cne/ovirt-csi-driver.git
Source:         %{name}-%{version}.tar.bz2
BuildRequires:  golang
BuildRequires:	make

%description
CSI driver for oVirt

%prep
%setup -q

%build
make build

%install
install -m 755 bin/%{app_name} %{buildroot}/%{app_name}

%files
%license LICENSE
/%{app_name}

%changelog
* {{{.changelog_timestamp}}} - {{{$version}}}-1
- Added Oracle specific build files for oVirt CSI driver.
