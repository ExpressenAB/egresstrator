Name:           egresstrator
Version:        %{_version}
Release:        1
Summary:        Docker iptables egress orchestrator.
Group:          System Environment/Daemons
License:        Apache Software License
URL:            https://github.com/ExpressenAB/egresstrator
Source0:        %{name}
Source1:        %{name}.service
Source2:        %{name}.sysconfig
BuildRoot:      %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)

%description
Egresstrator builds iptables egress rules defined in Consul for containers running in Docker.

%install
mkdir -p %{buildroot}/%{_sbindir}
cp %{SOURCE0} %{buildroot}/%{_sbindir}/%{name}

mkdir -p %{buildroot}/%{_sysconfdir}/sysconfig
cp %{SOURCE2} %{buildroot}/%{_sysconfdir}/sysconfig/%{name}


mkdir -p %{buildroot}/%{_unitdir}
cp %{SOURCE1} %{buildroot}/%{_unitdir}/

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}
%{_unitdir}/%{name}.service
%attr(755, root, root) %{_sbindir}/*

%doc


%changelog
* Fri Dec 2 2016 Magnus Bengtsson <magnus.bengtsson@expressen.se>
- Release 0.0.2
* Thu Dec 1 2016 Magnus Bengtsson <magnus.bengtsson@expressen.se>
- Initial release 0.0.1
