#!/bin/sh
set -e

BINARY_NAME="fxtunnel"
INSTALL_DIR="$HOME/.local/bin"
BASE_URL="${FXTUNNEL_BASE_URL:-{{.BaseURL}}}"
WEBSITE_URL="${FXTUNNEL_WEBSITE_URL:-{{.WebsiteURL}}}"

main() {
    detect_os
    detect_arch
    check_dependencies

    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        CURRENT_VERSION=$("$BINARY_NAME" version 2>/dev/null || echo "unknown")
        CURRENT_PATH=$(command -v "$BINARY_NAME" 2>/dev/null || true)
        echo "fxtun is already installed (${CURRENT_VERSION})."
        if [ -n "$CURRENT_PATH" ] && [ "$(dirname "$CURRENT_PATH")" != "$INSTALL_DIR" ]; then
            echo "Note: existing binary is at ${CURRENT_PATH}"
            echo "New version will be installed to ${INSTALL_DIR}/${BINARY_NAME}"
            echo "You may want to remove the old binary: rm ${CURRENT_PATH}"
        fi
        echo "Reinstalling..."
    fi

    echo "Downloading fxtun for ${OS}/${ARCH}..."

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    DOWNLOAD_URL="${BASE_URL}/cli-${OS}-${ARCH}"
    TARGET="${TMP_DIR}/${BINARY_NAME}"

    download "$DOWNLOAD_URL" "$TARGET"

    chmod +x "$TARGET"

    mkdir -p "$INSTALL_DIR"

    echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    mv "$TARGET" "${INSTALL_DIR}/${BINARY_NAME}"
    ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/fxtun"
    echo "$WEBSITE_URL" > "${INSTALL_DIR}/.fxtunnel-website"

    ensure_path

    echo ""
    echo "fxtun installed successfully!"
    echo "Available as: fxtun, fxtunnel"
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

ensure_path() {
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) return ;;
    esac

    ADDED=0
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$rc" ]; then
            if ! grep -q '\.local/bin' "$rc" 2>/dev/null; then
                echo '' >> "$rc"
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$rc"
                ADDED=1
            fi
        fi
    done

    if [ "$ADDED" = "1" ]; then
        echo ""
        echo "Added ~/.local/bin to PATH. Restart your terminal or run:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    elif ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo ""
        echo "Warning: $INSTALL_DIR is not in your PATH."
        echo "Add this to your shell config:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
}

main
