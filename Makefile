PKG := github.com/blob42/gosuki
CGO_ENABLED=1
CGO_CFLAGS="-g -Wno-return-local-addr"
SRC := $(shell git ls-files | grep '.*\.go$$' | grep -v 'log\.go')
GOBUILD := go build -v
GOINSTALL := go install -v
GOTEST := go test
OS := $(shell go env GOOS)
TARGETS := gosuki suki
COMPLETIONS := fish bash zsh
COMPLETION_TARGETS := $(foreach target,$(TARGETS),$(foreach type, $(COMPLETIONS), contrib/$(target)-$(type).completions))

VERSION := $(shell git describe --tags --dirty 2>/dev/null || echo "unknown")

ifdef NODIRTY
	VERSION := $(shell git describe --tags --no-abbrev 2>/dev/null || echo "unknown")
endif

# Cross-compilation support via Zig (for CGO targets: go-sqlite3)
ZIG    := $(shell which zig 2>/dev/null)
ZIG_OK := $(if $(ZIG),ok,missing)

# Zig installation helper
# Fetches latest version from https://ziglang.org/download/index.json
ZIG_LATEST_VERSION = $(shell curl -sf https://ziglang.org/download/index.json | jq -r 'keys | map(select(. != "master")) | sort_by(split(".") | map(tonumber)) | last')
ZIG_INSTALL_DIR    := $(HOME)/.local/share/zig
ZIG_BIN_DIR        := $(ZIG_INSTALL_DIR)/latest
ZIG_ARCH           := x86_64

ifeq ($(ZIG_OK),missing)
$(info WARNING: zig not found in PATH. Cross-compile targets require zig.)
endif

#.PHONY: install-zig-linux install-zig-windows install-zig-macos
# Pattern target: install-zig-{linux,windows,macos}
# Downloads latest stable Zig release and installs to $(ZIG_INSTALL_DIR)
# Usage: make install-zig-linux
install-zig-%:
	@echo "==> Installing Zig for $*"
	@ZIG_VER=$(ZIG_LATEST_VERSION); \
	if [ -z "$$ZIG_VER" ]; then echo "ERROR: Could not fetch latest Zig version from https://ziglang.org/download/index.json"; exit 1; fi; \
	echo "==> Latest Zig version: $$ZIG_VER"; \
	mkdir -p $(ZIG_INSTALL_DIR); \
	if [ "$*" = "windows" ]; then \
		URL="https://ziglang.org/download/$$ZIG_VER/zig-$(ZIG_ARCH)-windows-$$ZIG_VER.zip"; \
		EXT="zip"; \
	else \
		URL="https://ziglang.org/download/$$ZIG_VER/zig-$(ZIG_ARCH)-$*-$$ZIG_VER.tar.xz"; \
		EXT="tar.xz"; \
	fi; \
	echo "==> Downloading from: $$URL"; \
	echo "==> Downloading to: /tmp/zig-$$ZIG_VER.$$EXT"; \
	curl -C - -L -o /tmp/zig-$$ZIG_VER.$$EXT "$$URL"; \
	if [ "$$EXT" = "zip" ]; then \
		unzip -o /tmp/zig-$$ZIG_VER.$$EXT -d /tmp/ >/dev/null; \
	else \
		tar -xf /tmp/zig-$$ZIG_VER.$$EXT -C /tmp/ >?dev/null; \
	fi; \
	ZIG_DIR="zig-$(ZIG_ARCH)-$*-$$ZIG_VER"; \
	if [ -d "$(ZIG_INSTALL_DIR)/$$ZIG_VER" ]; then \
		echo "==> Zig $$ZIG_VER already installed, skipping"; \
	else \
		mv /tmp/$$ZIG_DIR $(ZIG_INSTALL_DIR)/$$ZIG_VER; \
		ln -sfn $(ZIG_INSTALL_DIR)/$$ZIG_VER $(ZIG_BIN_DIR); \
		echo "==> Installed to $(ZIG_INSTALL_DIR)/$$ZIG_VER"; \
	fi; \
	
	@echo "==> Add to PATH: export PATH=$(ZIG_BIN_DIR):\$${PATH}"


make_ldflags = $(1) -X $(PKG)/pkg/build.Describe=$(VERSION)
#https://go.dev/doc/gdb
# disable gc optimizations
DEV_GCFLAGS := -gcflags "all=-N -l"
DEV_LDFLAGS = -ldflags "$(call make_ldflags)"

#TODO: add optimization flags
RELEASE_LDFLAGS := -ldflags "$(call make_ldflags, -s -w -buildid=)"

BUILD_FLAGS = $(DEV_GCFLAGS) $(DEV_LDFLAGS)

TARGET_TAGS := $(shell go env GOOS) $(shell go env GOARCH)
TAGS:=
ifdef SYSTRAY
    TAGS += systray
endif

TEST_TAGS=
ifdef INTEGRATION
	TEST_TAGS += integration
endif

ifdef RELEASE
    TAGS += release
    BUILD_FLAGS = $(RELEASE_LDFLAGS)
endif

BROWSER_PLATFORMS := linux darwin freebsd netbsd openbsd windows
BROWSER_DEFS := $(foreach os,$(BROWSER_PLATFORMS),pkg/browsers/defined_browsers_$(os).go)

# TODO: remove, needed for testing mvsqlite
# SQLITE3_SHARED_TAGS := $(TAGS) libsqlite3

ifeq ($(origin EXTRA_TEST_FLAGS), environment)
	TEST_FLAGS += $(EXTRA_TEST_FLAGS)
endif

# shared: TAGS = $(SQLITE3_SHARED_TAGS)

.PHONY: all
all: prepare build

.PHONY: prepare
prepare:
	@mkdir -p build


.PHONY: build
build: $(foreach target,$(TARGETS),build/$(target))


build/%: $(BROWSER_DEFS) $(SRC)
	$(GOBUILD) -tags "$(TAGS) $(TARGET_TAGS)" -o build/$* $(BUILD_FLAGS) ./cmd/$*

.PHONY: release
release: TAGS += release
release: BUILD_FLAGS = $(RELEASE_LDFLAGS)
release: build

# Cross-compilation targets (requires zig in PATH)
# Uses per-target $(eval export ...) to scope env vars and prevent leakage.
.PHONY: cross-windows-amd64 cross-windows-386 cross-linux-386 cross-linux-arm64 cross-all


cross-windows-amd64: $(BROWSER_DEFS)
	$(eval export CC=zig cc -target x86_64-windows-gnu)
	$(eval export CGO_ENABLED=1)
	$(eval export GOOS=windows)
	$(eval export GOARCH=amd64)

	@echo "==> Building gosuki.exe (windows/amd64)"
	$(GOBUILD) -tags "$(TAGS)" -o build/gosuki.exe $(BUILD_FLAGS) ./cmd/gosuki/

	@echo "==> Building suki.exe (windows/amd64)"
	$(GOBUILD) -tags "$(TAGS)" -o build/suki.exe $(BUILD_FLAGS) ./cmd/suki/

cross-windows-386: $(BROWSER_DEFS)
	$(eval export CC=zig cc -target x86-windows-gnu)
	$(eval export CGO_ENABLED=1)
	$(eval export GOOS=windows)
	$(eval export GOARCH=386)

	@echo "==> Building gosuki-386.exe (windows/386)"
	$(GOBUILD) -tags "$(TAGS)" -o build/gosuki-386.exe $(BUILD_FLAGS) ./cmd/gosuki/
	@echo "==> Building suki-386.exe (windows/386)"
	$(GOBUILD) -tags "$(TAGS)" -o build/suki-386.exe $(BUILD_FLAGS) ./cmd/suki/

cross-linux-386: $(BROWSER_DEFS)
	$(eval export CC=zig cc -target x86-linux-gnu)
	$(eval export CGO_ENABLED=1)
	$(eval export GOOS=linux)
	$(eval export GOARCH=386)

	@echo "==> Building gosuki-386 (linux/386)"
	$(GOBUILD) -tags "$(TAGS)" -o build/gosuki-386 $(BUILD_FLAGS) ./cmd/gosuki/
	@echo "==> Building suki-386 (linux/386)"
	$(GOBUILD) -tags "$(TAGS)" -o build/suki-386 $(BUILD_FLAGS) ./cmd/suki/

cross-linux-arm64: $(BROWSER_DEFS)
	$(eval export CC=zig cc -target aarch64-linux-gnu)
	$(eval export CGO_ENABLED=1)
	$(eval export GOOS=linux)
	$(eval export GOARCH=arm64)

	@echo "==> Building gosuki-arm64 (linux/arm64)"
	$(GOBUILD) -tags "$(TAGS)" -o build/gosuki-arm64 $(BUILD_FLAGS) ./cmd/gosuki/
	@echo "==> Building suki-arm64 (linux/arm64)"
	$(GOBUILD) -tags "$(TAGS)" -o build/suki-arm64 $(BUILD_FLAGS) ./cmd/suki/

cross-all: cross-windows-amd64 cross-windows-386 cross-linux-386 cross-linux-arm64


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
ARCH := $(shell go env GOARCH)

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
	gotestsum -f dots-v2 -- -tags "$(TEST_TAGS)" $(TEST_FLAGS) . ./...


.PHONY: ci-test
ci-test:
ifeq (, $(shell which gotestsum))
	$(GOINSTALL) gotest.tools/gotestsum@latest
endif
	gotestsum -f github-actions -- -tags integration $(TEST_FLAGS) . ./...


.PHONY: test
test:
	go test -v ./...

.PHONY: cover
cover:
	@echo "==> Test coverage by package"
	@GOSUKI_CLEAN_FILES=false go test -cover ./... 2>&1 | grep -E "^ok |^FAIL " | \
		awk '{pkg=$$2; cov=""; for(i=1;i<=NF;i++) if($$i=="coverage:") cov=$$(i+1); if(pkg) printf "  %-50s %s\n", pkg, cov}'
	@echo ""
	@echo "==> Function coverage: queries.go"
	@GOSUKI_CLEAN_FILES=false go test -coverprofile=/tmp/gosuki.cov ./internal/database/ >/dev/null 2>&1 && \
		go tool cover -func=/tmp/gosuki.cov 2>/dev/null | grep "queries\.go"
	@echo "==> Function coverage: api.go"
	@GOSUKI_CLEAN_FILES=false go test -coverprofile=/tmp/gosuki.cov ./internal/api/ >/dev/null 2>&1 && \
		go tool cover -func=/tmp/gosuki.cov 2>/dev/null | grep "api\.go"
	@echo "==> Function coverage: commands.go"
	@GOSUKI_CLEAN_FILES=false go test -coverprofile=/tmp/gosuki.cov ./cmd/suki/ >/dev/null 2>&1 && \
		go tool cover -func=/tmp/gosuki.cov 2>/dev/null | grep "command"
	@rm -f /tmp/gosuki.cov


.PHONY: clean
clean:
	rm -rf build dist
	rm -f contrib/*.completion
	rm -f **/**/defined_*.go
	rm -f __debug_bin*


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
