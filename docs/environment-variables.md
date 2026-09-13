# Environment Variables

The system is configured entirely through environment variables — there is no
`.env` file, no config flags. All variables are optional; each has a sensible
default.

## Reference

| Variable | Allowed values | Default | What it controls |
|----------|----------------|:-------:|------------------|
| `DATA_PATH` | filesystem path | `~/.arch-agent` | Directory that holds all configuration files and data (see [configuration.md](configuration.md)) |
| `PORT` | port number | `8080` | Port the HTTP API listens on |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `error` | Minimum severity written to the console log |
| `LOG_JSON` | `true`, `false` | `false` | Write console log in JSON format instead of plain text |
| `LOG_INDENTED` | `true`, `false` | `false` | Pretty-print multi-line log values |
| `LOG_SOURCE` | `true`, `false` | `false` | Include the source location (file and line) in every log record |
| `CLEAN_UP_INTERVAL` | integer, hours | `12` | How often routine cleanup runs (temporary files, expired sessions, log trimming) |
| `SESSION_RETENTION` | integer, hours | `240` | How long a session lives before it is considered stale and removed (see [sessions.md](sessions.md)) |
| `MAX_LOG_LINES` | integer | `1000` | Maximum number of lines kept in the agent-visible log file; the oldest lines are trimmed (see [operations.md](operations.md)) |

## Units

`CLEAN_UP_INTERVAL` and `SESSION_RETENTION` are always expressed in **hours**.

## Examples

```bash
# Use a dedicated data directory and a non-default port
DATA_PATH=/var/lib/arch-agent PORT=9000 ./arch-agent

# Verbose, structured logging for troubleshooting
LOG_LEVEL=debug LOG_JSON=true LOG_SOURCE=true ./arch-agent

# More aggressive cleanup and longer session retention
CLEAN_UP_INTERVAL=4 SESSION_RETENTION=720 ./arch-agent
```

With Docker:

```bash
docker run -e DATA_PATH=/agents -e PORT=8080 -p 8080:8080 -v agents:/agents ghcr.io/arch-agent/arch-agent
```

## Notes

- A variable set to an empty string is treated as unset and the default applies.
- If a variable has an invalid value (for example `LOG_LEVEL=trace`), the
  process refuses to start and reports the problem.
- Changing variables requires a restart — unlike configuration files, they are
  not hot-reloaded.