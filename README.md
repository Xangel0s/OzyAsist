# OzyAssist

OzyAssist es un asistente autónomo de escritorio, desarrollo y cowork de última generación para Windows (inspirado en Claude Desktop, Claude Code y Manus). Integra un backend en Go de alto rendimiento, arquitectura **Zero-Docker** con SQLite puro (`modernc.org/sqlite`), memoria híbrida en RAM, terminal con bucle de auto-reparación (*Self-Healing*), interfaz de consola TUI reactiva en **Bubble Tea** con cola de mensajes, y un ecosistema unificado de proveedores LLM.

---

## ⚡ Capacidades Principales

- **Chat Unificado & Flujo ReAct Inteligente**: Conversación multimodal con streaming ultra-rápido, razonamiento colapsable (*Thinking* desplegable con `Ctrl+T` o `/thinking`), supresión de alucinaciones y ejecución autónoma paso a paso.
- **Tríada Cognitiva Nativa (Ozy, Charc, Nine)**: Enrutamiento cognitivo bajo un único modelo LLM:
  - ⚡ **OZY** (Ejecutor): Planifica y ejecuta herramientas del sistema y MCP.
  - 🛡️ **CHARC** (Auditor de Calidad): Valida pre y post-condiciones, auditando resultados antes de darlos por buenos.
  - 🧠 **NINE** (Estratega): Descompone problemas complejos y sintetiza conclusiones accionables.
- **Bucle de Auto-Reparación de Terminal (*Self-Healing Terminal*)**: Diagnóstico automático de fallos en ejecución (`os_run_command`). Si un script o comando falla (ej. `ModuleNotFoundError: No module named 'openpyxl'`, error de sintaxis o políticas de PowerShell):
  - Captura automáticamente `stderr`.
  - Diagnostica la causa raíz y ejecuta silenciosamente `pip install <paquete>` o ajusta la política de ejecución.
  - Reintenta hasta 3 veces de forma completamente autónoma sin interrumpir al usuario.
- **Cola de Mensajes, Priorización y Cancelación Ininterrumpida**:
  - Mientras Ozy procesa una tarea o ejecuta herramientas, puedes seguir escribiendo libremente; tus mensajes se almacenan en una cola visual ordenada (`📥 Cola: N`).
  - **Ejecución Inmediata (`/now <orden>`)**: Interrumpe la tarea actual e inicia la nueva orden de inmediato.
  - **Cancelación Fluida (`Esc` o `/cancel`)**: Cancela la petición en curso de forma instantánea sin cerrar la terminal; `/cancel all` o `/clearqueue` vacía la cola.
- **Resolución Inteligente de Rutas de Usuario (*Smart Path Resolver*)**: Detección y resolución prioritaria de archivos en carpetas naturales (`Desktop` primero para archivos de trabajo, `Documents` para proyectos y repositorios, `Downloads`, `Projects`), evitando que los comandos apunten erróneamente al directorio de trabajo del servidor (`CWD`).
- **Mesa de Trabajo Segura (Sandbox & Staging)**: Carpeta de staging aislada para manipulación segura de documentos, hojas de cálculo y código sin alterar archivos del sistema operativo directamente.
- **Generación y Lectura Nativa de Excel (`os_create_excel` / `os_read_document`)**: Creación de hojas de cálculo `.xlsx` con estilo profesional (encabezados de marca `#D1F107`, formato numérico y celdas tipadas) y extracción de datos de archivos `.xlsx`, `.docx`, `.pdf`.
- **Explorador Seguro de Bases de Datos SQLite (`os_query_db`)**: Consulta de bases de datos locales en modo estricto de solo lectura (`?mode=ro`) con validación léxica de seguridad contra sentencias destructivas (`DROP`, `DELETE`, `UPDATE`, `INSERT`).
- **Auditoría Multi-Repo Git Real**: Detección automática de workspaces compuestos (multi-repositorios como `crmgeofal`), inspección verídica de ramas, commits y diffs.
- **Visión Multimodal de Pantalla (`os_analyze_screen`)**: Captura instantánea de la pantalla o ventana activa y análisis visual mediante modelos multimodales.
- **Watchdog Proactivo en Segundo Plano (`os_watchdog`)**: Monitorización continua de puertos TCP, procesos del sistema y servicios críticos con alertas sonoras y del SO.
- **Investigación Web Profunda (`os_deepsearch`)**: Búsqueda en la web en tiempo real con extracción estructurada de fuentes y citas.
- **Ecosistema de Habilidades (`SKILL.md`) y Conectores MCP**: Compatibilidad nativa con el estándar Model Context Protocol (MCP) vía stdio y archivos `mcp_servers.json`.
- **Motor de Voz Nativo ("Hey Ozy")**: Activación por palabra clave en segundo plano (*wake-word*), transcripción ultra-rápida (Groq Whisper ~150ms) y controlador de audio con watchdog.
- **Arquitectura Zero-Docker**: 100% libre de Docker. SQLite en Go puro (`modernc.org/sqlite`, `CGO_ENABLED=0`) con embeddings y caché en memoria RAM.

---

## 🤖 Proveedores LLM Compatibles

OzyAssist soporta configuración dinámica de claves en caliente (guardadas en `.env`) y cambio de modelos sin reiniciar la aplicación:

