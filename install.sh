#!/bin/bash
# sysc-walls one-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Nomadcxx/sysc-walls/master/install.sh | sudo bash

set -e

echo "sysc-walls installer"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    echo "Usage: curl -fsSL https://raw.githubusercontent.com/Nomadcxx/sysc-walls/master/install.sh | sudo bash"
    exit 1
fi

# Check for Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    echo "Install Go first: https://go.dev/doc/install"
    exit 1
fi

REPO=https://github.com/Nomadcxx/sysc-walls.git

# Build a released tag, not whatever master holds at this moment. Set
# SYSC_WALLS_REF to install a specific tag.
REF="${SYSC_WALLS_REF:-}"
if [ -z "$REF" ]; then
    REF=$(git ls-remote --tags --refs "$REPO" 'v*' | sed 's|.*refs/tags/||' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1)
fi
if [ -z "$REF" ]; then
    echo "Error: could not find a sysc-walls release tag"
    exit 1
fi

# A private (0700) directory only root can enter, removed on any exit.
TEMP_DIR=$(mktemp -d)
trap 'cd / && rm -rf "$TEMP_DIR"' EXIT
cd "$TEMP_DIR"

echo "Cloning sysc-walls $REF..."
git -c advice.detachedHead=false clone --depth 1 --branch "$REF" "$REPO" sysc-walls
cd sysc-walls

echo "Building installer..."
go build -o install-sysc-walls ./cmd/installer/

echo "Running installer..."
./install-sysc-walls < /dev/tty

echo ""
echo "Installation complete!"
