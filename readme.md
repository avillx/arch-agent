# ARCH.agent

## config
Must be runned with `--config` flag and path to config file e.g. `config.toml`
Example in `example.config.toml`

## env
You must define secrects in envvars and leave references on it on config file
Accepts unneccecary vars:
LOG_PRETTY (true/false)
LOG_LEVEL (debug/info/warn/error)

## for compose
Workdir is `/agent`
All memory stores in `/agent/data/...`

## TODO
### polish
- [ ] All services names to Service to svc
- [ ] isolate packages with little ifaces
- [ ] check depricated (before restruct) variable names
- [X] divide agent service and AgenticLoopRuntime
- [X] validate structs for ifaces e.g. `var _ iface = (*struct)(nil)`

### debt
- [ ] agent fs as sandbox iface over fs
- [ ] better web search gateway 
- [X] agent domain service over ChatService

### features
- [ ] API endpints
- [ ] skills
- [ ] activity drop
- [ ] Multimodal input
- [ ] instructions interface (tool usage guide preload)
- [ ] MCP Service and tools
- [ ] Summarizations
- [ ] live session is not a session (is runtime supervisor)

### Tools
- [X] filesystem
- [X] fetch
- [X] web_search
- [X] a2a 
- [X] cron control
- [ ] clarification
- [ ] think
- [ ] todo
- [ ] view over file_read (files, images and docs)
- [ ] async a2a
- [ ] modelpicker
- [ ] ssh
- [ ] shell