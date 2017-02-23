SHELL = /bin/bash

TARGETS = microblob
PKGNAME = microblob

all: $(TARGETS)

$(TARGETS): %: cmd/%/main.go
	go get -v ./...
	go build -o $@ $<

clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)*.deb
	rm -f $(PKGNAME)-*.rpm
	rm -rf packaging/deb/$(PKGNAME)/usr

deb: $(TARGETS)
	mkdir -p packaging/deb/$(PKGNAME)/usr/sbin
	cp $(TARGETS) packaging/deb/$(PKGNAME)/usr/sbin
	cd packaging/deb && fakeroot dpkg-deb --build $(PKGNAME) .
	mv packaging/deb/$(PKGNAME)_*.deb .

rpm: $(TARGETS)
	mkdir -p $(HOME)/rpmbuild/{BUILD,SOURCES,SPECS,RPMS}
	cp ./packaging/rpm/$(PKGNAME).spec $(HOME)/rpmbuild/SPECS
	cp $(TARGETS) $(HOME)/rpmbuild/BUILD
	./packaging/rpm/buildrpm.sh $(PKGNAME)
	cp $(HOME)/rpmbuild/RPMS/x86_64/$(PKGNAME)*.rpm .

# ==== vm-based packaging ====
#
# Required, if development and deployment OS have different versions of libc.
# Examples: CentOS 6.5 has 2.12 (2010-08-03), Ubuntu 14.04 2.19 (2014-02-07).
#
# ----
#
# Initially, setup a CentOS 6.5 machine, install dependencies and git clone:
#
#     $ vagrant up
#
# To build an rpm, subsequently run:
#
#     $ make rpm-compatible
#
# If vagrant ssh runs on a port other than 2222, adjust (e.g. to port 2200):
#
#     $ make rpm-compatible PORT=2200
#
# A span-<version>-0.x86_64.rpm file should appear on your host machine, that
# has been built againts CentOS' 6.5 libc.
#
# Cleanup VM:
#
#     $ vagrant destroy --force

PORT = 2222
SSHCMD = ssh -o StrictHostKeyChecking=no -i vagrant.key vagrant@127.0.0.1 -p $(PORT)
SCPCMD = scp -o port=$(PORT) -o StrictHostKeyChecking=no -i vagrant.key

# Helper to build RPM on a RHEL6 VM, to link against glibc 2.12
vagrant.key:
	curl -sL "https://raw.githubusercontent.com/mitchellh/vagrant/master/keys/vagrant" > vagrant.key
	chmod 0600 vagrant.key

rpm-compatible: vagrant.key
	$(SSHCMD) "GOPATH=/home/vagrant go get -f -u github.com/jteeuwen/go-bindata/... golang.org/x/tools/cmd/goimports"
	$(SSHCMD) "cd /home/vagrant/src/github.com/miku/$(PKGNAME) && git pull origin master && pwd && GOPATH=/home/vagrant make clean && GOPATH=/home/vagrant make all rpm"
	$(SCPCMD) vagrant@127.0.0.1:/home/vagrant/src/github.com/miku/$(PKGNAME)/*rpm .
