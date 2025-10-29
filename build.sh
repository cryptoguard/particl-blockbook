#!/bin/bash
# Build script for Blockbook with version information
#
# WARNING: This build may fail on some Linux systems due to filename length
# limitations in the gnark-crypto dependency. See DEPLOY_README.md for details.

set -e

cd "$(dirname "$0")"

# Version information
VERSION="v1.0.0-particl"
GITCOMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILDTIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "Building Blockbook for Particl..."
echo "  Version:   $VERSION"
echo "  Commit:    $GITCOMMIT"
echo "  BuildTime: $BUILDTIME"
echo ""
echo "NOTE: If build fails with 'file name too long' errors, see DEPLOY_README.md"
echo ""

# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin

# Try with short cache path workaround
export GOMODCACHE=/tmp/gomod
mkdir -p /tmp/gomod

# Build with version info embedded
go build \
  -ldflags="-X github.com/trezor/blockbook/common.version=$VERSION \
            -X github.com/trezor/blockbook/common.gitcommit=$GITCOMMIT \
            -X github.com/trezor/blockbook/common.buildtime=$BUILDTIME" \
  -o build/blockbook .

echo ""
echo "Build complete! Binary: build/blockbook"
echo ""
echo "To run:"
echo "./build/blockbook -blockchaincfg=particl-runtime.json -datadir=/tmp/blockbook-particl -sync -public=:9235 -internal=:9135 -logtostderr"
