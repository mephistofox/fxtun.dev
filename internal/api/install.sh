#!/bin/sh
set -e

BINARY_NAME="fxtunnel"
INSTALL_DIR="/usr/local/bin"
BASE_URL="${FXTUNNEL_BASE_URL:-{{.BaseURL}}}"

main() {
    detect_os
    detect_arch
    check_dependencies

    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        CURRENT_VERSION=$("$BINARY_NAME" version 2>/dev/null || echo "unknown")
        echo "fxTunnel is already installed (${CURRENT_VERSION}). Reinstalling..."
    fi

    echo "Downloading fxTunnel for ${OS}/${ARCH}..."

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    DOWNLOAD_URL="${BASE_URL}/cli-${OS}-${ARCH}"
    TARGET="${TMP_DIR}/${BINARY_NAME}"

    download "$DOWNLOAD_URL" "$TARGET"

    chmod +x "$TARGET"

    echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TARGET" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$TARGET" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    echo "fxTunnel installed successfully!"
    "${INSTALL_DIR}/${BINARY_NAME}" version || true
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
        *)
            echo "Error: unsupported operating system '$(uname -s)'" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            echo "Error: unsupported architecture '$(uname -m)'" >&2
            exit 1
            ;;
    esac

    # Windows only supports amd64
    if [ "$OS" = "windows" ] && [ "$ARCH" != "amd64" ]; then
        echo "Error: Windows builds are only available for amd64" >&2
        exit 1
    fi
}

check_dependencies() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi
}

download() {
    url="$1"
    output="$2"

    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fSL --progress-bar -o "$output" "$url"
    else
        wget -q --show-progress -O "$output" "$url"
    fi

    if [ ! -f "$output" ] || [ ! -s "$output" ]; then
        echo "Error: download failed" >&2
        exit 1
    fi
}

main
