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
- [X] All services names to Service to svc 
- [X] Polish telegram module
- [X] Should compact refactor (systemPrompt and summarization dialog accounting and tools)
- [X] preload last 50 lines of logs when option
- [ ] Skill adding tools
- [ ] Refactor openAI adapter (settings map parsing)
- [ ] Observer skill selected tool - result excluded
- [ ] func (a *agent) HasMemory() bool
- [ ] AutoDream
- [ ] Error system refactor
- [ ] check depricated (before restruct) variable names
- [ ] eliminate edge case vulnurabilities for every package
- [ ] agent fs as sandbox iface over fs
- [ ] better web search gateway 
- [ ] MCP Service and tools
- [X] Multimodal input
- [ ] API endpints
- [ ] view over file_read (files, images and docs)
- [ ] bm25 fileSearch
