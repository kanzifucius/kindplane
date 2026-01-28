# Prerequisites

This page lists all prerequisites for running kindplane.

## Required

### Docker

Docker is required to run Kind clusters. Kind uses Docker containers to simulate Kubernetes nodes.

=== "macOS"

    Install [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/):

    ```bash
    brew install --cask docker
    ```

    Or download from the [Docker website](https://docs.docker.com/desktop/install/mac-install/).

=== "Linux"

    Install Docker Engine:

    ```bash
    # Ubuntu/Debian
    sudo apt-get update
    sudo apt-get install docker-ce docker-ce-cli containerd.io

    # Fedora
    sudo dnf install docker-ce docker-ce-cli containerd.io

    # Start and enable Docker
    sudo systemctl start docker
    sudo systemctl enable docker
    ```

    Add your user to the docker group to run without sudo:

    ```bash
    sudo usermod -aG docker $USER
    newgrp docker
    ```

=== "Windows"

    Install [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/).

    Make sure WSL 2 backend is enabled for better performance.

Verify Docker is running:

```bash
docker version
```

### kubectl

kubectl is needed to interact with your Kubernetes cluster.

=== "macOS"

    ```bash
    brew install kubectl
    ```

=== "Linux"

    ```bash
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    ```

=== "Windows"

    ```powershell
    choco install kubernetes-cli
    ```

    Or download from the [Kubernetes website](https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/).

Verify kubectl is installed:

```bash
kubectl version --client
```

## Optional

### Cloud Provider CLIs

If you plan to use cloud provider credentials, you may need their respective CLIs:

#### AWS CLI

=== "macOS"

    ```bash
    brew install awscli
    ```

=== "Linux"

    ```bash
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
    unzip awscliv2.zip
    sudo ./aws/install
    ```

Configure credentials:

```bash
aws configure
```

#### Azure CLI

=== "macOS"

    ```bash
    brew install azure-cli
    ```

=== "Linux"

    ```bash
    curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
    ```

Login to Azure:

```bash
az login
```

### Task (Optional)

[Task](https://taskfile.dev/) is a task runner used for development:

```bash
# macOS
brew install go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

## System Requirements

### Minimum Requirements

| Resource | Minimum |
|----------|---------|
| CPU | 2 cores |
| RAM | 4 GB |
| Disk | 10 GB free |

### Recommended Requirements

| Resource | Recommended |
|----------|-------------|
| CPU | 4+ cores |
| RAM | 8+ GB |
| Disk | 20+ GB free |

!!! tip "Docker Resources"
    If using Docker Desktop, ensure you've allocated sufficient resources in Docker Desktop preferences. For best performance, allocate at least 4 CPUs and 8 GB RAM.

## Checking Prerequisites

kindplane includes a built-in doctor command to verify all prerequisites:

```bash
kindplane doctor
```

This checks:

- Docker is running
- kubectl is installed
- Sufficient disk space
- Optional tools (helm)

If any required checks fail, you'll see suggestions for how to fix them.

For more details, see the [doctor command documentation](../commands/doctor.md).
