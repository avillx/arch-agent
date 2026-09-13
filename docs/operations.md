# Operations

Routine operational facts: starting, stopping, logs, cleanup and maintenance.

## Starting and stopping

The system is a single HTTP server process:

- Start it with the binary; the server serves the API on `PORT` (default 8080).
  All runtime parameters come from environment variables — see
  [environment-variables.md](environment-variables.md).
- On the first start the data directory is created and initialized
  ([configuration.md](configuration.md)).
- Stop it with `Ctrl-C` or `SIGTERM`: the process shuts down gracefully,
  saving current state, and exits.

## Logs

Two destinations:

- **Console** — the process output. JSON lines by default, or human-readable
  plain text (`LOG_JSON=false`); the level is controlled by `LOG_LEVEL`
  (default `error`).
- **Agent log** — a plain text log file inside the data directory
  (`agent.log`). It is meant to be readable by the agent itself, so it is the
  only log the agent may read. During cleanup it is trimmed to the last
  `MAX_LOG_LINES` lines (default 1000).

## Routine cleanup

A maintenance pass runs **at startup and then every `CLEAN_UP_INTERVAL` hours**
(default 12, minimum 5). Each pass:

- deletes **expired sessions** — sessions whose last update is older than
  `SESSION_RETENTION` hours (default 240, minimum 10); see
  [sessions.md](sessions.md);
- deletes **broken sessions** — conversation files whose metadata cannot be
  read;
- trims the agent log to `MAX_LOG_LINES`.

Additionally, the temporary directory (`tmp/`, used for file staging) clears
its entries **10 minutes after they are created** — files placed there do not
need manual cleanup.

Invalid values at startup (retention under 10 h, interval under 5 h) prevent the
service from starting.

## Config changes

All configuration files are hot-reloaded: `models.toml`, `secrets.toml`,
`mcp.toml`, `memory.toml`, `tasks.toml`. A change is applied automatically; if
it is invalid, the change is **rejected and the last valid state stays active**
(see [configuration.md](configuration.md)). Prefer editing configs through the
API or while the service is stopped; manual edits may be overwritten.

## Data directory

The data directory is the whole state of the system: configuration, agents,
sessions, skills, memory, activity, secrets. **Back it up regularly**, and back
up before version upgrades. Never edit its files while the service runs
(except through the API) — the agent treats some of them (own logs, secrets)
as read-only and may overwrite manual changes.

## Health checks

- The API responding on `PORT` means the HTTP layer is up: try a `GET /agent`
  or `GET /tools`.
- Config errors (e.g. unreachable model provider) are logged and reported via
  the API error format rather than crashing the process — a running process is
  not necessarily a healthy one, check the logs.
- `GET /mcp` and `GET /providers` show the connected state of external
  integrations at a glance.