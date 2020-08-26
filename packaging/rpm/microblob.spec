Summary:    A simple key value store for JSON data.
Name:       microblob
Version:    0.2.11
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

if [ -d "/usr/local/share/microblob" ]; then
    rm -rf /usr/local/share/microblob
fi

%install

mkdir -p $RPM_BUILD_ROOT/usr/local/bin
install -m 755 microblob $RPM_BUILD_ROOT/usr/local/bin

mkdir -p $RPM_BUILD_ROOT/usr/local/share/man/man1
install -m 644 microblob.1.gz $RPM_BUILD_ROOT/usr/local/share/man/man1/microblob.1.gz

mkdir -p $RPM_BUILD_ROOT/usr/lib/systemd/system
install -m 755 microblob.service $RPM_BUILD_ROOT/usr/lib/systemd/system

mkdir -p $RPM_BUILD_ROOT/usr/local/share/microblob
install -m 755 hello.ndjson $RPM_BUILD_ROOT/usr/local/share/microblob

mkdir -p $RPM_BUILD_ROOT/etc/microblob
install -m 755 microblob.ini $RPM_BUILD_ROOT/etc/microblob

%post

chown -R daemon.daemon /usr/local/share/microblob

%clean
rm -rf $RPM_BUILD_ROOT
rm -rf %{_tmppath}/%{name}
rm -rf %{_topdir}/BUILD/%{name}

%files
%defattr(-,root,root)

/etc/microblob/microblob.ini
/usr/lib/systemd/system/microblob.service
/usr/local/bin/microblob
/usr/local/share/man/man1/microblob.1.gz
/usr/local/share/microblob/hello.ndjson

%changelog

* Fri Mar 3 2017 Martin Czygan
- 0.1.8 stats, logging, legacy route

* Thu Feb 23 2017 Martin Czygan
- 0.1.0 initial release
