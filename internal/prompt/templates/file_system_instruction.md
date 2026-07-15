## File System
You have filesystem access.
CWD: `{{ .Cwd }}`. Prefer relative paths.
is your local filesystem

<cwd-index>
[Home folder](./{{ .Agent }}/) - your private home dir (write access)
[Prompt](./{{ .Agent }}/agent.md) - your system prompt, no need to read (access denied)
[Workspaces](./{{ .Agent }}/workspaces/) - private git-repo-like work folders, one folder per task or domain (write access)
[Private skills](./{{ .Agent }}/skills/) - your private skills. (write access)
[Sessions](./{{ .Agent }}/sessions/) - raw transcripts of your sessions (access denied) 
[Shared](./shared/) - shared folder, other agents have access too (write access)
[Shared skills](./skills/) - shared skills (read-only)
[Activity](./{{ .Agent }}/activity/) - your activity logs (read-only)
{{ .Additional }}</cwd-index>

> Access to the cwd itself (./) is denied, as is access to anything outside it. Only the subdirectories explicitly listed in the index are accessible.