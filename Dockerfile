FROM debian:12.8-slim

# Install dependencies
RUN apt-get update && apt-get install -y curl git

# Add kdeps user
RUN adduser --disabled-password --gecos '' kdeps

# Switch to kdeps user
USER kdeps

# Define build arguments
ARG VERSION=latest

# Install kdeps
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh -s -- -d ${VERSION}

# Determine architecture and install latest pkl version
RUN ARCH=$(uname -m) && \
    PKL_VERSION=$(curl -s https://api.github.com/repos/apple/pkl/releases/latest | grep -o '"tag_name": *"[^"]*"' | sed 's/"tag_name": *"\(.*\)"/\1/') && \
    echo "Installing pkl version: $PKL_VERSION" && \
    if [ "$ARCH" = "aarch64" ]; then \
        curl -L -o /home/kdeps/.local/bin/pkl "https://github.com/apple/pkl/releases/download/${PKL_VERSION}/pkl-linux-aarch64"; \
    elif [ "$ARCH" = "x86_64" ]; then \
        curl -L -o /home/kdeps/.local/bin/pkl "https://github.com/apple/pkl/releases/download/${PKL_VERSION}/pkl-linux-amd64"; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi && \
    chmod +x /home/kdeps/.local/bin/pkl && \
    /home/kdeps/.local/bin/pkl --version

# Set Docker mode environment variable to skip config file lookup
ENV DOCKER_MODE=1

# Set the default command
CMD ["/home/kdeps/.local/bin/kdeps"]
