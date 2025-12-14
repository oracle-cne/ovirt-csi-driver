%if 0%{?with_debug}
%global _dwz_low_mem_die_limit 0
%else
%global debug_package %{nil}
%endif

%global app_name                ovirt-csi-driver
%global app_version             4.21.0
%global oracle_release_version  4
%global _buildhost              build-ol%{?oraclelinux}-%{?_arch}.oracle.com

Name:           %{app_name}-container-image
Version:        %{app_version}
Release:        %{oracle_release_version}%{?dist}
Summary:        CSI driver for oVirt
License:        Apache 2.0
Group:          Development/Tools
Url:            https://github.com/oracle-cne/ovirt-csi-driver.git
Source:         %{name}-%{version}.tar.bz2

%description
CSI driver for oVirt

%prep
%setup -q

%build
%global rpm_name %{app_name}-%{version}-%{release}.%{_build_arch}
%global docker_image container-registry.oracle.com/olcne/%{app_name}:v%{version}-2

yum clean all
yumdownloader --destdir=${PWD}/rpms %{rpm_name}
podman build --pull \
    --build-arg https_proxy=${https_proxy} \
    -t %{docker_image} -f ./olm/builds/Dockerfile .
podman save -o %{app_name}.tar %{docker_image}

%install
%__install -D -m 644 %{app_name}.tar %{buildroot}/usr/local/share/olcne/%{app_name}.tar

%files
%license LICENSE THIRD_PARTY_LICENSES.txt olm/SECURITY.md
/usr/local/share/olcne/%{app_name}.tar

%changelog
* Thu Dec 11 2025 Michael Gianatassio <michael.gianatassio@oracle.com> - 4.21.0-4
- Remove folder named "deploy" that contained obsolete helm templates.

* Thu Nov 6 2025 Michael Gianatassio <michael.gianatassio@oracle.com> - 4.21.0-3
- Resolve issue of burst request to create PVCs getting allocated across available storage domains

* Mon Oct 6 2025 Michael Gianatassio <michael.gianatassio@oracle.com> - 4.21.0-2
- Update version for next release
- Update base image to be OracleLinux:9-slim

* Mon Jul 21 2025 Paul Mackin <paul.mackin@oracle.com> - 4.21.0-1
- Update versions to 4.21.0 for disk_profile work for merging to master.

* Wed Jun 25 2025 Paul Mackin <paul.mackin@oracle.com> - 4.20.0-4
- Base64 encode password in config file.

* Tue Jun 24 2025 Paul Mackin <paul.mackin@oracle.com> - 4.20.0-3
- Delete config file containing password.

* Thu Mar 27 2025 Paul Mackin <paul.mackin@oracle.com> - 4.20.0-2
- Initial fixes for both controller plugin and node plugin.

* Wed Mar 12 2025 Michael Gianatassio <michael.gianatassio@oracle.com> - 4.20.0-1
- Added Oracle specific build files for oVirt CSI driver.
