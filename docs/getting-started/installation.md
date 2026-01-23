# Installation

There are several ways to install kindplane on your system.

## Quick Install (Recommended)

The easiest way to install kindplane is using the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash
```

This script will:

- Detect your operating system and architecture
- Download the appropriate binary
- Install it to `/usr/local/bin` (or `~/.local/bin` if no sudo access)
- Verify the installation

### Install a Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | KINDPLANE_VERSION=v0.1.0 bash
```

### Custom Installation Directory

```bash
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | KINDPLANE_INSTALL_DIR=/opt/bin bash
```

## Download Binary

Download the latest release directly from the [releases page](https://github.com/kanzifucius/kindplane/releases).

=== "macOS (Apple Silicon)"

    ```bash
    curl -LO https://github.com/kanzifucius/kindplane/releases/latest/download/kindplane_darwin_arm64.tar.gz
    tar -xzf kindplane_darwin_arm64.tar.gz
    sudo mv kindplane /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -LO https://github.com/kanzifucius/kindplane/releases/latest/download/kindplane_darwin_amd64.tar.gz
    tar -xzf kindplane_darwin_amd64.tar.gz
    sudo mv kindplane /usr/local/bin/
    ```

=== "Linux (amd64)"

    ```bash
    curl -LO https://github.com/kanzifucius/kindplane/releases/latest/download/kindplane_linux_amd64.tar.gz
    tar -xzf kindplane_linux_amd64.tar.gz
    sudo mv kindplane /usr/local/bin/
    ```

=== "Linux (arm64)"

    ```bash
    curl -LO https://github.com/kanzifucius/kindplane/releases/latest/download/kindplane_linux_arm64.tar.gz
    tar -xzf kindplane_linux_arm64.tar.gz
    sudo mv kindplane /usr/local/bin/
    ```

## Build from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/kanzifucius/kindplane.git
cd kindplane

# Build using Go
go build -o bin/kindplane ./cmd/kindplane

# Or use Task (if installed)
task build

# Move to PATH
sudo mv bin/kindplane /usr/local/bin/
```

### Requirements for Building

- Go 1.23 or later
- Git

## Verify Installation

After installation, verify kindplane is working:

```bash
kindplane --help
```

You should see the help output with available commands.

## Updating

To update kindplane, simply run the install script again:

```bash
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash
```

## Uninstalling

To remove kindplane:

```bash
sudo rm /usr/local/bin/kindplane
```

Or if installed to user directory:

```bash
rm ~/.local/bin/kindplane
```
