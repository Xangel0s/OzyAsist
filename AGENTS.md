# AGENTS.md — OzyAssist Guidelines & Instructions

## Project Overview
OzyAssist is an open-source AI desktop/code/cowork assistant inspired by Claude Desktop, Claude Code, and Cowork environments. It features a Go backend with SQLite, WebSocket streaming, task execution capabilities, and a React + Vite + TypeScript frontend.

## Architecture & Tech Stack
- **Backend**: Go (Gin / gorilla/websocket / SQLite / Qdrant optional / local LLM or API providers)
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, Zustand state management.
- **Desktop Wrapper**: Tauri (optional for desktop build) / Web mode.

## Development & Execution Commands
- **Backend**: `cd backend && go run cmd/server/main.go`
- **Frontend**: `cd frontend && npm run dev`
- **Tests**:
  - Backend: `cd backend && go test ./...`
  - Frontend: `cd frontend && npm run test`

## UI Architecture & Claude Desktop Alignment
- **Navigation & Layout**:
  - `Sidebar.tsx`: Features top mode pill (`Inicio | Code`), brand lime `+ Nuevo` button (`#d1f107`), and navigation (`Proyectos`, `Artefactos`, `Personalizar`).
  - `Personalizar`: Triggers `SettingsModal` directly for centralized management of Skills, Connectors, Plugins, and LLM Providers.
- **Settings Modal & Ecosystem**:
  - `SettingsModal.tsx`: Full settings dialog with 8 categories, MCP Connector creation modal, Skill Upload modal, Inline `SKILL.md` editor, and Marketplace Directory modal.
  - **LLM Providers**: Supports OpenRouter, OpenAI, Anthropic, DeepSeek, and Ollama local host URL with `localStorage` persistence and live connection testing.
- **Brand Theme Colors**:
  - Primary Brand Color: Vibrant Electric Neon Lime `#d1f107` (`bg-[#d1f107] text-[#181e00] font-bold`).
  - Dark Surface Colors: Surface Dim `#131313`, Surface Container `#1e1e1e` / `#222222`.

## Code Principles & Rules
1. Follow SOLID and DRY principles.
2. Ensure clear error handling, logging, and state synchronization.
3. Validate user inputs and ensure secure path operations (prevent path traversal).
4. Maintain responsive, clean UI with fully wired interactive elements and clear state feedback.
