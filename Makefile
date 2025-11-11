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

# We only return the part inside the double quote here to avoid escape issues
# when calling the external release script. The second parameter can be used to
# add additional ldflags if needed (currently only used for the release).

VERSION := $(shell git describe --tags --dirty 2>/dev/null || echo "unknown")

make_ldflags = $(1) -X $(PKG)/pkg/build.Describe=$(VERSION)
#https://go.dev/doc/gdb
# disable gc optimizations
DEV_GCFLAGS := -gcflags "all=-N -l"
DEV_LDFLAGS := -ldflags "$(call make_ldflags)"

#TODO: add optimization flags
RELEASE_LDFLAGS := -ldflags "$(call make_ldflags, -s -w -buildid=)"

TAGS := $(OS) $(shell go env GOARCH)
ifdef SYSTRAY
    TAGS += systray
endif


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

.PHONY: build
build: $(foreach target,$(TARGETS),build/$(target))

build/%: $(SRC)
ifeq ($(OS), darwin)
	@ sed -i '' 's/LoggingMode = .*/LoggingMode = Dev/' pkg/logging/log.go
else
	@ sed -i 's/LoggingMode = .*/LoggingMode = Dev/' pkg/logging/log.go
endif
	$(call set_logging_mode)
	$(GOBUILD) -tags "$(TAGS)" -o build/$* $(DEV_GCFLAGS) $(DEV_LDFLAGS) ./cmd/$*


.PHONY: debug
debug: 
	@#dlv debug . -- server
	@# @go build -v $(DEV_GCFLAGS) -o build/gosuki ./cmd/gosuki
	dlv debug --headless --listen 127.0.0.1:38697 ./cmd/gosuki -- \
		-c /tmp/gosuki.conf.temp \
		--db=/tmp/gosuki.db.tmp start

.PHONY: release
release: 
ifeq ($(OS), darwin)
	@ sed -i '' 's/LoggingMode = .*/LoggingMode = Release/' pkg/logging/log.go
else
	@ sed -i 's/LoggingMode = .*/LoggingMode = Release/' pkg/logging/log.go
endif
	@$(call print, "Building release gosuki and suki.")
	$(GOBUILD) -tags "$(TAGS)" -o build/gosuki $(RELEASE_LDFLAGS) ./cmd/gosuki
	$(GOBUILD) -tags "$(TAGS)" -o build/suki   $(RELEASE_LDFLAGS) ./cmd/suki

.PHONY: docs
	@gomarkdoc -u ./... > docs/API.md


.PHONY: genimports
genimports: 
	@go generate ./...

# Distribution packaging
ARCH := x86_64

.PHONY: checksums
checksums:
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

# ifeq ($(OS), darwin)
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
	@echo '	<string>Copyright © 2023 Your Company. All rights reserved.</string>' >> build/gosuki.app/Contents/Info.plist
	@echo '</dict>' >> build/gosuki.app/Contents/Info.plist
	@echo '</plist>' >> build/gosuki.app/Contents/Info.plist

	# Add entitlements file 
	@cp ./assets/macos/Info.entitlements build/gosuki.app/Contents/
	@echo "App bundle created at build/gosuki.app"
# endif

.PHONY: completions
completions: $(COMPLETION_TARGETS)

contrib/%.completions:
	@echo $@
	$(eval bin=$(shell target='$*'; echo "$${target%-*}"))
	$(eval sh=$(shell target='$*'; echo "$${target#*-}"))
	@go run ./cmd/$(bin) -S completion $(sh) > $@

