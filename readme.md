# ARCH.agent

> The project is Agent SDK with memory. written in golang

Key points:
- Project focus on control clean context as possible and present only necceccary accesses
- Agent SDK is designed for work as server daemon, not as desktop app.
- File database is a decigion for minimize infrastructure.
- Agents have access for filesystem only for manage memory, not for dev or complete tasks.
- Agents can works autonomusly by cron scheduling.
- If agent has memory or works autonomusly it logging consolidated activity.

--- 

### Config
Must be runned with `--config` flag and path to config file e.g. `config.toml`
Example in `example.config.toml`

### ENV
You must define secrects in envvars and leave references on it on config file
Accepts unneccecary vars:
LOG_PRETTY (true/false)
LOG_LEVEL (debug/info/warn/error)

### For compose
Workdir is `/agent`
All memory stores in `/agent/data/...`

### TODO

**alpha v0.1**
- [X] All services names to Service to svc 
- [X] Polish telegram module
- [X] Should compact refactor (systemPrompt and summarization dialog accounting and tools)
- [X] preload last 50 lines of logs when option
- [X] Refactor openAI adapter (settings map parsing)
- [X] func (a *agent) HasMemory() bool
- [X] AutoDream
- [X] check depricated (before restruct) variable names
- [X] agent fs as sandbox iface over fs
- [X] Multimodal input
- [X] processing interruption (edge case first user message concat with last sess user message)
- [X] remove subsessions
- [ ] MCP Service and tools
- [ ] Services proto, services managing funcs
- [ ] API endpints

**alpha v0.5**
- [ ] shell access
- [ ] envvars managment
- [ ] web search gateway as CLI
- [ ] exclude telegram integration

**alpha v0.7**
- [ ] Skill adding tools
- [ ] Observer skill selected tool - result excluded
- [ ] harness hooks (on complete, on done, on tool use, on sub agent call)

**alpha v0.9**
- [ ] view over file_read (files, images and docs)
- [ ] bm25 fileSearch
- [ ] sqlite database

**v1**
- [ ] eliminate edge case vulnurabilities for every package
- [ ] error system refactor
- [ ] add tests
- [ ] add validations
- [ ] e2e tests 
- [ ] from zero launch (preloaded agent, cli, skills)