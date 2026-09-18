# AGENTS.md — OzyAssist Guidelines & Instructions

## Project Overview
OzyAssist es un asistente autónomo de escritorio, código y cowork para Windows inspirado en Claude Desktop, Manus y Claude Code. Cuenta con un núcleo en Go de alto rendimiento, arquitectura Zero-Docker con SQLite en Go puro (`modernc.org/sqlite`), memoria híbrida en RAM, terminal con bucle de auto-reparación (*Self-Healing*), interfaz de consola TUI reactiva en Bubble Tea con sistema de colas, y soporte para un ecosistema multi-proveedor (Mistral AI, KiloCode Gateway, Cohere, Groq, OpenAI, Anthropic, DeepSeek y modelos locales).

## Architecture & Tech Stack
- **Core / Backend**: Go (Bubble Tea TUI / Gin REST / gorilla/websocket / pure-Go SQLite / local RAM embeddings).
- **Zero-Docker Policy**: No Docker containers. Pure-Go SQLite driver, in-memory caching, and local vector search.
- **Interfaces**:
  - `cmd/ozy`: CLI / TUI interactiva basada en Bubble Tea con streaming, pensamiento colapsable (`Ctrl+T`), colas de mensajes y atajos rápidos.
  - `cmd/server`: Servidor HTTP y WebSocket para integraciones cliente y streaming reactivo.
  - `cmd/ozyctl`: Herramienta de utilidades y administración.
- **Cognitive Triad (Single-LLM Architecture)**:
  - **OZY**: Ejecutor de acciones y llamadas a herramientas del sistema/MCP.
  - **CHARC**: Auditor de calidad que valida precondiciones y audita resultados antes de confirmarlos.
  - **NINE**: Estratega cognitivo que descompone tareas complejas y sintetiza respuestas finales.
- **Self-Healing Loop**:
  - Intercepción de `stderr` tras `os_run_command`.
  - Detección de dependencias ausentes (ej. `ModuleNotFoundError`) y auto-instalación silenciosa (`pip install <pkg>`).
  - Hasta 3 reintentos autónomos sin intervención del usuario.
- **Message Queuing & Prioritization**:
  - Encolado de mensajes entrantes cuando el agente está ocupado (`📥 Cola: N`).
  - Cancelación rápida con `Esc` o `/cancel`.
  - Envío prioritario con interrupción inmediata vía `/now <orden>`.
  - Consulta y limpieza de cola vía `/queue` y `/clearqueue`.
- **User Profile & Continuous Memory (Auto-Learning)**:
  - Ficha de perfil persistente en SQLite (`users.profile_md`), inyectada automáticamente en el System Prompt.
  - Memoria continua en `user_memories` con motor de búsqueda semántica y FTS5 BM25.
  - Auto-aprendizaje en segundo plano tras cada turno con `FactExtractor.ExtractAndPersistAsync`.
  - Herramientas nativas del agente: `remember_fact`, `search_memory`, `update_user_profile`.
  - Comandos TUI sobrios y sin emojis: `/profile`, `/memories`, `/remember <hecho>`.
- **Smart Path Resolver**:
  - Resolución inteligente de rutas de usuario: `Desktop` prioritario para archivos de trabajo del usuario, `Documents` para proyectos y repositorios.
  - Protección estricta: nunca resolver rutas del usuario hacia el directorio de trabajo del servidor (`CWD`).

## Development & Execution Commands
- **TUI Interactiva**: `cd backend && go run cmd/ozy/main.go`
- **Compilar Binario**: `cd backend && go build -o ozy.exe ./cmd/ozy/.`
- **Servidor Web**: `cd backend && go run cmd/server/main.go`
- **Tests**:
  - Proveedores: `cd backend && go test ./internal/providers/... -v`
  - Sistema & Paths: `cd backend && go test ./internal/system/... -v`
  - Agente & Tríada: `cd backend && go test ./internal/agent/... -v`
  - TUI & Colas: `cd backend && go test ./internal/tui/... -v`
  - Suite completa: `cd backend && go test ./internal/...`

## Brand Theme Colors
- Primary Brand Color: Vibrant Electric Neon Lime `#d1f107` (`bg-[#d1f107] text-[#181e00] font-bold`).
- Dark Surface Colors: Surface Dim `#131313`, Surface Container `#1e1e1e` / `#222222`.
- Queue Indicator Color: Warm Amber `#f1c40f`.

## Code Principles & Rules
1. **SOLID & DRY**: Código modular, desacoplado y reutilizable.
2. **Robust Error Handling**: Registro claro de errores y recuperación sin caídas del proceso.
3. **Security & Path Safety**: Prevenir path traversal, validar rutas contra carpetas del sistema y ejecutar SQLite en modo solo lectura para consultas exploratorias.
4. **Zero-Docker Constraint**: Mantener la dependencia en librerías Go puras (`modernc.org/sqlite`) sin exigir CGO ni herramientas externas.
