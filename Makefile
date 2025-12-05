PKG := github.com/blob42/gosuki
CGO_ENABLED=1
CGO_CFLAGS="-g -Wno-return-local-addr"
SRC := **/*.go
GOBUILD := go build -v
GOINSTALL := go install -v
GOTEST := go test
OS := $(shell go env GOOS)
TARGETS := gosuki suki
COMPLETIONS := fish bash zsh
COMPLETION_TARGETS := $(foreach target,$(TARGETS),$(foreach type, $(COMPLETIONS), contrib/$(target)-$(type).completions))

VERSION := $(shell git describe --tags --dirty 2>/dev/null || echo "unknown")

make_ldflags = $(1) -X $(PKG)/pkg/build.Describe=$(VERSION)
#https://go.dev/doc/gdb
# disable gc optimizations
DEV_GCFLAGS := -gcflags "all=-N -l"
DEV_LDFLAGS = -ldflags "$(call make_ldflags)"

#TODO: add optimization flags
RELEASE_LDFLAGS := -ldflags "$(call make_ldflags, -s -w -buildid=)"

BUILD_FLAGS = $(DEV_GCFLAGS) $(DEV_LDFLAGS)

TAGS := $(OS) $(shell go env GOARCH)
ifdef SYSTRAY
    TAGS += systray
endif

ifdef CI
   TAGS += ci
endif

BROWSER_PLATFORMS := linux darwin
BROWSER_DEFS := $(foreach os,$(BROWSER_PLATFORMS),pkg/browsers/defined_browsers_$(os).go)

# TODO: remove, needed for testing mvsqlite
# SQLITE3_SHARED_TAGS := $(TAGS) libsqlite3

ifeq ($(origin TEST_FLAGS), environment)
	override TEST_FLAGS := $(TEST_FLAGS)
endif

# shared: TAGS = $(SQLITE3_SHARED_TAGS)

.PHONY: all
all: prepare build

.PHONY: prepare
prepare:
	@mkdir -p build


SED_IN_PLACE = sed -i 
ifeq ($(OS), darwin)
	SED_IN_PLACE = sed -i ''
endif

release_logging = $(SED_IN_PLACE) 's/LoggingMode = .*/LoggingMode = Release/' pkg/logging/log.go


.PHONY: build
build: sanitize $(foreach target,$(TARGETS),build/$(target)) 


.PHONY: sanitize
sanitize: 
	$(call release_logging)


build/%: $(BROWSER_DEFS) $(SRC)
	$(GOBUILD) -tags "$(TAGS)" -o build/$* $(BUILD_FLAGS) ./cmd/$*


.PHONY: release
release: BUILD_FLAGS = $(RELEASE_LDFLAGS)
release: build


.PHONY: debug
debug: 
	@#dlv debug . -- server
	@# @go build -v $(DEV_GCFLAGS) -o build/gosuki ./cmd/gosuki
	dlv debug --headless --listen 127.0.0.1:38697 ./cmd/gosuki -- \
		-c /tmp/gosuki.conf.temp \
		--db=/tmp/gosuki.db.tmp start


.PHONY: docs
	@gomarkdoc -u ./... > docs/API.md


# Generate everything
.PHONY: gen
gen: 
	@go generate ./...


$(BROWSER_DEFS) &:
	@go generate ./pkg/browsers


.PHONY: genmods
genmods: mods/generated_imports.go

MOD_ASSETS = $(shell find mods -type f -name '*.go')
mods/generated_imports.go: mods
	@go generate ./mods

# Distribution packaging
ARCH := x86_64

.PHONY: checksums
checksums:
	@[ -d dist ] || (echo run 'make dist' first && exit 10)
	cd dist && sha256sum *.tar.gz *.zip > SHA256SUMS
	rm -f dist/SHA256SUMS.sig
	gpg --detach-sign -u $(GPG_SIGN_KEY) dist/SHA256SUMS


.PHONY: testsum
testsum:
ifeq (, $(shell which gotestsum))
	$(GOINSTALL) gotest.tools/gotestsum@latest
endif
	gotestsum -f dots-v2 $(TEST_FLAGS) . ./...


.PHONY: ci-test
ci-test:
ifeq (, $(shell which gotestsum))
	$(GOINSTALL) gotest.tools/gotestsum@latest
endif
	gotestsum -f github-actions $(TEST_FLAGS) . ./...


.PHONY: test
test:
	go test -v ./...


.PHONY: clean
clean:
	rm -rf build dist
	rm -f contrib/*.completion
	rm -f **/**/defined_*.go


.PHONY: bundle-macos
bundle-macos: release
	@echo "Creating macOS app bundle..."
	@mkdir -p build/gosuki.app/Contents/{MacOS,Resources}
	@cp build/{gosuki,suki} build/gosuki.app/Contents/MacOS/
	@cp contrib/macos/launch.sh build/gosuki.app/Contents/MacOS/
	@chmod +x build/gosuki.app/Contents/MacOS/launch.sh
	@cp assets/icon/gosuki.icns build/gosuki.app/Contents/Resources/
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > build/gosuki.app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> build/gosuki.app/Contents/Info.plist
	@echo '<plist version="1.0">' >> build/gosuki.app/Contents/Info.plist
	@echo '<dict>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleDevelopmentRegion</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>en</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleExecutable</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>launch.sh</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleIdentifier</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>$(PKG)</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleIconFile</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>gosuki.icns</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleName</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>gosuki</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundlePackageVersion</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleShortVersionString</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>CFBundleVersion</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>LSApplicationCategoryType</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>com.apple.application-type.gui</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<key>NSHumanReadableCopyright</key>' >> build/gosuki.app/Contents/Info.plist
	@echo '	<string>Copyright © 2025 Chakib Benziane (contact@blob42.xyz). All rights reserved.</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '</dict>' >> build/gosuki.app/Contents/Info.plist
	@echo '</plist>' >> build/gosuki.app/Contents/Info.plist

	# Add entitlements file 
	@cp ./assets/macos/Info.entitlements build/gosuki.app/Contents/
	@echo "App bundle created at build/gosuki.app"

.PHONY: completions
completions: $(COMPLETION_TARGETS)

contrib/%.completions:
	@echo $@
	$(eval bin=$(shell target='$*'; echo "$${target%-*}"))
	$(eval type=$(shell target='$*'; echo "$${target#*-}"))
	@go run -tags ci ./cmd/$(bin) -S completion $(type) > $@
