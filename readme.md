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
### debt
- [ ] check depricated (before restruct) variable names
- [ ] agent fs as sandbox iface over fs
- [ ] isolate packages with little ifaces

### features
- [ ] API
- [ ] skills
- [ ] activity drop
- [ ] Multimodal input
- [ ] instructions interface (usage guide preload)
- [ ] MCP
- [ ] Summarizations
- [ ] live session is not a session (in AgentRuntime)

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