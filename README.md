# OzyAssist

OzyAssist es un asistente inteligente de escritorio y cowork con arquitectura moderna, diseñado para ofrecer control seguro de archivos, ejecución de tareas, integración de herramientas y chat interactivo impulsado por LLMs locales y en la nube.

---

## Características Principales

- **Chat Unificado Inteligente**: Conversación fluida con soporte para renderizado de artefactos de código, bloques interactivos de aprobación de herramientas y razonamiento colapsable (*Thinking*).
- **Control Local y Gestión de Proyectos**: Exploración de código, diffs en tiempo real, operaciones Git nativas y ejecución de terminal.
- **Seguridad Multi-Perfil con PIN**: Gestión de múltiples perfiles locales en SQLite protegidos con hash bcrypt y PIN de 6 dígitos.
- **Proveedores LLM en Tiempo Real**: Conexión dinámica con OpenRouter, Anthropic, OpenAI, DeepSeek y Ollama local con validación en vivo.
- **Ecosistema de Habilidades y Conectores**: Soporte completo para conectores MCP, plugins y habilidades modulares (`SKILL.md`).

---

## Estructura del Proyecto

```
ozyAsis/
├── backend/               # Servidor Go (Gin, WebSocket, SQLite, Providers, Agent Engine)
│   ├── cmd/server/        # Punto de entrada del servidor
│   ├── internal/api/      # Handlers HTTP y WebSocket
│   ├── internal/agent/    # Bucle ReAct y ejecución de herramientas
│   ├── internal/db/       # Migraciones SQLite y consultas
│   └── internal/providers/# Integraciones LLM
├── frontend/              # Aplicación React 18, TypeScript, Vite, Tailwind CSS
│   ├── src/components/    # Componentes modulares (Chat, Auth, Settings, Projects)
│   ├── src/store/         # Estado global con Zustand
│   └── src/services/      # Clientes API y WebSocket
└── docs/                  # Documentación de arquitectura y guías
```

---

## Instalación y Ejecución

### 1. Backend (Go)
```bash
cd backend
go run cmd/server/main.go
```
*El servidor iniciará en `http://localhost:8080`.*

### 2. Frontend (React + Vite)
```bash
cd frontend
npm install
npm run dev
```
*La interfaz estará disponible en `http://localhost:1420/`.*

---

## Licencia
MIT
