#!/bin/bash
#
# kindplane installer script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash
#
# Options:
#   KINDPLANE_VERSION  - Version to install (default: latest)
#   KINDPLANE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#
# Examples:
#   # Install latest version
#   curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash
#
#   # Install specific version
#   curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | KINDPLANE_VERSION=v0.1.0 bash
#
#   # Install to custom directory
#   curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | KINDPLANE_INSTALL_DIR=~/.local/bin bash
#

set -euo pipefail

# Configuration
VERSION="${KINDPLANE_VERSION:-latest}"
INSTALL_DIR="${KINDPLANE_INSTALL_DIR:-/usr/local/bin}"
GITHUB_REPO="kanzifucius/kindplane"
BINARY_NAME="kindplane"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Print functions
print_banner() {
    echo -e "${PURPLE}"
    echo '   _    _           _       _                  '
    echo '  | | _(_)_ __   __| |_ __ | | __ _ _ __   ___ '
    echo '  | |/ / | '\''_ \ / _'\'' | '\''_ \| |/ _'\'' | '\''_ \ / _ \'
    echo '  |   <| | | | | (_| | |_) | | (_| | | | |  __/'
    echo '  |_|\_\_|_| |_|\__,_| .__/|_|\__,_|_| |_|\___|'
    echo '                     |_|                       '
    echo -e "${NC}"
    echo -e "${CYAN}Bootstrap Kind clusters with Crossplane${NC}"
    echo ""
}

info() {
    echo -e "${BLUE}${BOLD}INFO${NC} $1"
}

success() {
    echo -e "${GREEN}${BOLD}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}${BOLD}!${NC} $1"
}

error() {
    echo -e "${RED}${BOLD}✗${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)          error "Unsupported operating system: $(uname -s)" ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}-${arch}"
}

# Get the latest release version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    
    if command -v curl &> /dev/null; then
        curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget &> /dev/null; then
        wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Download the binary
download_binary() {
    local version="$1"
    local platform="$2"
    local tmp_dir="$3"
    
    local ext=""
    local archive_ext="tar.gz"
    if [[ "$platform" == windows-* ]]; then
        ext=".exe"
        archive_ext="zip"
    fi

    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${BINARY_NAME}-${platform}-${version}.${archive_ext}"
    local archive_file="${tmp_dir}/${BINARY_NAME}.${archive_ext}"

    info "Downloading kindplane ${version} for ${platform}..."
    
    if command -v curl &> /dev/null; then
        if ! curl -fsSL -o "$archive_file" "$download_url"; then
            # Try without version suffix (for older releases)
            download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${BINARY_NAME}-${platform}.${archive_ext}"
            curl -fsSL -o "$archive_file" "$download_url" || error "Failed to download from: $download_url"
        fi
    elif command -v wget &> /dev/null; then
        if ! wget -q -O "$archive_file" "$download_url"; then
            download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${BINARY_NAME}-${platform}.${archive_ext}"
            wget -q -O "$archive_file" "$download_url" || error "Failed to download from: $download_url"
        fi
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    # Extract the archive
    info "Extracting archive..."
    cd "$tmp_dir"
    
    if [[ "$archive_ext" == "tar.gz" ]]; then
        tar -xzf "$archive_file"
    else
        unzip -q "$archive_file"
    fi

    # Find the binary
    local binary_file="${BINARY_NAME}-${platform}${ext}"
    if [[ ! -f "$binary_file" ]]; then
        # Try to find it
        binary_file=$(find . -name "${BINARY_NAME}*" -type f ! -name "*.tar.gz" ! -name "*.zip" | head -1)
    fi

    if [[ ! -f "$binary_file" ]]; then
        error "Could not find binary after extraction"
    fi

    echo "$binary_file"
}

# Install the binary
install_binary() {
    local binary_path="$1"
    local install_dir="$2"
    local binary_name="$3"

    # Create install directory if it doesn't exist
    if [[ ! -d "$install_dir" ]]; then
        info "Creating installation directory: $install_dir"
        mkdir -p "$install_dir" || sudo mkdir -p "$install_dir"
    fi

    local dest="${install_dir}/${binary_name}"

    # Check if we need sudo
    if [[ -w "$install_dir" ]]; then
        cp "$binary_path" "$dest"
        chmod +x "$dest"
    else
        info "Requesting sudo access to install to $install_dir..."
        sudo cp "$binary_path" "$dest"
        sudo chmod +x "$dest"
    fi

    success "Installed kindplane to $dest"
}

# Verify the installation
verify_installation() {
    local install_dir="$1"
    local binary_name="$2"

    if command -v "$binary_name" &> /dev/null; then
        success "Installation verified!"
        echo ""
        "$binary_name" version
        return 0
    fi

    # Check if it's in the install dir but not in PATH
    if [[ -x "${install_dir}/${binary_name}" ]]; then
        warn "kindplane installed but ${install_dir} is not in your PATH"
        echo ""
        echo "Add the following to your shell profile (.bashrc, .zshrc, etc.):"
        echo ""
        echo -e "  ${CYAN}export PATH=\"\$PATH:${install_dir}\"${NC}"
        echo ""
        return 0
    fi

    error "Installation verification failed"
}

# Check for required tools
check_prerequisites() {
    local missing=()

    # Check for download tools
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        missing+=("curl or wget")
    fi

    # Check for extraction tools
    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Missing required tools: ${missing[*]}"
    fi
}

# Main installation function
main() {
    print_banner

    # Check prerequisites
    check_prerequisites

    # Detect platform
    local platform
    platform=$(detect_platform)
    info "Detected platform: $platform"

    # Get version
    if [[ "$VERSION" == "latest" ]]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [[ -z "$VERSION" ]]; then
            error "Could not determine latest version. Please specify KINDPLANE_VERSION."
        fi
    fi
    info "Version: $VERSION"

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download and extract
    local binary_path
    binary_path=$(download_binary "$VERSION" "$platform" "$tmp_dir")

    # Install
    install_binary "$binary_path" "$INSTALL_DIR" "$BINARY_NAME"

    # Verify
    echo ""
    verify_installation "$INSTALL_DIR" "$BINARY_NAME"

    echo ""
    echo -e "${GREEN}${BOLD}Installation complete!${NC}"
    echo ""
    echo "Get started with:"
    echo ""
    echo -e "  ${CYAN}kindplane init${NC}      # Create configuration file"
    echo -e "  ${CYAN}kindplane up${NC}        # Bootstrap the cluster"
    echo -e "  ${CYAN}kindplane status${NC}    # Check cluster status"
    echo ""
    echo "For more information, visit: https://github.com/${GITHUB_REPO}"
    echo ""
}

# Run main
main "$@"
