FROM golang:1.25.5-alpine AS builder

WORKDIR /agent

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /agent/arch_agent ./cmd/

FROM alpine:3.20

ENV USER=runner

WORKDIR /agent

RUN adduser $USER -h /home/$USER -D

RUN chown -R $USER:$USER /agent

USER $USER

COPY --from=builder /agent/arch_agent .

ENTRYPOINT ["/agent/arch_agent"]