# Installation

The agent runs as a single HTTP server process. Two ways to run it: a
pre-built **container image** or a **compiled binary**.

The API has no authentication and is meant
to run inside a trusted network or behind your gateway.

## Option A — Docker (recommended)

A ready image is published to GitHub Container Registry:

```
ghcr.io/avillx/arch-agent:latest
```

The image runs as a non-root user, has Python available inside (used by some
shell-based workflows) and stores all data under `/agents`.

Run it directly:

```sh
docker run -d \
  -p 8080:8080 \
  -v arch-agent-data:/agents \
  ghcr.io/avillx/arch-agent:latest
```

Or with Docker Compose:

```yaml
services:
  arch-agent:
    image: ghcr.io/avillx/arch-agent:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - arch-agent-data:/agents
    environment:
      - PORT=8080
      - LOG_LEVEL=info

volumes:
  arch-agent-data:
```

- The volume mounted at `/agents` is the **data directory** — it holds all
  configuration, agents, sessions and memory. Back it up; without it the system
  starts empty.
- Any environment variable from
  [environment-variables.md](environment-variables.md) can be passed in the same
  way; `PORT` changes the port the API listens on.

## Option B — compiled binary

Build from source (Go) or run a provided binary:

```sh
go build -o arch-agent ./cmd/agent
./arch-agent
```

With no configuration the server:

- uses `~/.arch-agent` as the data directory (override with `DATA_PATH`);
- listens on port `8080` (override with `PORT`).

The very first start creates the data directory and its internal structure.
Nothing else is required to boot.

## Network placement and security

**The HTTP API has no authentication.** Anyone who can reach the port can read
and change agents, secrets, sessions and execute chats. That is intentional —
it is designed to be used:

- inside a private/trusted network only;
- or behind a gateway that you control (e.g. a Telegram/chat gateway) that does
  the authentication and forwards requests to the API.

Do not expose the port to the public internet — the data directory also holds
your model API keys and secrets ([secrets.md](secrets.md)).

Next step: [getting-started.md](getting-started.md).
