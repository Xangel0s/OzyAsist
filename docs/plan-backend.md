# Plan de Desarrollo — Backend OzyAssist

> Fecha inicio: Julio 2026
> Stack: Go + Gin + SQLite + Qdrant

---

## Estado general

| Fase | Estado | Inicio | Fin |
|---|---|---|---|
| **Fase 1 — Scaffolding** | ✅ Completado | Jul 2026 | Jul 2026 |
| **Fase 2 — Chat funcional** | ⏳ Pendiente | — | — |
| **Fase 3 — Memoria** | ⏳ Pendiente | — | — |
| **Fase 4 — Agent System** | ⏳ Pendiente | — | — |
| **Fase 5 — MCP + Skills** | ⏳ Pendiente | — | — |
| **Fase 6 — Browser + Sidebar** | ⏳ Pendiente | — | — |
| **Fase 7 — Modelos locales** | ⏳ Pendiente | — | — |
| **Pre-release** | ⏳ Pendiente | — | — |

---

## Fase 1 — Scaffolding

### 1.1 Estructura de carpetas
- [x] `backend/cmd/server/main.go`
- [x] `backend/internal/api/router.go`
- [x] `backend/internal/api/handlers/` (todos los handlers stub)
- [x] `backend/internal/api/middleware/` (auth + ratelimit)
- [x] `backend/internal/providers/` (interface + 4 stubs)
- [x] `backend/internal/memory/` (7 archivos)
- [x] `backend/internal/agent/` (9 archivos)
- [x] `backend/internal/db/sqlite.go`
- [x] `backend/internal/db/migrations/`
- [x] `backend/internal/db/models/`
- [x] `backend/internal/websocket/`
- [x] `backend/internal/skills/`
- [x] `backend/internal/connectors/`
- [x] `backend/internal/search/`

### 1.2 Go module
- [x] `go.mod` (module, Go 1.26, dependencias)
- [x] `go.sum` (generado con `go mod tidy`)

### 1.3 SQLite + Migraciones
- [x] Conexión SQLite (`modernc.org/sqlite`)
- [x] Migraciones aplicadas al iniciar
- [x] Tablas: users, projects, chats, messages, skills, connectors, agent_tasks, agent_actions
- [x] FTS5 virtual table para messages

### 1.4 Router + Server
- [x] `main.go` con startup
- [x] `router.go` con todas las rutas
- [ ] Servidor escuchando en :8080 *
- [x] CORS configurado

> * Pendiente de probar al integrar con frontend (Fase 2).

### 1.5 Docker Compose
- [ ] `docker-compose.yml` (solo Qdrant)
- [ ] `Dockerfile` (backend)

### Checklist final Fase 1
- [x] `go build ./...` sin errores
- [ ] `go vet ./...` sin errores  
- [ ] Servidor responde en `GET /api/health` *
- [ ] Migraciones SQLite aplicadas *
- [x] Qdrant conectable (si está corriendo) — stub listo

> * Requiere ejecutar el servidor (próximo paso al integrar con frontend).

---

## Fase 2 — Chat funcional

### 2.1 Provider Router
- [ ] Interface `Provider` completa
- [ ] `anthropic.go` — streaming Anthropic API
- [ ] `openai.go` — streaming OpenAI API
- [ ] `openrouter.go` — streaming OpenRouter
- [ ] `lmstudio.go` — stub (fase 7)

### 2.2 WebSocket
- [ ] `hub.go` — connection manager
- [ ] `client.go` — read/write pump
- [ ] `WS /ws/chat/:id` operativo

### 2.3 CRUD Chats
- [ ] `GET /api/chats` — listar
- [ ] `POST /api/chats` — crear
- [ ] `GET /api/chats/:id` — obtener con mensajes
- [ ] `DELETE /api/chats/:id` — eliminar

### 2.4 Mensajes
- [ ] `POST /api/chats/:id/messages` — enviar y recibir stream por WS
- [ ] FTS5 search en mensajes

### Checklist Fase 2
- [ ] Chat end-to-end: frontend envía mensaje → backend recibe → enruta a Anthropic → stream por WS → se guarda en SQLite
- [ ] FTS5 search devuelve resultados

---

## Fase 3 — Memoria

### 3.1 Qdrant
- [ ] `qdrant.go` — cliente Qdrant (crear colección, upsert, search)
- [ ] Colección `ozy_memory` con vector 768d (nomic-embed-text-v1.5)

### 3.2 Embeddings
- [ ] `embeddings.go` — generar embeddings vía LM Studio API
- [ ] Fallback a API (OpenAI) si LM Studio no está disponible

### 3.3 Chunking
- [ ] `chunker.go` — dividir texto en chunks (512 tokens, 64 overlap)
- [ ] Preservar boundaries de párrafos

### 3.4 Memory Import
- [ ] `POST /api/memory/import` — sube .md → chunking → embeddings → Qdrant
- [ ] Feedback de IA sobre la memoria importada

### 3.5 Memoria de Proyecto (Capa 1)
- [ ] `project_memory.go` — .md vivo asociado a cada proyecto
- [ ] Carga completa al iniciar tarea sobre proyecto
- [ ] Editable por usuario

### 3.6 Memoria Episódica (Capa 2)
- [ ] `episodic.go` — al completar tarea, genera resumen → embebe → guarda en Qdrant
- [ ] Búsqueda semántica filtrada por proyecto al iniciar tarea nueva

### 3.7 Auto-memory de Chat
- [ ] Cada intercambio → embedding → Qdrant
- [ ] Top-5 chunks relevantes inyectados en prompt del siguiente mensaje

### Checklist Fase 3
- [ ] Import de .md → Qdrant funcional
- [ ] Búsqueda semántica devuelve resultados relevantes
- [ ] Contexto de memoria inyectado en prompts de chat

