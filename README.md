# OzyAssist

OzyAssist es un asistente inteligente de escritorio, desarrollo y cowork autónomo de última generación (inspirado en Claude Desktop, Claude Code y Manus). Integra un backend robusto en Go con persistencia local SQLite, WebSocket streaming reactivo, motor de herramientas de sistema seguras, y un frontend moderno en React 18, TypeScript, Tailwind CSS y Tauri.

---

## ⚡ Capacidades Principales

- **Chat Unificado & Flujo ReAct Inteligente**: Conversación multimodal con streaming ultra-rápido, razonamiento colapsable (*Thinking*), supresión de alucinaciones y ejecución autónoma paso a paso.
- **Auditoría y Exploración Multi-Repo Git Real**: Detección automática de workspaces compuestos (multi-repositorios como `crmgeofal`), inspección de commits verídicos, branches y diffs sin suposiciones.
- **Generación y Lectura Nativa de Excel (`os_create_excel` / `os_read_document`)**: Creación de hojas de cálculo `.xlsx` con estilo profesional (encabezados de marca `#D1F107`, formato numérico y celdas tipadas) y extracción de texto/datos de archivos `.xlsx`, `.xlsm`, `.docx`, `.pdf`.
- **Explorador Seguro de Bases de Datos SQLite (`os_query_db`)**: Consulta de bases de datos locales en modo estricto de solo lectura (`?mode=ro`) con validación léxica de seguridad contra sentencias destructivas (`DROP`, `DELETE`, `UPDATE`, `INSERT`, etc.).
- **Visión Multimodal de Pantalla (`os_analyze_screen`)**: Captura instantánea de la pantalla o ventana activa y análisis visual mediante modelos multimodales (OpenRouter Gemini / GPT-4o / LM Studio).
- **Watchdog Proactivo en Segundo Plano (`os_watchdog`)**: Monitorización continua de puertos de red (TCP), procesos de Windows y servicios críticos con notificaciones nativas del sistema operativo en caso de caída o recuperación.
- **Investigación Web Profunda (`os_deepsearch`)**: Búsqueda en la web en tiempo real con extracción estructurada de fuentes, citas y síntesis de verdad fáctica.
- **Automatización del Sistema Operativo Windows**: Lanzamiento y control seguro de aplicaciones (`os_open_app`, `os_start_process`), redacción de correos con apertura automática en navegador/cliente (`os_open_email`), control de ventanas y portapapeles.
- **Ecosistema de Habilidades (`SKILL.md`) y Conectores MCP**: Protocolo Model Context Protocol (MCP) nativo vía stdio y soporte para habilidades modulares de desarrollo.
- **Arquitectura Tríada (Ozy, Charc, Nine)**: Enrutamiento cognitivo con auto-sanación (*Self-Heal*), bucles de verificación adversarial y memoria persistente.
- **Seguridad Multi-Perfil con PIN**: Perfiles locales cifrados en SQLite con hash bcrypt y PIN de 6 dígitos.

---

## 📁 Estructura del Proyecto

```
ozyAsis/
├── .agents/skills/        # Catálogo de habilidades modulares (loop, backend, deploy, etc.)
├── backend/               # Servidor Go de alto rendimiento
│   ├── cmd/server/        # Entrypoint del servidor WebSocket y API HTTP
│   ├── cmd/ozy/           # CLI interactiva de Ozy
│   ├── internal/agent/    # Bucle ReAct, ejecutor, watchdog, excel, vision, db_query
│   ├── internal/api/      # Handlers REST, WebSocket y router SPA embebido
│   ├── internal/audio/    # Captura de audio y detección de wake-word
│   ├── internal/db/       # Migraciones SQLite, esquemas y consultas
│   ├── internal/mcp/      # Cliente de protocolo Model Context Protocol (MCP)
│   ├── internal/providers/# Conectores LLM (OpenRouter, OpenAI, Cohere, Groq, Ollama)
│   ├── internal/search/   # Motor de DeepSearch con citas y reportes
│   └── internal/system/   # Resolución de rutas, navegadores y telemetría
├── frontend/              # Interfaz de usuario React 18 + Vite + Tailwind CSS
│   ├── src/components/    # Componentes de Chat, Consola, Settings, Voz y Layout
│   ├── src/store/         # Estado reactivo global (Zustand)
│   ├── src/services/      # Conexiones WebSocket y API
│   └── src-tauri/         # Wrapper de escritorio nativo con Rust / Tauri
├── docs/                  # Documentación de arquitectura y guías operativas
└── training/              # Scripts de fine-tuning y generación de datasets para modelos locales
```

---

## 🚀 Instalación y Ejecución

### Requisitos
- **Go**: 1.22+ (con CGO habilitado y GCC en PATH para SQLite)
- **Node.js**: 18+ y npm
- **Windows 10 / 11**

### 1. Backend (Go)
```powershell
cd backend
go run cmd/server/main.go
```
*El servidor HTTP y WebSocket se inicia en `http://localhost:8080`.*

### 2. Frontend (React + Vite)
```powershell
cd frontend
npm install
npm run dev
```
*La interfaz estará disponible en `http://localhost:1420/`.*

### 3. Suite de Pruebas Automatizadas
```powershell
# Backend (Pruebas del agente, db_query, excel_generator, watchdog, etc.)
cd backend
go test ./internal/agent/...
```

---

## 🛡️ Licencia
MIT License.
