# build with the following command:
# rpmbuild -bb
%define debug_package %{nil}

Name:       dynamore-feature-extraction-runner
Version:    %{getenv:VERSION}
Release:    1%{?dist}
Summary:    A deamon for lauching dynamore feature-extraction jobs
License:    FIXME
URL: https://github.com/Donders-Institute/%{name}
Source0: https://github.com/Donders-Institute/%{name}/archive/%{version}.tar.gz

BuildArch: x86_64
BuildRequires: systemd

# defin the GOPATH that is created later within the extracted source code.
%define gopath %{_tmppath}/go.rpmbuild-%{name}-%{version}

%description
A deamon for lauching dynamore feature-extraction jobs.

%prep
%setup -q

%preun
if [ $1 -eq 0 ]; then
    echo "stopping service dfe_runnerd ..." 
    systemctl stop dfe_runnerd.service
    systemctl disable dfe_runnerd.service
fi

%build
# create GOPATH structure within the source code
rm -rf %{gopath}
mkdir -p %{gopath}/src/github.com/Donders-Institute
# copy entire directory into gopath, this duplicate the source code
cp -R %{_builddir}/%{name}-%{version} %{gopath}/src/github.com/Donders-Institute/%{name}
cd %{gopath}/src/github.com/Donders-Institute/%{name}; GOPATH=%{gopath} make; GOPATH=%{gopath} go clean --modcache

%install
mkdir -p %{buildroot}/%{_sbindir}
mkdir -p %{buildroot}/usr/lib/systemd/system
mkdir -p %{buildroot}/etc/sysconfig
## install the service binary
install -m 755 %{gopath}/bin/dynamore-feature-extraction-runner.linux_amd64 %{buildroot}/%{_sbindir}/dfe_runnerd
## install files for trqhelpd_srv service
install -m 644 scripts/dfe_runnerd.service %{buildroot}/usr/lib/systemd/system/dfe_runnerd.service
install -m 644 scripts/dfe_runnerd.env %{buildroot}/etc/sysconfig/dfe_runnerd

%files
%{_sbindir}/dfe_runnerd
/usr/lib/systemd/system/dfe_runnerd.service
/etc/sysconfig/dfe_runnerd

%post
echo "enabling service dfe_runnerd ..."
systemctl daemon-reload
systemctl enable dfe_runnerd.service
echo "starting service dfe_runnerd ..."
systemctl stop dfe_runnerd.service
systemctl start dfe_runnerd.service

%postun
if [ $1 -eq 0 ]; then
    systemctl daemon-reload
fi

%clean
rm -rf %{gopath}
rm -f %{_topdir}/SOURCES/%{version}.tar.gz
rm -rf $RPM_BUILD_ROOT

%changelog
* Fri Feb 05 2021 Hong Lee <h.lee@donders.ru.nl> - 0.1-1
- first implementation
