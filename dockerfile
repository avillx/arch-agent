FROM golang:1.25.5-alpine AS builder

WORKDIR /agent

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /agent/arch_agent ./cmd/agent/

FROM ubuntu:resolute-20260811.1

ENV USER=runner

WORKDIR /agent

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d /home/$USER -s /bin/bash $USER

RUN chown -R $USER:$USER /agent

USER $USER

COPY --from=builder /agent/arch_agent .

ENTRYPOINT ["/agent/arch_agent"]