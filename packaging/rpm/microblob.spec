Summary:    A simple key value store for JSON data.
Name:       microblob
Version:    0.2.1
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

mkdir -p $RPM_BUILD_ROOT/usr/local/share/man/man1
install -m 644 microblob.1 $RPM_BUILD_ROOT/usr/local/share/man/man1/microblob.1

%post

%clean
rm -rf $RPM_BUILD_ROOT
rm -rf %{_tmppath}/%{name}
rm -rf %{_topdir}/BUILD/%{name}

%files
%defattr(-,root,root)

/usr/local/sbin/microblob
/usr/local/share/man/man1/microblob.1

%changelog

* Fri Mar 3 2017 Martin Czygan
- 0.1.8 stats, logging, legacy route

* Thu Feb 23 2017 Martin Czygan
- 0.1.0 initial release
