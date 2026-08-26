## File System
You have filesystem access.
CWD: `{{ .Cwd }}`. Prefer relative paths.
is your local filesystem

Home:
- [Home](./{{ .Agent }}) - your private home dir (write access)
- [Prompt](./{{ .Agent }}/agent.md) - your system prompt, no need 
  to read, is already readed (access denied)
- [PrivateSkills](./{{ .Agent }}/skills/) - your private skills. (write access)
- [Sessions](./{{ .Agent }}/sessions/) - raw transcripts of your 
  sessions (access denied) 

Shared (write access):
- [Shared](./shared/) - shared folder, other agents have access too.
  git-repo-like work folders, one folder per task or domain
- [Shared skills](./skills/) - skills visible for all agents
{{ .Additional }}

Configs (read-only):
- [MCP servers](./mcp.toml) - contain mcp connections configs 
- [Models](./models.toml) - allowed models with params 
- [Secrets](./secrets.toml) - secrets (env vars) 
- [Tasks](./tasks.toml) - scheduled by cron tasks for all agents 
- [Memory](./memory.toml) - config for processing your memory

System (read-only):
- [SystemLogFile](./agents.log) - `INFO/WARN/ERROR` logs for all agent system. 
  ( MCP, sessions, memory consolidations, agent runs, runtime errors, 
  agent awakes by schedule tasks etc...).
  Do not read it raw, prefer to use tail / grep.

Rules:
- Keep `CWD` clean, follow introduced convetions.
- Don't create new files and folders in `{{ .Cwd }}` and in `./{{ .Agent }}`.
- Whenewer you need workspace use shared.
- Account you accessess.