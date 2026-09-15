#!/bin/bash
# ============================================
# echosystem site+server build script
# compiles the application for Linux AMD64
# ============================================
set -euo pipefail

echo "========================================="
echo "  echosystem site+server build script"
echo "========================================="
echo ""
echo "starting build process for linux/amd64..."
echo ""

# config
BINARY_NAME="echosystem"
TARGET_OS="linux"
TARGET_ARCH="amd64"
OUTPUT_DIR="./dist"
GIT_TAG=$(git describe --tags --always --dirty 2>/dev/null || echo "dev-$(date +%s)")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# pre-build checks
if ! command -v go &> /dev/null; then
    echo "[error] go compiler not found in PATH. please install Go."
    exit 1
fi

if [ ! -f "go.mod" ]; then
    echo "[error] go.mod not found. are you in the project root directory?"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"

# build process
FULL_OUTPUT="${OUTPUT_DIR}/${BINARY_NAME}"
LDFLAGS="-X echosystem/util.AppVersion=${GIT_TAG} -X echosystem/util.AppTime=${BUILD_TIME}"

echo "git tag/commit : ${GIT_TAG}"
echo "build time     : ${BUILD_TIME}"
echo "output binary  : ${FULL_OUTPUT}"
echo "-----------------------------------------"
echo ""
echo "running go build with verbose output (-v -x)..."
echo ""

if GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build \
    -v \
    -x \
    -ldflags "-s -w ${LDFLAGS}" \
    -trimpath \
    -o "$FULL_OUTPUT" .; then
    
    chmod +x "$FULL_OUTPUT"
    echo ""
    echo "-----------------------------------------"
    echo "[success] compilation finished successfully!"
    FILE_SIZE=$(du -h "$FULL_OUTPUT" | cut -f1)
    echo "[info] binary size: ${FILE_SIZE}"
    echo "[success] build artifact saved to '${OUTPUT_DIR}/'"
else
    echo ""
    echo "[error] compilation failed. check logs above."
    exit 1
fi
