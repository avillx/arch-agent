# ARCH.agent

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![openapi initiative](https://img.shields.io/badge/openapiinitiative-%23000000.svg?style=for-the-badge&logo=openapiinitiative&logoColor=white)


An agent SDK that runs as a server daemon: autonomous AI agents with memory,
tools, MCP integration and cron-scheduled tasks, exposed through a plain HTTP
API. All state lives in a file database on disk — no external infrastructure.

Key points:
- Container first approach
- Agents work autonomously on a cron schedule (scheduled tasks).
- Everything an agent does is written to activity logs and consolidated once a
  day into persistent memory the agent can use.
- Built-in tools (filesystem, shell, web, to-do, delegation to other agents)
  plus external tools via MCP or provided by your API client per request.
- Works with any OpenAI-compatible endpoint.

Relevant **[Documentation](docs/index.md)**.

## Getting started
See current [guide](docs/getting-started.md)

## License
This program is licensed under the GNU General Public License v3.0 (GPLv3). See the full license: `https://www.gnu.org/licenses/gpl-3.0.txt`

Source: `https://github.com/avillx/arch-agent`

```
avillx/arch-agent: Copyright (C) 2026 avillx

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
```
