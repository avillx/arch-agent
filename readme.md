# ARCH.agent

## Config
Must be runned with `--config` flag and path to config file e.g. `config.toml`
Example in `example.config.toml`

## env
You must define secrects in envvars and leave references on it on config file
Accepts unneccecary vars:
LOG_PRETTY (true/false)
LOG_LEVEL (debug/info/warn/error)

## For compose
Workdir is `/agent`
All memory stores in `/agent/data/...`

## TODO
- completion agregation
- Multimodal input
- TODO tool
- Skills
- file system tools
- bash tools
- Advanced schedule control
- web_search + fetch_page
- Dreaming + divie own knowladges (dreamable) + external knowledges (undreamable)
- Rest endpoints and server
- isolate service packages by interfaces
- check depricated (before restruct) variable names

И LiveSession внутри AgentRuntime — просто скользящее окно контекста, 
детали реализации. Снаружи это не "сессия" вообще.
