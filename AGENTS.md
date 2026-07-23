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

## Code Principles & Rules
1. Follow SOLID and DRY principles.
2. Ensure clear error handling, logging, and state synchronization.
3. Validate user inputs and ensure secure path operations (prevent path traversal).
4. Maintain responsive, clean UI with fully wired interactive elements and clear state feedback.
