SHELL = /bin/bash

TARGETS = microblob
PKGNAME = microblob
VERSION = 0.3.0

all: $(TARGETS)

$(TARGETS): %: cmd/%/main.go
	CGO_ENABLED=0 go build -ldflags="-s -w" -v -o $@ $<

.PHONY: clean
clean:
	rm -f $(TARGETS)
	rm -f $(PKGNAME)*.deb
	rm -f $(PKGNAME)-*.rpm

.PHONY: test
test:
	go test -v ./...

.PHONY: deb
deb: $(TARGETS)
	GOARCH=amd64 SEMVER=$(VERSION) nfpm package --config nfpm.yaml -p deb

.PHONY: rpm
rpm: $(TARGETS)
	GOARCH=amd64 SEMVER=$(VERSION) nfpm package --config nfpm.yaml -p rpm