---

## Fase 4 — Agent System

### 4.1 Planner
- [ ] `planner.go` — recibe goal, descompone en pasos usando LLM
- [ ] Plan estructurado: `[{step, tool, target, expected_outcome}]`

### 4.2 Executor
- [ ] `executor.go` — ejecuta pasos secuencialmente
- [ ] Cada paso: decide tool → ejecuta → evalúa resultado → siguiente paso
- [ ] Manejo de errores y reintentos

### 4.3 Permisos
- [ ] `permissions.go` — niveles read_only / sandboxed / trusted
- [ ] `ValidatePath()` para sandboxed
- [ ] `DestructivePatterns` para trusted
- [ ] Confirmación por WebSocket cuando se requiere

### 4.4 Filesystem
- [ ] `filesystem.go` — read/write archivos con sandbox
- [ ] Diff preview antes de escribir (para el frontend)

### 4.5 Terminal
- [ ] `terminal.go` — ejecutar comandos con timeout
- [ ] Capturar stdout/stderr en tiempo real
- [ ] Streaming de output por WS

### 4.6 Code Index
- [ ] `code_index.go` — parseo AST ligero en Go
- [ ] Indexar funciones, tipos, símbolos por proyecto
- [ ] Embeddings de descripciones, no código crudo

### 4.7 Audit
- [ ] `audit.go` — registrar toda acción en `agent_actions`
- [ ] Task status transitions: planning → running → completed/failed/cancelled

### 4.8 API endpoints
- [ ] `POST /api/agent/task` — crear tarea
- [ ] `GET /api/agent/task/:id` — estado
- [ ] `POST /api/agent/task/:id/cancel` — cancelar
- [ ] `POST /api/agent/task/:id/confirm` — confirmar acción
- [ ] `WS /ws/agent/task/:id` — stream de progreso

### Checklist Fase 4
- [ ] Tarea end-to-end: goal → plan → ejecución → resultado
- [ ] Permisos funcionan (read_only bloquea escritura, sandboxed respeta ProjectRoot)
- [ ] Audit log completo
- [ ] Confirmaciones por WS operativas

---

## Fase 5 — MCP + Skills

### 5.1 MCP Client
- [ ] `mcp_client.go` — cliente MCP estándar (JSON-RPC)
- [ ] Conexión a servidores MCP
- [ ] Listar herramientas disponibles
- [ ] Ejecutar herramientas

### 5.2 Skills
- [ ] `executor.go` — ejecutar skills según trigger
- [ ] Tipos: script, prompt_template, api_call
- [ ] Integración con agente (skills como tools disponibles)

### 5.3 CRUD endpoints
- [ ] `GET/POST /api/skills`
- [ ] `DELETE /api/skills/:id`
- [ ] `GET/POST /api/connectors`
- [ ] `DELETE /api/connectors/:id`

### Checklist Fase 5
- [ ] Conector MCP se conecta y lista tools
- [ ] Skill se ejecuta y devuelve resultado

---

## Fase 6 — Browser + Sidebar

### 6.1 Browser Control
- [ ] `browser.go` — integración chromedp/rod (feature flag)
- [ ] Operaciones: navegar, click, extraer texto, screenshot, rellenar formularios
- [ ] Integración con agent planner (pasos de navegación)

### 6.2 Sidebar Assist
- [ ] `screenshot.go` — recibir screenshot del frontend
- [ ] Análisis de contexto (qué hay en pantalla)
- [ ] Sugerencias contextuales vía WS

### 6.3 API endpoints
- [ ] `POST /api/sidebar/observe`
- [ ] `POST /api/sidebar/command`

### Checklist Fase 6
- [ ] Browser control funcional (si feature flag activo)
- [ ] Sidebar recibe screenshot y responde con sugerencias

---

## Fase 7 — Modelos Locales

### 7.1 LM Studio Provider
- [ ] `lmstudio.go` — conexión a LM Studio API local
- [ ] Soporte de streaming
- [ ] Soporte de tools (si el modelo lo permite)

### 7.2 Embeddings Locales
- [ ] `embeddings.go` — usar `nomic-embed-text-v1.5` vía LM Studio
- [ ] Fallback ordenado si LM Studio no corre

### Checklist Fase 7
- [ ] Chat con modelo local funcional
- [ ] Embeddings locales funcionales

---

## Pre-release

### Checklist final
- [ ] Cifrado de API keys en reposo
- [ ] Qdrant standalone (sin Docker) opcional
- [ ] Cross-compile test (Linux, macOS, Windows)
- [ ] Documentación de instalación
- [ ] Prueba de import/export de SQLite

---

## Decisiones técnicas (cerradas)

| Decisión | Resolución |
|---|---|---|
| Base de datos | SQLite (`modernc.org/sqlite` — pure Go, sin cgo) |
| Búsqueda vectorial | Qdrant (Docker o standalone) |
| Full-text search | FTS5 (SQLite nativo) + triggers automáticos |
| Auth | Sin auth en modo local. JWT opcional para multiusuario a futuro |
| Modelo embeddings | `nomic-embed-text-v1.5` vía LM Studio (768d, local, gratuito) |
| API keys en reposo | Texto plano en MVP. Cifrado TODO pre-release |
| Browser automation | chromedp — feature flag (opcional, requiere Chromium ~300MB) |
| Permission model | read_only → sandboxed (ProjectRoot) → trusted (confirmaciones) |
| Migraciones | `golang-migrate` versionado desde día 1 |
| Provider lead | OpenAI primero (referencia), luego Anthropic, OpenRouter, LM Studio |
| Streaming | WebSocket con `context.WithCancel` — si el cliente desconecta, se cancela el HTTP request al LLM |
