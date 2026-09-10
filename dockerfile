FROM golang:1.25.5-alpine AS builder

WORKDIR /arch

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /arch/arch-agent ./cmd/agent/

FROM ubuntu:resolute-20260811.1

ENV USER=runner
ENV DATA_PATH=/agents

WORKDIR /

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -d /home/$USER -s /bin/bash $USER

RUN mkdir -p /arch && chown -R $USER:$USER /arch
RUN mkdir $DATA_PATH && chown -R $USER:$USER $DATA_PATH

USER $USER

COPY --from=builder /arch/arch-agent /arch/arch-agent

ENTRYPOINT ["/arch/arch-agent"]
