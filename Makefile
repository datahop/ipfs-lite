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

patch-mobile:
	@make patch-release-mobile
	@make build-mobile

minor-mobile:
	@make minor-release-mobile
	@make build-mobile

major-mobile:
	@make major-release-mobile
	@make build-mobile

build-cli: VERSION := $(shell . $(SUPPORT); getCliVersion)
build-cli:
	@cd ./cli && GOOS=linux GOARCH=amd64 go build -ldflags "\
		-X '$(VERSION_PACKAGE).CliVersion=$(VERSION)'" -o datahop-cli-linux-v$(VERSION) datahop.go
	@cd ./cli && GOOS=darwin GOARCH=amd64 go build -ldflags "\
    	-X '$(VERSION_PACKAGE).CliVersion=$(VERSION)'" -o datahop-cli-darwin-v$(VERSION) datahop.go
	@cd ./cli && GOOS=windows GOARCH=amd64 go build -ldflags "\
		-X '$(VERSION_PACKAGE).CliVersion=$(VERSION)'" -o datahop-cli-windows-v$(VERSION).exe datahop.go

patch-release-cli: VERSION := $(shell . $(SUPPORT); nextCliPatchVersion)
patch-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

minor-release-cli: VERSION := $(shell . $(SUPPORT); nextCliMinorVersion)
minor-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

major-release-cli: VERSION := $(shell . $(SUPPORT); nextCliMajorVersion)
major-release-cli:
	@. $(SUPPORT) ; setCliVersion $(VERSION)

patch-cli:
	@make patch-release-cli
	@make build-cli

minor-cli:
	@make minor-release-cli
	@make build-cli

major-cli:
	@make major-release-cli
	@make build-cli

run-tests:
	go test -json -p 1 -race -v ./...