| Proveedor | Comando de Configuración | Modelo por Defecto | Notas |
|---|---|---|---|
| **Mistral AI** | `/key mistral <api-key>` | `mistral-large-latest` | Endpoint OpenAI compatible (`https://api.mistral.ai/v1`). Modelos: `codestral-latest`, `open-mixtral-8x22b`, `open-mistral-nemo`. |
| **KiloCode Gateway** | `/key kilocode <jwt-token>` | `kilo/anthropic/claude-sonnet-4-5` | Gateway unificado con +500 modelos (`https://api.kilo.ai/api/gateway`). Soporta `kilo/openai/gpt-4o`, `kilo/google/gemini-2-5-pro`, etc. |
| **Cohere** | `/key cohere <api-key>` | `command-r-plus-08-2024` | Excelente para razonamiento agéntico, multilingüe y estructuración. |
| **Groq** | `/key groq <api-key>` | `llama-3.3-70b-versatile` | Ultra-rápido (~800 tok/s). Activa automáticamente el motor de voz STT. |
| **OpenAI** | `/key openai <api-key>` | `gpt-4o` | Modelos oficiales de OpenAI y Whisper STT. |
| **OpenRouter** | `/key openrouter <api-key>` | `deepseek/deepseek-chat` | Acceso a cientos de modelos en la nube. |
| **Anthropic** | `/key anthropic <api-key>` | `claude-3-5-sonnet-20241022` | API oficial de Anthropic. |
| **DeepSeek / OpenCode** | `/key deepseek <api-key>` | `deepseek-chat` | Modelos de razonamiento profundo y código. |
| **Local (LM Studio / Ollama)** | `/key local <url>` | `local-model` | Inferencia 100% privada y local sin conexión a internet (`http://localhost:1234` o `11434`). |

---

## 📁 Estructura del Proyecto

```
Ozyasist/
├── .agents/skills/             # Catálogo de habilidades modulares (.md)
├── backend/                    # Núcleo en Go de alto rendimiento
│   ├── cmd/
│   │   ├── ozy/                # Entrypoint de la CLI / TUI interactiva (Bubble Tea)
│   │   ├── server/             # Entrypoint del servidor WebSocket y API REST
│   │   └── ozyctl/             # Herramienta de control y administración
│   ├── internal/
│   │   ├── agent/              # Bucle ReAct, Tríada Cognitiva, Watchdog, Excel, Self-Healing
│   │   ├── api/                # Handlers HTTP, WebSocket y streaming reactivo
│   │   ├── audio/              # Captura de micrófono y detección de wake-word
│   │   ├── browser/            # Automatización y navegación de Chrome
│   │   ├── db/                 # Base de datos SQLite pura y modelos
│   │   ├── mcp/                # Cliente nativo de Model Context Protocol (stdio/JSON-RPC)
│   │   ├── memory/             # Memoria híbrida (SQLite + RAM) y embeddings
│   │   ├── providers/          # Conectores LLM (Mistral, KiloCode, Groq, Cohere, OpenAI, etc.)
│   │   ├── search/             # Motor DeepSearch de investigación web
│   │   ├── security/           # Multi-perfil, PIN de 6 dígitos y hashing
│   │   ├── system/             # Smart Path Resolver, control de SO y procesos
│   │   ├── tui/                # Interfaz Bubble Tea (model, view, update, colas de mensajes)
│   │   └── voice/              # Motor de voz y pipeline STT/TTS
│   └── go.mod
├── docs/                       # Documentación de arquitectura y guías
├── training/                   # Scripts de fine-tuning y datasets locales
├── AGENTS.md                   # Reglas operativas y convenciones de agentes
└── README.md                   # Documentación principal
```

---

## 🚀 Instalación y Ejecución

### Requisitos
- **Go**: 1.22 o superior (compilación Go pura, no requiere GCC ni Docker)
- **Windows 10 / 11**

### 1. Iniciar la Interfaz TUI Interactiva
```powershell
cd backend
go run cmd/ozy/main.go
# O compilar el binario optimizado:
go build -o ozy.exe ./cmd/ozy/.
.\ozy.exe
```

### 2. Iniciar el Servidor API / WebSocket (opcional para integraciones externas)
```powershell
cd backend
go run cmd/server/main.go
```
*El servidor se inicia en `http://localhost:8080` con soporte WebSocket en `/ws/chat`.*

---

## ⌨️ Atajos y Comandos de la TUI

| Comando / Atajo | Función |
|---|---|
| `Enter` | Enviar mensaje (o encolar si Ozy está ocupado) |
| `Esc` o `/cancel` | Cancelar inmediatamente la tarea en curso |
| `/cancel all` | Cancelar la tarea actual y vaciar la cola |
| `/now <orden>` | Interrumpir la tarea activa y ejecutar la orden de inmediato |
| `/queue` o `/cola` | Mostrar la lista de mensajes encolados |
| `/clearqueue` | Vaciar la cola de mensajes pendientes |
| `Ctrl + T` o `/thinking`| Alternar visibilidad del hilo de razonamiento (*Thinking*) |
| `Ctrl + C` | Cancelar tarea activa (o salir si está inactivo) |
| `Ctrl + L` o `/clear` | Limpiar la pantalla de la conversación |
| `/provider [nombre]` | Consultar o cambiar el proveedor LLM activo |
| `/model [nombre]` | Cambiar el modelo de IA utilizado |
| `/key <prov> <clave>` | Configurar API Key o token en caliente (guardada en `.env`) |
| `/voice` | Alternar la escucha por voz ("Hey Ozy") |
| `/tools` | Listar todas las herramientas activas (Sistema + MCP) |
| `/mcp [reload]` | Ver estado o recargar servidores MCP en caliente |

---

## 🧪 Pruebas Automatizadas

```powershell
cd backend

# Pruebas de proveedores LLM (Mistral, KiloCode, Cohere, Groq, etc.)
go test ./internal/providers/... -v

# Pruebas del sistema y Path Resolver
go test ./internal/system/... -v

# Pruebas de la Tríada Cognitiva y bucle ReAct
go test ./internal/agent/... -v

# Pruebas del sistema de colas de la TUI
go test ./internal/tui/... -v
```

---

## 🛡️ Licencia
MIT License.
