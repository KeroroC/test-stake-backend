# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go backend that monitors an Ethereum staking contract via WebSocket, persists on-chain events to MySQL, and exposes them through a REST API. The contract ABI (`internal/abi/Stake.abi.json`) is embedded at compile time.

## Build & Run

```bash
# Build
go build -o bin/server .

# Run (requires config.yaml in working directory)
./bin/server

# Run tests
go test ./...

# Run a single test
go test ./internal/repository/ -run TestStakedEventRepository

# Lint (if golangci-lint is installed)
golangci-lint run
```

## Configuration

Configuration is loaded from `config.yaml` via Viper. See `config.yaml.sample` for the required structure:
- `server` — HTTP host/port/mode (debug/release)
- `database` — MySQL connection (GORM auto-migrates with `t_` table prefix)
- `redis` — Redis connection
- `eth` — RPC/WS URLs, contract address, `start_block` (must be set to the contract deployment block)

## Architecture

Four-layer structure per event type. Each new contract event requires changes in all four layers plus registration in `main.go` and `internal/api/router.go`:

```
listener → repository → service → api
(Chain)    (MySQL)      (Logic)   (HTTP/Gin)
```

**Key interfaces and patterns:**

- `listener.ContractEventHandler` — interface for event handlers. Each handler decodes logs via the embedded ABI and writes to its repository. Registered on `ContractEventListener` by event topic0 hash.
- `ContractEventListener` — connects via WebSocket, replays missed blocks in batches of 1000, then subscribes to real-time logs. Tracks last synced block in `t_contract` table for crash recovery.
- Repository layer — each event type has its own repository with `Create` (idempotent, ignores duplicate keys) and `List`/`GetByID` (paginated). Validation and query helpers live in `repository/common.go`.
- Service layer — thin wrapper around repositories, returns paginated result structs.
- API layer — Gin handlers, one group per event type (e.g., `/staked-events`). Query params parsed manually.

**Adding a new event type:**
1. Add model in `internal/models/`
2. Add repository in `internal/repository/` following the existing pattern (use `BaseQuery`, `isDuplicateKeyError`, `normalizeStrings`)
3. Add service in `internal/service/`
4. Add listener handler in `internal/listener/` implementing `ContractEventHandler`
5. Add API handler in `internal/api/`
6. Register repository + handler in `main.go`, register routes in `internal/api/router.go`

## Conventions

- Table names use `t_` prefix with singular nouns (GORM `NamingStrategy`)
- All hex addresses and hashes are lowercased before storage (`normalizeStrings`)
- uint256 values stored as decimal strings (varchar 78)
- Duplicate key errors on Create are silently ignored (idempotent writes)
- Pagination defaults: page=1, pageSize=20, maxSize=100
