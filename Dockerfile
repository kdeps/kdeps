FROM debian:12.8-slim

# Install dependencies
RUN apt-get update && apt-get install -y curl git

# Add kdeps user
RUN adduser --disabled-password --gecos '' kdeps

# Switch to kdeps user
USER kdeps

# Define build arguments
ARG VERSION=latest
ARG GITHUB_TOKEN=secret

# Install kdeps
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh -s -- -d ${VERSION}

# Determine architecture and install pkl accordingly
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then \
        curl -L -o /home/kdeps/.local/bin/pkl 'https://github.com/apple/pkl/releases/download/0.27.1/pkl-linux-aarch64'; \
    elif [ "$ARCH" = "x86_64" ]; then \
        curl -L -o /home/kdeps/.local/bin/pkl 'https://github.com/apple/pkl/releases/download/0.27.1/pkl-linux-amd64'; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi && \
    chmod +x /home/kdeps/.local/bin/pkl && \
    /home/kdeps/.local/bin/pkl --version

# Set the default command
CMD ["/home/kdeps/.local/bin/kdeps"]
