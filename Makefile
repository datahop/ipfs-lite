SUPPORT := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))/.release
VERSION_PACKAGE := github.com/datahop/ipfs-lite/version

build-mobile: VERSION := $(shell . $(SUPPORT); getMobileVersion)
build-mobile:
	@gomobile bind -o ./mobile/datahop.aar -target=android -ldflags "\
	-X '$(VERSION_PACKAGE).MobileVersion=$(VERSION)'" \
	github.com/datahop/ipfs-lite/mobile

patch-release-mobile: VERSION := $(shell . $(SUPPORT); nextMobilePatchVersion)
patch-release-mobile:
	@. $(SUPPORT) ; setMobileVersion $(VERSION)

minor-release-mobile: VERSION := $(shell . $(SUPPORT); nextMobileMinorVersion)
minor-release-mobile:
	@. $(SUPPORT) ; setMobileVersion $(VERSION)

major-release-mobile: VERSION := $(shell . $(SUPPORT); nextMobileMajorVersion)
major-release-mobile:
	@. $(SUPPORT) ; setMobileVersion $(VERSION)

patch-mobile: patch-release-mobile build-mobile

minor-mobile: minor-release-mobile build-mobile

major-mobile: major-release-mobile build-mobile


build-cli: VERSION := $(shell . $(SUPPORT); getCliVersion)
build-cli:
	cd ./cli && go build -o datahop-cli datahop.go

patch-release-cli: VERSION := $(shell . $(SUPPORT); nextCliPatchVersion)
patch-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

minor-release-cli: VERSION := $(shell . $(SUPPORT); nextCliMinorVersion)
minor-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

major-release-cli: VERSION := $(shell . $(SUPPORT); nextCliMajorVersion)
major-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

patch-cli: patch-release-cli build-cli

minor-cli: minor-release-cli build-cli

major-cli: major-release-cli build-cli
