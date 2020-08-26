NAME = microblob
VERSION = 0.2.11
RELEASE = 0

$(NAME)-$(VERSION)-$(RELEASE)-x86_64.pkg.tar.xz: PKGBUILD
	makepkg

# https://github.com/golang/go/issues/28137
# https://github.com/golang/go/issues/27161#issuecomment-415213240
clean:
	rm -f $(NAME)-$(VERSION)-$(RELEASE)-x86_64.pkg.tar.xz
	rm -f v$(VERSION).tar.gz
	rm -rf pkg/
	rm -rf src/
