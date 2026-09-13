---
name: ozy-backend-go
description: >-
  Use this skill for any task involving the OzyAssist Go backend: adding API endpoints,
  modifying the agent loop, working with providers, SQLite schema changes, WebSocket handlers,
  or any file under backend/. Provides architecture context, file map, and development workflow.
---

# OzyAssist Backend Go — Skill

## Architecture Overview

```
backend/
├── cmd/server/main.go          # Entry point, server init, port 8080
├── internal/
│   ├── api/
│   │   ├── router.go           # Gin route registration — add new routes here
│   │   ├── handlers/           # HTTP REST handlers (one file per domain)
│   │   └── ws/chat.go          # WebSocket handler — message dispatch & streaming
│   ├── agent/
│   │   ├── loop.go             # ReAct Loop (AgentLoopParams, VoiceMode, maxTurns=25)
│   │   ├── tools.go            # AgentTools + VoiceAgentTools definitions
│   │   ├── executor.go         # Tool execution dispatch (os_*, web_search, etc.)
│   │   └── permissions.go      # Permission levels: Sandboxed/Supervised/Autonomous
│   ├── providers/
│   │   ├── registry.go         # Provider registration & discovery
│   │   ├── openai.go           # OpenAI-compatible (LM Studio, Ollama, OpenRouter)
│   │   └── anthropic.go        # Anthropic Claude provider
│   ├── db/
│   │   ├── sqlite.go           # DB init, migrations (initSchema)
│   │   ├── models/             # Go structs: Chat, Message, Skill, Project, etc.
│   │   └── *.go                # CRUD functions per model
│   ├── memory/                 # Episodic memory (Qdrant optional, SQLite fallback)
│   ├── system/                 # OS tools: desktop, apps, windows, file ops
│   └── audio/                  # Audio capture (WASAPI), wake word engine
```

## Key Patterns

### Adding a New Tool to the Agent
1. Define schema in `agent/tools.go` — add to `AgentTools` AND `VoiceAgentTools` if OS-related
2. Add case in `agent/executor.go` `executeToolCall()` switch
3. Implement logic in `internal/system/` or inline
4. Test with `go test ./internal/agent/...`

### Adding a New REST Endpoint
1. Write handler in `internal/api/handlers/name.go`
2. Register route in `internal/api/router.go`
3. Add frontend call in `frontend/src/services/api.ts`

### Adding a New Provider
1. Implement `providers.Provider` interface in `internal/providers/name.go`
2. Register in `providers.init()` or via `RegisterProvider()`

### SQLite Schema Change
1. Add migration SQL to `initSchema()` in `db/sqlite.go` (use `IF NOT EXISTS`)
2. Update struct in `db/models/`
3. Add/update CRUD functions

## Build & Test Commands
```powershell
# Build
cd backend; go build -o ozy-server.exe ./cmd/server/main.go

# Run tests
cd backend; go test ./...

# Run specific package
cd backend; go test ./internal/agent/... -v

# Run server (dev)
cd backend; go run cmd/server/main.go
```

## Critical Rules
- **WebSocket messages** use `serverMessage` struct in `ws/chat.go` — never add raw JSON
- **VoiceMode** (`AgentLoopParams.VoiceMode=true`) uses compact prompt (~50 tokens) + `VoiceAgentTools` only
- **SQLite migrations** must be idempotent — always use `IF NOT EXISTS` or `ADD COLUMN IF NOT EXISTS`
- **Provider errors** must be propagated to the WS client via `emit(AgentEvent{Type:"error", ...})`
- All handlers must call `db.DefaultUserID()` — multi-user auth is PIN-based
