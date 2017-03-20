SHELL = /bin/bash

TARGETS = microblob
PKGNAME = microblob

all: $(TARGETS)

$(TARGETS): %: cmd/%/main.go
	go get -v ./...
	go build -v -o $@ $<

clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)*.deb
	rm -f $(PKGNAME)-*.rpm
	rm -rf packaging/deb/$(PKGNAME)/usr

deb: $(TARGETS)
	mkdir -p packaging/deb/$(PKGNAME)/usr/sbin
	cp $(TARGETS) packaging/deb/$(PKGNAME)/usr/sbin
	# md2man-roff microblob.md > microblob.1
	mkdir -p packaging/deb/$(PKGNAME)/usr/local/share/man/man1
	cp docs/microblob.1 packaging/deb/$(PKGNAME)/usr/local/share/man/man1
	cd packaging/deb && fakeroot dpkg-deb --build $(PKGNAME) .
	mv packaging/deb/$(PKGNAME)_*.deb .

rpm: $(TARGETS)
	mkdir -p $(HOME)/rpmbuild/{BUILD,SOURCES,SPECS,RPMS}
	cp ./packaging/rpm/$(PKGNAME).spec $(HOME)/rpmbuild/SPECS
	cp $(TARGETS) $(HOME)/rpmbuild/BUILD
	cp docs/microblob.1 $(HOME)/rpmbuild/BUILD
	./packaging/rpm/buildrpm.sh $(PKGNAME)
	cp $(HOME)/rpmbuild/RPMS/x86_64/$(PKGNAME)*.rpm .
