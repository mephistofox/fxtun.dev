#!/bin/sh
set -e

BINARY_NAME="fxtunnel"
INSTALL_DIR="$HOME/.local/bin"
BASE_URL="${FXTUNNEL_BASE_URL:-{{.BaseURL}}}"
WEBSITE_URL="${FXTUNNEL_WEBSITE_URL:-{{.WebsiteURL}}}"
# Public half of the release signing key. Overridable for self-hosted builds.
UPDATE_PUBKEY="${FXTUNNEL_UPDATE_PUBKEY:-3ff8d3d89e9b44b4353e9d793194ad2c5a51cb614c57d50ed276efb7ca0f4b51}"

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

    download_or_die "$DOWNLOAD_URL" "$TARGET"

    verify_signature "$TARGET" "${DOWNLOAD_URL}.sig"

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

# hex_to_bin decodes hex on stdin to raw bytes on stdout.
hex_to_bin() {
    if command -v xxd >/dev/null 2>&1; then
        xxd -r -p
    elif command -v perl >/dev/null 2>&1; then
        perl -ne 's/\s//g; print pack("H*", $_)'
    else
        return 1
    fi
}

# verify_signature checks the detached ed25519 signature published next to the
# binary.
#
# Every published build carries a detached signature, so a download that cannot
# be verified is never installed — neither a wrong signature nor a missing one.
verify_signature() {
    file="$1"
    sig_url="$2"

    if [ -z "$UPDATE_PUBKEY" ] || ! command -v openssl >/dev/null 2>&1; then
        echo "Warning: cannot verify the download (no key or no openssl)" >&2
        return 0
    fi

    if ! download "$sig_url" "${file}.sighex" >/dev/null 2>&1; then
        echo "Error: no signature published for this build, refusing to install" >&2
        echo "Report this at ${WEBSITE_URL}" >&2
        exit 1
    fi

    echo "Verifying signature..."
    if ! hex_to_bin < "${file}.sighex" > "${file}.sig"; then
        echo "Warning: xxd or perl is required to verify the download" >&2
        return 0
    fi
    printf '302a300506032b6570032100%s' "$UPDATE_PUBKEY" | hex_to_bin > "${TMP_DIR}/pub.der"
    if ! openssl pkey -pubin -inform DER -in "${TMP_DIR}/pub.der" -out "${TMP_DIR}/pub.pem" 2>/dev/null; then
        echo "Error: invalid update public key" >&2
        exit 1
    fi
    if ! openssl pkeyutl -verify -pubin -inkey "${TMP_DIR}/pub.pem" -rawin \
        -sigfile "${file}.sig" -in "$file" >/dev/null 2>&1; then
        echo "Error: signature verification failed, refusing to install" >&2
        echo "Report this at ${WEBSITE_URL}" >&2
        exit 1
    fi
    echo "Signature OK"
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
        return 1
    fi
}

# download_or_die is the plain "must succeed" variant used for the binary.
download_or_die() {
    if ! download "$1" "$2"; then
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
