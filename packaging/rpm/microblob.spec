Summary:    A simple key value store for JSON data.
Name:       microblob
Version:    0.1.6
Release:    0
License:    GPL
BuildArch:  x86_64
BuildRoot:  %{_tmppath}/%{name}-build
Group:      System/Base
Vendor:     Leipzig University Library, https://www.ub.uni-leipzig.de
URL:        https://github.com/miku/microblob

%description

A simple key value store for JSON data.

%prep

%build

%pre

%install
mkdir -p $RPM_BUILD_ROOT/usr/local/sbin
install -m 755 microblob $RPM_BUILD_ROOT/usr/local/sbin

%post

%clean
rm -rf $RPM_BUILD_ROOT
rm -rf %{_tmppath}/%{name}
rm -rf %{_topdir}/BUILD/%{name}

%files
%defattr(-,root,root)

/usr/local/sbin/microblob

%changelog
* Thu Feb 23 2016 Martin Czygan
- 0.1.0 initial release
