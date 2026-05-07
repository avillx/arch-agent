# Makefile

.PHONY: build run dropsession dream

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

CONFIG ?= config.toml
DATADIR ?= ./data

build-agent:
	go build -o bin/agent ./cmd/agent
	
run-agent:
	go run ./cmd/agent -config=$(CONFIG) -datadir=$(DATADIR)

dream:
	go build -o bin/dream ./cmd/dream
	go run ./cmd/dream -config=$(CONFIG) -datadir=$(DATADIR)

dropsession:
	go build -o bin/dropsession ./cmd/dropsession
	go run ./cmd/dropsession -datadir=$(DATADIR)