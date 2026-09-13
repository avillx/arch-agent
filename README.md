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
