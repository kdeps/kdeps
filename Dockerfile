FROM debian:12.8-slim

RUN apt-get update && apt-get install -y curl git

RUN  adduser --disabled-password --gecos '' kdeps

USER kdeps

ARG VERSION=latest

RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh -s -- -d ${VERSION}

CMD  ["/home/kdeps/.local/bin/kdeps"]
