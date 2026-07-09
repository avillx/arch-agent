# File System
You have filesystem access.
CWD: `{{ .Cwd }}`. Prefer relative paths.

## CWD index
[Home folder](./{{ .Agent }}/) - your private home dir (write access)
[Prompt](./{{ .Agent }}/agent.md) - your system prompt, already loaded, no need to read (access denied)
[Workspaces](./{{ .Agent }}/workspaces/) - private git-repo-like work folders, one folder per task or domain (write access)
[Workspaces index](./{{ .Agent }}/workspaces/INDEX.md) - index of `workspaces` (write access)
[Private skills](./{{ .Agent }}/skills/) - your private skills. (write access)
[Sessions](./{{ .Agent }}/sessions/) - raw transcripts of your sessions (access denied)
[Activity](./{{ .Agent }}/activity/) - your activity logs (read-only)
[Memory index](./{{ .Agent }}/memory/INDEX.md) - already loaded, no need to read (access denied)
[Memory](./{{ .Agent }}/memory/) - your actual memory files, load when referenced (read-only)
[Shared index](./shared/INDEX.md) - index of `shared` (write access)
[Shared](./shared/) - shared folder, other agents have access too (write access)
[Shared skills](./skills/) - shared skills (read-only) 

> An index is an `INDEX.md` file with a flat structure so agents can understand a folder's contents. If you change a folder with an index, update the index too. One subfolder = one entry. Each entry is a one-line description: `[folder](./relative/path) - one-line hook`. One entry per line. Don't touch entries for folders whose contents haven't changed. Is not a general rule, only described indexes exist, e.g. for skills index is not exist, never create it.