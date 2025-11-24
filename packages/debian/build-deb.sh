#!/bin/bash
set -euo pipefail

# IMPORTANT: This script is intended to be called from a build workflow

# Debian revision: <upstream>-<debian-revision>
SRCDIR=$PWD

# === PREPARE WORKING DIR ===
BUILD_DIR="${BUILD_DIR:-$(mktemp -d)}"
mkdir -p "$BUILD_DIR" && cd "$BUILD_DIR"
mkdir -p DEBIAN usr/bin usr/share/fish/completions \
         usr/share/bash-completion/completions \
         usr/share/zsh/site-functions \
         usr/lib/systemd/user \
         usr/share/man/man1

# === COPY ARTIFACTS FROM BUILD DIR (MIMICS PKGBUILD) ===
cp -a $SRCDIR/build/gosuki $SRCDIR/build/suki ./usr/bin/
chmod 755 ./usr/bin/gosuki ./usr/bin/suki

# Completions
for completion_file in "$SRCDIR/contrib"/*-*.completions; do
    # Skip if no files match the pattern
    [[ -e "$completion_file" ]] || continue

    # Extract the completion type (e.g., fish, bash, zsh) from filename
    basename=$(basename "$completion_file")
    basename=${basename%.completions}
    type=${basename#*-}
    bin=${basename%-*}

    case "$type" in
        fish)
            cp -a "$completion_file" "usr/share/fish/completions/${bin}.fish"
            ;;
        bash)
            cp -a "$completion_file" "usr/share/bash-completion/completions/${bin}"
            ;;
        zsh)
            cp -a "$completion_file" "usr/share/zsh/site-functions/_${bin}"
            ;;
        *)
            echo "Warning: Unsupported completion type: $type"
            ;;
    esac
done

# Systemd service (Linux only)
if [ -f "$SRCDIR/contrib/linux/etc/systemd/user/${PKG_NAME}.service" ]; then
  cp $SRCDIR/contrib/linux/etc/systemd/user/${PKG_NAME}.service ./usr/lib/systemd/user/
fi

# Man pages
cp -a $SRCDIR/contrib/gosuki.1 usr/share/man/man1/
cp -a $SRCDIR/contrib/suki.1 usr/share/man/man1/

# Documentation (optional, but matches PKGBUILD)
mkdir -p usr/share/doc/${PKG_NAME}
for doc in README.md LICENSE; do
  [ -f "$SRCDIR/$doc" ] && cp "$SRCDIR/$doc" "usr/share/doc/${PKG_NAME}/"
done

# Additional scripts (from PKGBUILD)
mkdir -p usr/share/${PKG_NAME}/scripts
cp -a $SRCDIR/contrib/rofi.sh usr/share/${PKG_NAME}/scripts/
cp -a $SRCDIR/contrib/marktab/example.marktab usr/share/doc/${PKG_NAME}/
find $SRCDIR/contrib/marktab/scripts -type f | while read script; do
  cp "$script" "usr/share/${PKG_NAME}/scripts/$(basename "$script")"
done

# === CREATE DEBIAN CONTROL FILE ===
cat > DEBIAN/control << EOF
Package: ${PKG_NAME}
Version: ${DEB_VERSION}
Architecture: ${ARCH}
Maintainer: Chakib Benziane <contact@blob42.xyz>
Description: Multi-browser, real-time, extension-free bookmark manager with sync and archival
 A lightweight, privacy-first bookmark manager that unifies
 your bookmarks across multiple browsers, syncs them in real
 time (locally or P2P), requires no extensions,
 and stores everything locally.
Section: misc
Priority: optional
Homepage: https://gosuki.net
License: AGPL-3.0-or-later
EOF

# === BUILD DEB PACKAGE ===
dpkg-deb -b "$BUILD_DIR" "${PKG_NAME}_${DEB_VERSION}_${ARCH}.deb"

# === FINAL OUTPUT ===
DEBIAN_OUTPUT="$PWD/${PKG_NAME}_${DEB_VERSION}_${ARCH}.deb"
echo "Built: $DEBIAN_OUTPUT"
echo "deb-package=$DEBIAN_OUTPUT" >> "$GITHUB_OUTPUT"
echo "✅ Debian package created:"


