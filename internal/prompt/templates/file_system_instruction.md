## File System
You have filesystem access.
CWD: `{{ .Cwd }}`. Prefer relative paths.
is your local filesystem

<cwd-index>
- [Home folder](./{{ .Agent }}) - your private home dir (write access)
- [Prompt](./{{ .Agent }}/agent.md) - your system prompt, no need 
  to read (access denied)
- [Workspaces](./{{ .Agent }}/workspaces/) - private git-repo-like work folders, 
  one folder per task or domain (write access)
- [Private skills](./{{ .Agent }}/skills/) - your private skills. (write access)
- [Sessions](./{{ .Agent }}/sessions/) - raw transcripts of your 
  sessions (access denied) 
- [Shared](./shared/) - shared folder, other agents have access too (write access)
- [Shared skills](./skills/) - shared skills (read-only)
- [Activity](./{{ .Agent }}/activity/) - your activity logs (read-only)
- [MCP servers](./mcp.toml) - contain mcp connections configs
- [Models](./models.toml) - allowed models with params 
- [Secrets](./secrets.toml) - secrets (env vars)
- [Tasks](./tasks.toml) - scheduled by cron tasks for all agents
- [System logs](./.log) - contains logs for all agent system. 
  ( MCP, sessions, memory consolidations, agent runs, 
  runtime errors, agent api calls, agent awakes by schedule tasks etc...)
  do not read it raw, prefer to use tail / grep
{{ .Additional }}</cwd-index>