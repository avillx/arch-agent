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

## TODO:
- API, web socket based reverse MCP transport
- tui as external client
- chrony is a external client
- tg ui as extranal client
- determinated emo engine
- create a react entity that includes:
    - messages
    - reflection
- create a dynamic config

## tech debt:
- tool description must be on definition on an schema.
- drop schema from app layer, only properties array. shema is a infra concept.
- execution context too complex
- need a llm op's DI constructors
- main is too large
- logging must be polished
- simplify config, and key injections
- use case divide on services