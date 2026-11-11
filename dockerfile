FROM golang:1.27.3-alpine AS builder

WORKDIR /arch

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /build/ard ./cmd/agent/

FROM ubuntu:resolute-20260811.1

ENV USER=agent
ENV AGENT_DATAPATH=/agent

WORKDIR /

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    ca-certificates \
    # python3-pip \
    # python3-venv \
    # python3-dev \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d /home/$USER -s /bin/bash $USER

RUN mkdir $AGENT_DATAPATH && chown $USER:$USER $AGENT_DATAPATH && chmod 755 $AGENT_DATAPATH

COPY --chmod=444 ./README.md ./LICENSE /opt/ard/
COPY --chmod=555 ./docs $AGENT_DATAPATH/docs
COPY --chmod=555 --from=builder /build/ard /opt/ard/ard
COPY --chmod=555  scripts/entrypoint.sh /opt/ard/entrypoint.sh

USER $USER

ENTRYPOINT ["/opt/ard/entrypoint.sh"]
