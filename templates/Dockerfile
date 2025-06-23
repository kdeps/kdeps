# syntax=docker.io/docker/dockerfile:1
FROM ollama/ollama:{{.ImageVersion}}

ENV SCHEMA_VERSION={{.SchemaVersion}}
ENV OLLAMA_HOST={{.HostIP}}:{{.OllamaPortNum}}
ENV KDEPS_HOST={{.KdepsHost}}
ENV DEBUG=1

{{.EnvsSection}}

{{.ArgsSection}}

# Copy DownloadDir to local Downloads
COPY cache /cache
RUN chmod +x /cache/pkl*
{{if .InstallAnaconda}}
RUN chmod +x /cache/anaconda*
{{end}}

ARG DEBIAN_FRONTEND=noninteractive
ENV TZ={{.Timezone}}

# Install necessary tools
RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends \
    bzip2 ca-certificates git subversion mercurial libglib2.0-0 \
    libsm6 libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxinerama1 libxrandr2 libxrender1 \
    gpg-agent openssh-client procps software-properties-common wget curl nano jq python3 python3-pip musl musl-dev \
    musl-tools

# Determine the architecture and download the appropriate pkl binary
RUN arch=$(uname -m) && \
    if [ "$arch" = "x86_64" ]; then \
	cp /cache/pkl-linux-{{.PklVersion}}-amd64 /usr/bin/pkl; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /cache/pkl-linux-{{.PklVersion}}-aarch64 /usr/bin/pkl; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi

{{.PkgSection}}

# Setup kdeps
{{if .DevBuildMode}}
RUN cp /cache/kdeps /bin/kdeps
RUN chmod a+x /bin/kdeps
{{else}}
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh -s -- -b /bin/ -d "latest"
{{end}}

# Copy workflow
COPY workflow /agent/project
COPY workflow /agent/workflow

{{if .InstallAnaconda}}
# Conditionally Install Anaconda and Additional Packages
RUN arch=$(uname -m) && if [ "$arch" = "x86_64" ]; then \
	cp /cache/anaconda-linux-{{.AnacondaVersion}}-x86_64.sh /tmp/anaconda.sh; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /cache/anaconda-linux-{{.AnacondaVersion}}-aarch64.sh /tmp/anaconda.sh; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi

RUN /bin/bash /tmp/anaconda.sh -b -p /opt/conda
RUN ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh
RUN find /opt/conda/ -follow -type f -name '*.a' -delete
RUN find /opt/conda/ -follow -type f -name '*.js.map' -delete
RUN /opt/conda/bin/conda clean -afy
RUN rm /tmp/anaconda.sh
RUN . /opt/conda/etc/profile.d/conda.sh && conda activate base

RUN echo "export PATH=/opt/conda/bin:$PATH" >> /etc/environment
ENV PATH="/opt/conda/bin:$PATH"

{{.CondaPkgSection}}
{{end}}

{{.PythonPkgSection}}

# Cleanup
RUN apt-get clean && rm -rf /var/lib/apt/lists/*
RUN rm -rf /cache

{{if .ApiServerMode}}
EXPOSE {{.ExposedPort}}
{{end}}

# Entry Point and Command
ENTRYPOINT ["/bin/kdeps"] 