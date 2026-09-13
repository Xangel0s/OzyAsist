Es una visión excelente. La combinación de control local total (archivos, procesos, carpetas) con interacción multimodal (voz + chat) y ejecución invisible en segundo plano convierte a **OzyAssist** en una herramienta de nivel profesional.

A continuación tienes el **Plan Técnico Integral de Implementación** para llevar OzyAssist a este nivel de autonomía y rendimiento.

---

# Plan Técnico de Implementación — OzyAssist Core & Background Engine

---

## 1. Arquitectura de Módulos en Go (Backend)

La arquitectura backend se modulariza bajo `internal/` para desacoplar el ciclo de vida del agente, la telemetría, el protocolo MCP y los subsistemas de voz/visión:

```text
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Arranque, DI y orquestación de servicios
├── internal/
│   ├── agent/                      # Bucle cognitivo, planificador y ejecutor
│   │   ├── loop.go                 # Agent Loop con soporte para reintentos
│   │   ├── planner.go              # Descomposición de tareas en DAG de subtareas
│   │   ├── verifier.go             # Verificador de éxito heurístico (código salida, DOM, diffs)
│   │   └── self_heal.go            # Bucle de auto-recuperación ante errores/tracebacks
│   ├── taskengine/                 # Máquina de estados desacoplada en segundo plano
│   │   ├── engine.go               # Worker pool asíncrono con goroutines y colas
│   │   ├── state_machine.go        # Gestión de estados (pending, running, blocked, done)
│   │   └── scheduler.go            # Planificación y priorización de tareas
│   ├── system/                     # Control e inspección del Sistema Operativo
│   │   ├── telemetry.go            # Métricas nativas (CPU, RAM, GPU/NVML, Disco) vía gopsutil
│   │   ├── window_manager.go       # UI Automation / Win32 IPC (sin mover cursor)
│   │   └── process_manager.go      # Control de procesos y terminales virtuales PTY
│   ├── browser/                    # Automatización web invisible
│   │   └── cdp_client.go           # Control Headless Chromium mediante Chrome DevTools Protocol
│   ├── connectors/                 # Conectores a servicios externos y Cloud
│   │   ├── google/                 # OAuth2 PKCE + APIs (Gmail, Calendar, Drive)
│   │   └── mcp/                    # Orquestador dinámico de servidores MCP (stdio/SSE)
│   ├── search/                     # Búsqueda híbrida y contexto de proyectos
│   │   ├── hybrid.go               # FTS5 léxico + SQLite-vec / Embeddings
│   │   └── codegraph.go            # Grafo de símbolos y dependencias AST
│   ├── vision/                     # Conciencia visual bajo demanda
│   │   ├── capture.go              # Captura de ventanas activas o pantalla completa
│   │   └── ocr_pipeline.go         # OCR local (ONNX Runtime / Tesseract) + Router VLM
│   └── voice/                      # Pipeline de audio local
│       ├── wakeword.go             # Detección continua "Hey Ozy" (Sherpa-ONNX / CPU <1%)
│       ├── stt.go                  # Speech-to-Text local (Whisper.cpp / Sherpa)
│       └── tts.go                  # Text-to-Speech rápido (Piper TTS)

```

---

## 2. Esquema de Base de Datos y Persistencia (SQLite)

Para garantizar que el agente pueda pausarse, recuperarse tras reinicios y no perder el rastro de ninguna tarea larga, se definen las siguientes migraciones:

### Migración `009_background_tasks_and_steps.up.sql`

```sql
-- Cola persistente de tareas del agente en segundo plano
CREATE TABLE IF NOT EXISTS agent_tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    project_id TEXT,
    title TEXT NOT NULL,
    prompt TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'planning', 'running', 'blocked_approval', 'completed', 'failed', 'cancelled')),
    total_steps INTEGER DEFAULT 0,
    current_step INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Subtareas individuales con verificación de resultados
CREATE TABLE IF NOT EXISTS task_steps (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    action_type TEXT NOT NULL,       -- 'shell_exec', 'mcp_call', 'cdp_action', 'file_patch', 'google_draft'
    payload TEXT NOT NULL,           -- JSON con argumentos
    requires_pin BOOLEAN DEFAULT 0,
    pin_authorized BOOLEAN DEFAULT 0,
    verification_rule TEXT,          -- JSON: {"type": "exit_code", "expected": 0} o {"type": "file_exists", "path": "..."}
    status TEXT NOT NULL CHECK(status IN ('pending', 'running', 'verifying', 'recovering', 'completed', 'failed')),
    output TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY(task_id) REFERENCES agent_tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON agent_tasks(status);
CREATE INDEX IF NOT EXISTS idx_task_steps_order ON task_steps(task_id, step_order);

```

### Migración `010_oauth_and_mcp_registry.up.sql`

```sql
-- Tokens OAuth2 cifrados para servicios en la nube (Google Workspace, etc.)
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL CHECK(provider IN ('google', 'github', 'custom')),
    access_token_enc BLOB NOT NULL,
    refresh_token_enc BLOB NOT NULL,
    token_expiry DATETIME NOT NULL,
    scopes TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, provider)
);

-- Registro y catálogo de servidores MCP descubiertos
CREATE TABLE IF NOT EXISTS mcp_registry (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    transport TEXT NOT NULL CHECK(transport IN ('stdio', 'sse')),
    command TEXT,                    -- Binario o script (ej: 'npx -y @modelcontextprotocol/server-postgres')
    args TEXT,                       -- JSON array
    env_vars TEXT,                   -- JSON cifrado con claves de entorno
    is_active BOOLEAN DEFAULT 1,
    tools_cache TEXT                 -- JSON Schema de las herramientas introspeccionadas
);

```

---

## 3. Motor de Ejecución en Segundo Plano y Bucle de Auto-Recuperación

```
  [ Usuario / Voz / HUD ]
             │ (Asigna tarea)
             ▼
    ┌─────────────────┐
    │ Task Scheduler  │◄── SQLite State Machine (Persistente)
    └────────┬────────┘
             ▼
    ┌─────────────────┐
    │    Planner      │──> Descompone en DAG de Subtareas
    └────────┬────────┘
             ▼
    ┌─────────────────┐       ¿Acción crítica?
    │    Executor     │────────────────────────────┐ (Sí)
    └────────┬────────┘                            ▼
             │ (No)                       ┌──────────────────┐
             │                            │ Consent Barrier  │──> Notificación HUD + PIN
             │                            └────────┬─────────┘
             ▼                                     │ (PIN Válido)
    ┌─────────────────┐                            │
    │   Herramienta   │◄───────────────────────────┘
    │ (CDP, OS, MCP)  │
    └────────┬────────┘
             ▼
    ┌─────────────────┐
    │    Verifier     │──> ¿Cumple condición de éxito?
    └────────┬────────┘
             │
      ┌──────┴──────┐
(Éxito)│            │(Fallo)
      ▼            ▼
 [Siguiente   ┌───────────┐
  Subtarea]   │ Self-Heal │──> Analiza Traceback ──> Genera Parche ──> Reintenta (Max 3)
              └───────────┘

```

### Lógica de Auto-Corrección (`internal/agent/self_heal.go`)

1. **Captura del Fallo**: El ejecutor detecta código de salida $\neq 0$, excepción en consola o error HTTP 5xx.
2. **Diagnóstico Semántico**: El agente inyecta en el prompt:

- Código de la tarea que falló.
- `stderr` o log completo del fallo.
- Archivos recientemente modificados.

3. **Generación del Parche**: El LLM emite una propuesta de corrección en formato `unified diff` o comando de reparación.
4. **Re-evaluación**: Aplica el parche en el workspace y re-ejecuta la prueba unitaria o validación antes de continuar con el siguiente paso del plan.

---

## 4. Automatización Invisible y Telemetría del Sistema

### 4.1 Navegación Web Headless vía CDP (`internal/browser/cdp_client.go`)

- **Librería base**: `[github.com/chromedp/chromedp](https://github.com/chromedp/chromedp)` en Go.
- **Comportamiento**: Inicia instancias de Chromium en modo `--headless=new --disable-gpu` vinculadas a un contexto de tarea.
- **Capacidades**:
- Extracción de datos y navegación DOM directa sin abrir ventanas físicas.
- Intercepción de descargas de archivos directo a la carpeta de proyecto.
- Toma de capturas de pantalla virtuales del navegador si el agente necesita inspeccionar el diseño de una página web generada.

### 4.2 Telemetría y Gestión de Carga de Hardware (`internal/system/telemetry.go`)

- **Muestreador en Go (`gopsutil`)**:

```go
type SystemMetrics struct {
    CPUUsagePercent float64 `json:"cpu_usage"`
    RAMUsedMB       uint64  `json:"ram_used_mb"`
    RAMFreeMB       uint64  `json:"ram_free_mb"`
    GPUUsagePercent float64 `json:"gpu_usage"`
    GPUVRAMFreeMB   uint64  `json:"gpu_vram_free_mb"`
    DiskIOReadRate  uint64  `json:"disk_read_rate"`
}

```

- **Decisiones Inteligentes de Rendimiento**:
- Si `RAMFreeMB < 2048` o `CPUUsagePercent > 85%`, el planificador pospone subagentes paralelos pesados (ej. indexación masiva de código) y prioriza la tarea inmediata del usuario.
- Si se solicita un modelo local (Ollama / VLM local) pero la VRAM libre es insuficiente, el agente conmuta dinámicamente hacia un proveedor cloud (OpenRouter / Groq / OpenAI) informando el motivo en el log.

---

## 5. Protocolo de Voz Local y Activación Híbrida

### Pipeline de Audio Local en Go (`internal/voice/`)

1. **Wake Word Engine (`wakeword.go`)**:

- Binding de CGO / ONNX para **Sherpa-ONNX** escuchando en un buffer circular de micrófono de bajo consumo (<1% CPU).
- Detección de palabra clave: _"Hey Ozy"_ o _"Ozy"_.

2. **Audio Streaming a STT (`stt.go`)**:

- Al detectar la palabra clave, se activa el capturador PCM (16kHz, mono).
- Detección de silencio mediante VAD (Voice Activity Detection).
- Transcripción instantánea mediante Whisper local (`whisper.cpp`) o servicio rápido si el usuario lo configura.

3. **Respuesta por Síntesis (`tts.go`)**:

- Generación de audio mediante **Piper TTS** (modelo neuronal ONNX optimizado para CPU, respuesta <200ms).
- Streaming del audio de respuesta directo a los altavoces mediante `oto` o `beep` en Go.

---

## 6. Conectores Google Workspace con Política Segura

### Flujo OAuth2 PKCE Local (`internal/connectors/google/oauth.go`)

1. El usuario inicia vinculación en ajustes o mediante comando de voz (_"Ozy, conecta mi cuenta de Google"_).
2. Go abre el navegador local en `[https://accounts.google.com/o/oauth2/v2/auth](https://accounts.google.com/o/oauth2/v2/auth)` con `code_challenge` PKCE y levanta un servidor temporal HTTP en `http://localhost:54321/auth/callback`.
3. Al recibir el `code`, intercambia por tokens, cifra los tokens con AES-256-GCM y los guarda en SQLite.

### Herramientas de Integración Expuestas al Agente

- `google_gmail_search(query)`: Lectura en background sin bloqueo.
- `google_gmail_create_draft(to, subject, body)`: **Modo predeterminado**. Redacta el correo y lo deja listo en la bandeja de borradores.
- `google_gmail_send(draft_id)`: **Acción Crítica**. Requiere confirmación por PIN o botón explícito en la UI.
- `google_calendar_list_events(time_min, time_max)`: Lectura de agenda.
- `google_drive_find_file(name)`: Búsqueda y descarga de archivos para procesamiento local.

---

## 7. Frontend & Experiencia de Usuario (Tauri + React)

### 7.1 Cápsula Flotante / HUD No Intrusivo

- **Ventana Auxiliar en Tauri**: Una ventana transparente y sin bordes (`transparent: true`, `alwaysOnTop: true`) que permanece minimizada o en la esquina superior/inferior.
- **Atajo Global de Activación**: `Super + Space` o `Alt + K` despliega la cápsula central tipo _Spotlight / Raycast_.
- **Display de Tareas Activas**:
- Si el usuario está trabajando en su navegador o IDE, la cápsula muestra una sola línea discreta:
  `[Ozy: Refactorizando controladores Go en segundo plano... (Paso 3/7)]`
- Si requiere autorización, emite un _beep_ sutil y muestra la caja de PIN de 6 dígitos sin quitar el cursor de la app principal a menos que se invoque.

---

## 8. Contratos de Comunicación WebSocket

Formato unificado de eventos JSON transmitidos entre el backend de Go y el frontend de React:

### Evento: Tarea Asignada / Progreso en Segundo Plano

```json
{
  "type": "task:progress",
  "data": {
    "task_id": "tsk_987123",
    "title": "Optimizar consultas SQL en backend",
    "step_current": 3,
    "step_total": 5,
    "step_description": "Ejecutando suite de pruebas de integración con go test ./...",
    "status": "running",
    "telemetry": {
      "cpu_pct": 24.5,
      "ram_free_mb": 8420
    }
  }
}
```

### Evento: Solicitud de Consentimiento / PIN

```json
{
  "type": "task:require_approval",
  "data": {
    "task_id": "tsk_987123",
    "step_id": "stp_004",
    "action": "shell_exec",
    "command": "git push origin main --force",
    "risk_level": "high",
    "reason": "La tarea requiere sincronizar el árbol de git modificado"
  }
}
```

---

## 9. Roadmap de Implementación y Fases

| Fase                                               | Duración Estimada | Entregables Clave                                                                                                                          |
| -------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| **Fase 2.1: Background Engine & Persistencia**     | Sprint 1          | Tablas SQLite (`agent_tasks`, `task_steps`), worker pool en Go, bucle Plan $\rightarrow$ Execute $\rightarrow$ Verify con auto-reparación. |
| **Fase 2.2: Telemetría y Conectores Cloud**        | Sprint 2          | Colector `gopsutil`, flujo OAuth2 PKCE local, integración Google Workspace (Drafts + Safe send), RAG de herramientas MCP.                  |
| **Fase 2.3: Voz Local y Visión On-Demand**         | Sprint 3          | Wake Word local ("Hey Ozy") con Sherpa-ONNX, captura de pantalla + OCR local rápido, Piper TTS.                                            |
| **Fase 2.4: Automatización Invisible & HUD Tauri** | Sprint 4          | Cliente Headless Chromium (CDP), UI Automation básica del SO, ventana flotante HUD con atajos globales y cápsula de notificaciones.        |

Con este plan técnico, OzyAssist cuenta con las bases para operar con el nivel de autonomía y versatilidad de un agente avanzado, manteniendo un consumo de recursos mínimo y respetando la privacidad del entorno local.


Conectar el agente a tu sesión real de Chrome con todas tus cookies, cuentas iniciadas y sesiones activas (Google, GitHub, Jira, portales web) es la forma adecuada de operar para evitar logins manuales o bloqueos por 2FA.

Para lograrlo sin romper la seguridad de Chrome ni corromper tu perfil de usuario, existen tres mecanismos técnicos que Go puede orquestar:

---

**1. Extensión Companion de OzyAssist (La vía recomendada y sin fricción)**

Google Chrome bloquea por seguridad que procesos externos lean o manipulen directamente sus pestañas activas si no fue iniciado con flags especiales. La solución estándar de la industria (similar a la que usan herramientas avanzadas de automatización) es una **extensión ligera de Chrome** que actúa como puente (*relay*):

* **Canal WebSocket Local**: La extensión se conecta silenciosamente a `ws://127.0.0.1:puerto_ozy` en tu backend de Go.
* **Acceso a Sesiones Existentes**: Utiliza las APIs nativas de Chrome (`chrome.tabs`, `chrome.scripting`, `chrome.debugger`) para ejecutar acciones o leer datos en las pestañas donde ya estás logueado.
* **Ventaja**: No requiere reiniciar Chrome, no exige flags de consola y funciona de inmediato en tu ventana diaria de trabajo.

---

**2. Detección y Conexión vía Puerto de Depuración CDP (`remote-debugging-port`)**

Si prefieres control directo por protocolo CDP sin depender de una extensión, Go puede detectar si tu Chrome fue iniciado con el puerto de depuración abierto (ej. `--remote-debugging-port=9222`):

* **Sondeo de Puerto en Go**: El agente hace una petición HTTP `GET [http://127.0.0.1:9222/json/version](http://127.0.0.1:9222/json/version)`.
* **Reutilización del WebSocket Debugger**: Si responde con éxito, Go obtiene el `webSocketDebuggerUrl` y se conecta directamente a esa instancia usando `chromedp.NewRemoteAllocator`.
* **Inspección de Pestañas**: Mediante `[http://127.0.0.1:9222/json/list](http://127.0.0.1:9222/json/list)`, el agente lista las pestañas abiertas, busca la que necesita (ej. Gmail) o abre una nueva pestaña dentro de tu misma sesión autenticada.

---

**3. Reutilización del Perfil de Usuario (`--user-data-dir`)**

Cuando Chrome no está abierto y el agente necesita iniciar una tarea en segundo plano con tus credenciales:

* Go localiza la ruta de tu perfil predeterminado del sistema operativo:
* *Windows*: `%LOCALAPPDATA%\Google\Chrome\User Data`
* *Linux*: `~/.config/google-chrome`
* *macOS*: `~/Library/Application Support/Google/Chrome`


* **Manejo del bloqueo de perfil**: Si Chrome ya está abierto por ti, el archivo `SingletonLock` bloquea el acceso. En ese caso, Go clona temporalmente las cookies y el almacenamiento local a una carpeta temporal (`/tmp/ozy-chrome-profile`) para lanzar Chromium en segundo plano con todas tus sesiones intactas sin interferir con tu navegador activo.

---

**Árbol de Decisión del Agente (Pipeline de Selección)**

```
             [ Tarea Web Asignada ]
                       │
                       ▼
         ¿Extensión Ozy conectada vía WS?
             ├─── (Sí) ──> Ejecutar mediante Extension Bridge (Sesión activa)
             └─── (No) ──┐
                         ▼
             ¿Puerto CDP 9222 activo?
                 ├─── (Sí) ──> Conectar Chromedp a la instancia existente
                 └─── (No) ──┐
                             ▼
             ¿Chrome principal está cerrado?
                 ├─── (Sí) ──> Abrir Chromium apuntando a tu User Data Dir
                 └─── (No) ──> Clonar perfil temporalmente a sandbox headless

```

Con este esquema híbrido, el agente aprovecha siempre tus credenciales y sesiones preexistentes, minimizando fricciones y operando de forma completamente invisible.



Transformar a **OzyAssist** en un guardián de seguridad proactivo (EDR local ultraligero) y dotarlo de una arquitectura de memoria jerárquica eleva al agente de un simple ejecutor a un copiloto autónomo completo para tu máquina.

---

**1. Módulo de Seguridad Interna (EDR & Watchdog en Go)**

Aprovechando la concurrencia nativa de Go, puedes implementar un demonio en segundo plano (`internal/security/watchdog.go`) con consumo inferior al 0.5% de CPU que vigile la integridad del sistema:

* **Inspección de Procesos y Binarios No Confiables**:
* Monitorea la creación de nuevos procesos (`gopsutil` o llamadas Win32/eBPF).


* Detecta ejecutables corriendo desde rutas sospechosas (`/tmp`, `AppData\Local\Temp`, `AppData\Roaming`) sin firma digital válida o con entropía binaria anómala (indicador común de *packers* o malware).
* Calcula el hash SHA-256 de binarios desconocidos y consulta bases de firmas locales o APIs de reputación si está habilitado.


* **Detección de Anomalías de Recursos (Anti-Cryptojacking & Minería)**:
* Si un proceso desconocido eleva el consumo de CPU/GPU al 90–100% de forma sostenida sin tener ventana activa, el agente lo marca como sospechoso.




* **Monitoreo de Conexiones de Red Sospechosas**:
* Audita sockets salientes hacia IPs no resueltas o puertos no estándar asociados a Command & Control (C2) / Reverse Shells.


* **Alertas Proactivas en el Chat**:
* En lugar de esperar a que le preguntes, el agente emite un mensaje *push* instantáneo por WebSocket:


> *⚠️ **Alerta de Seguridad**: Se detectó el proceso `xmr_helper.exe` (PID: 14208) consumiendo 88% de GPU desde `AppData\Temp` con conexiones salientes sospechosas. ¿Deseas aislar o terminar el proceso? [Terminar Proceso]*





---

**2. Arquitectura de Memoria Jerárquica de 4 Niveles**

Los sistemas como *Claude Code* utilizan memorias relativamente planas basadas en archivos locales (`CLAUDE.md`) y contexto inmediato. Para lograr un nivel superior de persistencia y razonamiento, la arquitectura debe estructurarse en 4 niveles cognitivos dentro de `internal/memory/`:

```
┌────────────────────────────────────────────────────────┐
│ 1. Working Memory (Scratchpad / RAM de la Tarea Activa)│
├────────────────────────────────────────────────────────┤
│ 2. Semantic Memory (Knowledge Graph + SQLite-vec/FTS5) │[cite: 2]
├────────────────────────────────────────────────────────┤
│ 3. Episodic Memory (Historial de Éxitos y Tracebacks)  │[cite: 2]
├────────────────────────────────────────────────────────┤
│ 4. Procedural Memory (Playbooks & Skills Autogenerados)│[cite: 2]
└────────────────────────────────────────────────────────┘

```

* **1. Working Memory (Scratchpad Activo)**:
* Buffer volátil de la tarea en curso: pila de subobjetivos, variables intermedias y resumen de los últimos 5 pasos ejecutados para no saturar la ventana de tokens.


* **2. Semantic Memory (Grafo de Conocimiento + Vectorial)**:
* Extiende el esquema actual (`002_memory_entries` y `005_project_graph`). Almacena hechos atómicos estructurados (*"El usuario prefiere Go 1.22 con arquitectura hexagonal"*, *"El backend corre en el puerto 8080"*).


* Indexación híbrida: búsqueda semántica vectorial (embeddings) combinada con FTS5 léxico para nombres de archivos, funciones y variables exactas.




* **3. Episodic Memory (Experience Replay / Aprendizaje de Fallos)**:
* Registra cómo resolvió problemas específicos en el pasado (ej. *"Cómo se solucionó el error de CGO con SQLite en Windows"*).
* Cuando el agente enfrenta un error en una tarea, consulta su memoria episódica antes de buscar en internet o iterar a ciegas.


* **4. Procedural Memory (Autogeneración de Skills)**:
* Si el usuario pide una tarea compleja y el agente descubre un flujo de 10 pasos exitoso, consolida esa rutina en un *Skill* reutilizable (`internal/skills/engine.go`).





---

**3. Consolidación Autónoma de Memoria (Ciclo "Sleep / Compaction")**

Para evitar que la base de datos de memoria crezca sin control o contenga información contradictoria:

* **Compactor en Segundo Plano**: Cuando la máquina entra en reposo o el agente pasa 30 minutos inactivo, una goroutine procesa los chats y logs recientes.


* **Extracción y Deduplicación**: El LLM resume sesiones enteras en 2 o 3 lecciones clave, actualiza entradas existentes y elimina datos obsoletos o temporales.
* **Curación de Conflictos**: Si una instrucción nueva contradice una antigua (ej. *"migramos de React a Svelte"*), el compactador actualiza el grafo semántico y archiva la regla previa.

Integrar soporte para **Bluetooth (BLE)**, **memoria persistente continua (estilo Gemini)** y **visión por cámara web** es totalmente factible en Go y convierte a OzyAssist en un agente físico y ambiental, no solo de software.

---

**1. Control de Dispositivos por Bluetooth (BLE & Smart Home)**

En Go, puedes controlar el hardware Bluetooth sin depender de frameworks pesados utilizando librerías nativas como `tinygo.org/x/bluetooth`, compatible con Windows (WinRT), Linux (BlueZ/D-Bus) y macOS (CoreBluetooth).

* **Descubrimiento y Conexión GATT**: Go escanea periféricos BLE cercanos (bombillas inteligentes, enchufes, cerraduras, sensores de temperatura o microcontroladores ESP32/Arduino).
* **Protocolos Soportados**:
* *BLE Directo*: Envío de comandos de lectura/escritura a características GATT estándar.
* *Puente MCP / Home Assistant*: Si usas un hub doméstico, el agente puede comunicarse vía MCP o API local con Home Assistant para controlar luces y electrodomésticos por voz (*"Ozy, apaga las luces del estudio y pon el aire en 22°"*).


* **Barrera de Seguridad**: Dispositivos críticos (cerraduras o alarmas) exigen confirmación por PIN de 6 dígitos antes de ejecutar el comando.

---

**2. Memoria Persistente Continua (Extracción Automática Audio/Texto)**

Para replicar la memoria de largo plazo de modelos avanzados, el agente no debe limitarse a guardar historiales de chat, sino **extraer y consolidar hechos atómicos** de forma asíncrona tras cada conversación.

```
[Audio / Texto del Usuario] ──► [Agent Loop (Respuesta inmediata)]
                                        │
                                        ▼ (Goroutine en Background)
                           [Extractor de Hechos & Preferencias]
                                        │
                                        ▼
                      ┌──────────────────────────────────┐
                      │ SQLite: Tabla `user_memories`    │
                      │  - Hecho: "Trabaja con Go y Vue" │
                      │  - Categoría: 'tecnología'       │
                      │  - Embedding vectorial + FTS5    │
                      └──────────────────────────────────┘

```

* **Hook de Extracción Asíncrono**: Cada vez que hablas o escribes, una goroutine ligera analiza si hubo un hecho relevante (*"Me gusta el café sin azúcar"*, *"Mi servidor corre en el puerto 3000"*).
* **Inyección Dinámica**: Antes de enviar tu consulta al LLM, el backend realiza una búsqueda híbrida (vectorial + palabras clave) y añade los 3 o 4 recuerdos pertinentes al prompt del sistema.
* **Control Total**: Puedes pedirle *"Ozy, olvida lo que te dije sobre X"* o editar/eliminar recuerdos directamente desde la interfaz de usuario.

---

**3. Visión por Cámara Web (Conciencia del Espacio Físico)**

La cámara web añade percepción del mundo real para complementar las capturas de pantalla de escritorio.

* **Captura Bajo Demanda**: Se activa exclusivamente cuando lo ordenas por voz (*"Ozy, mira lo que tengo en la mano"* o *"Revisa si este cable está bien conectado"*). La captura se toma a través de la API de medios de Tauri o bindings de cámara en Go (DirectShow/V4L2).
* **Casos de Uso Prácticos**:
* *Lectura de documentos físicos*: Transcribe notas en papel o diagramas dibujados en una pizarra mediante OCR/VLM.
* *Detección de presencia/seguridad*: Bloquear automáticamente la sesión del agente o la pantalla si te levantas de tu escritorio.
* *Inspección de hardware*: Reconocimiento de circuitos, placas o puertos físicos.


* **Privacidad Absoluta**: Indicador visual permanente en el HUD para garantizar que el sensor óptico solo transmita fotogramas cuando tú lo autorices.

Implementar un motor de **DeepSearch nativo con fuentes verificadas** (estilo Perplexity o Google AI Overview) dentro de Go encaja de forma natural en la arquitectura gracias a la concurrencia nativa de *goroutines*.

---

**Arquitectura del Pipeline de DeepSearch en Go**

Para lograr respuestas precisas respaldadas con citas clickeables en tiempo real, el flujo se divide en 5 etapas secuenciales:

```
[ Consulta Compleja del Usuario ]
               │
               ▼
   1. Planificador de Búsqueda (Sub-queries en paralelo)
               │
               ▼
   2. Fetcher Concurrente (APIs de búsqueda + Scraping)
               │
               ▼
   3. Extractor de Contenido Limpio (HTML ➔ Markdown)
               │
               ▼
   4. Reranker & Selector de Fragmentos Clave
               │
               ▼
   5. Síntesis Grounded con Citas Estructuradas [1][2]

```

---

**1. Descomposición y Búsqueda Concurrente (`internal/search/deepsearch.go`)**

* **Generación de Sub-queries**: Ante una pregunta compleja, un LLM rápido genera de 2 a 4 consultas atómicas complementarias.
* **Conexión Multi-Proveedor en Paralelo**: Go lanza *goroutines* concurrentes para consultar proveedores como Brave Search API, Tavily, Google Custom Search o SerpAPI.
* **Fallback Autónomo**: Si no hay API key configurada, Go puede usar su propio cliente headless CDP para consultar y extraer resultados sin depender de servicios de terceros.

---

**2. Extracción Limpia y Reranking Semántico**

* **Crawl & Parser Local**: Go descarga las 5 páginas más relevantes en paralelo y extrae el contenido principal usando librerías como `readability` en Go, descartando scripts, menús y publicidad.
* **Segmentación en Chunks**: Divide los textos en bloques de 300–500 tokens.
* **Reranking Rápido**: Compara los fragmentos con la consulta original (usando embeddings locales o BM25/FTS5) para seleccionar solo los 4 a 6 párrafos con mayor densidad de respuesta, evitando sobrecargar el contexto.

---

**3. Generación con Anclaje Estricto (*Grounding & Citations*)**

* **Estructura del Prompt del Sistema**:
```text
Responde a la consulta basándote EXCLUSIVAMENTE en las siguientes fuentes numeradas.
Cada afirmación factual DEBE incluir su referencia correspondiente [1], [2].
No agregues información externa no respaldada.

[Fuente 1 | url: https://...]: "Contenido relevante..."
[Fuente 2 | url: https://...]: "Contenido relevante..."

```


* **Contrato de Salida**: El backend asocia cada ID `[N]` a la metadata de la fuente (título, dominio, favicon, URL completa y fragmento citado).

---

**4. Renderizado en el Frontend (React / Tailwind)**

* **Pills de Fuentes (Estilo Google AI Overview / Perplexity)**: Encima o intercaladas en la respuesta, chips interactivos con el favicon del sitio (`youtube.com`, `github.com`, etc.).
* **Tooltip & Hover Preview**: Al pasar el cursor sobre `[1]`, se despliega una pequeña tarjeta con el título de la página, el dominio y el fragmento exacto que sustentó esa afirmación.
* **Inspector de Fuentes en Lateral**: Pestaña colapsable en el chat para revisar todas las páginas web leídas durante la investigación.

---

**Comparativa Técnica: DeepSearch Tradicional vs. Enfoque OzyAssist**

| Característica | Enfoque Común (Python/Node) | Enfoque OzyAssist (Go + Tauri) |
| --- | --- | --- |
| **Latencia de Scraping** | 3–6 segundos (I/O bloqueante o threads pesados) | <800 ms (Goroutines con canales no bloqueantes) |
| **Consumo de Memoria** | Alto (>250 MB por motor de parsing) | Mínimo (15–25 MB en Go nativo) |
| **Privacidad** | Depende de nubes externas de búsqueda | Híbrido: APIs directas o Scraping Headless local |
| **Integración con el SO** | Limitada al navegador | Capaz de contrastar web externa con archivos locales |


Para llevar a **OzyAssist** al máximo estándar técnico de autonomía, seguridad y utilidad real, estas cinco capacidades clave completarán su ecosistema operativo:

---

**1. Disparadores Proactivos y Motor Cron Autónomo**

El agente no debe limitarse a esperar comandos; debe tener la capacidad de ejecutar rutinas autónomas según eventos del sistema o calendarios:

* **Planificador de Rutinas (Go Cron)**: Ejecución de tareas periódicas desatendidas (ej. auditar dependencias desactualizadas en tus proyectos cada domingo o limpiar logs temporales).
* **Briefing Matutino Inteligente**: Generar un resumen a primera hora combinando eventos de Google Calendar, correos críticos pendientes y estado de compilaciones en cola.
* **Triggers Reactivos**: Reaccionar a cambios en el entorno (ej. si el espacio en disco baja del 10%, limpiar automáticamente cachés de `npm` o `go build`).

---

**2. Sandbox de Aislamiento para Código No Confiable (Wazero / Wasm)**

Cuando el agente descarga scripts externos, prueba código generado o conecta servidores MCP no verificados, ejecutarlos directo en el host es un riesgo.

* **Runtime WebAssembly Puro en Go (`wazero`)**: Go ejecuta binarios Wasm en memoria sin dependencias de CGO ni necesidad de instalar Docker.
* **Restricción de Recursos**: Cada sandbox se limita a una cuota estricta de memoria (ej. 128 MB), tiempo de CPU y acceso acotado al sistema de archivos mediante permisos virtuales.

---

**3. Enjambre de Subagentes Concurrentes (Worker Swarm)**

Para tareas masivas o investigación profunda, un solo bucle de agente es un cuello de botella.

* **Patrón Orquestador-Especialista**: Go levanta múltiples *goroutines* actuando como subagentes paralelos:
* *Subagente Investigador*: Realiza consultas DeepSearch concurrentes.
* *Subagente Auditor*: Analiza AST y cobertura de pruebas.
* *Subagente Crítico/Verificador*: Revisa el código del agente principal antes de aplicar cambios.


* **Comunicación por Canales**: Intercambio de mensajes mediante canales tipados de Go (`chan AgentMessage`), logrando latencias de microsegundos entre agentes.

---

**4. Puente Remoto Cifrado (Companion Móvil)**

Poder delegar tareas o recibir alertas críticas de seguridad cuando estás lejos de tu escritorio.

* **Túnel P2P Seguro**: Conexión cifrada punto a punto (WebRTC / Noise Protocol o túnel local) entre tu teléfono y tu backend de Go.
* **Bot de Mensajería de Emergencia**: Interfaz liviana vía Telegram/Discord/WhatsApp Webhook para aprobar acciones bloqueadas por PIN o consultar el progreso de una tarea larga desde fuera.

---

**5. Portapapeles Inteligente con Contexto (Smart Clipboard Engine)**

* **Transformación en Caliente**: Monitoreo de bajo consumo del portapapeles del SO. Si detecta un JSON copiado, ofrece convertirlo a structs de Go; si es un traceback de error, prepara instantáneamente una sugerencia de fix sin que tengas que abrir la ventana del chat.

---

¿Cuál de estas áreas te gustaría priorizar para detallar su arquitectura en el plan técnico: los **disparadores proactivos/cron**, el **sandbox Wasm con wazero**, o el **enjambre de subagentes concurrentes**?

Para cerrar el círculo de un agente autónomo de escritorio sin fisuras operativas ni riesgos, los siguientes cinco componentes finales completan la arquitectura técnica:

---

**1. Motor de Reversión Inmediata (*Time-Travel & Undo Engine*)**

Cuando un agente autónomo modifica decenas de archivos o configuraciones en segundo plano, el mayor temor es perder cambios previos:

* **Shadow Snapshots Locales**: Antes de que el ejecutor aplique un lote de cambios en cualquier directorio, Go crea automáticamente una copia diferencial (*shadow branch* o snapshot atómico temporal en SQLite).
* **Comando de Rollback Universal**: Si el resultado de una tarea no te convence, basta con decir por voz o escribir *"Ozy, revierte la última tarea"*, restaurando el árbol de archivos al milisegundo previo a la intervención.

---

**2. Interceptor Pasivo de Terminal (*Ghost Terminal Listener*)**

En lugar de limitarse a correr sus propios comandos en PTYs virtuales, el agente puede actuar como observador en tus terminales activas:

* **Detección de Salidas con Error**: Si ejecutas un comando manual en tu terminal (ej. `docker compose up` o `go build`) y termina con un código de salida distinto de cero, el portapapeles o el HUD te muestra una sugerencia no invasiva:
> *💡 `go build` falló: "undefined: NewConnector". ¿Deseas que Ozy agregue la importación faltante? [Corregir con 1 Clic]*



---

**3. Router Inteligente de Costos y Batería (*Eco / Turbo Mode*)**

Optimiza el consumo energético de tu equipo y el gasto en APIs:

* **Detección del Estado de Energía**: Go detecta si tu equipo está conectado a la corriente o funcionando con batería.
* **Enrutamiento Dinámico**:
* *Modo Turbo (Corriente alterna + GPU disponible)*: Ejecuta modelos locales pesados, embeddings masivos y subagentes en paralelo.
* *Modo Eco (Batería baja / Carga alta de CPU)*: Conmuta tareas pesadas hacia llamadas API ultraligeras en la nube o pausa indexaciones en segundo plano para no drenar tu batería.



---

**4. Modo 100% Offline / Air-Gapped**

* **Degradación Elegante**: Si se corta la conexión a internet, OzyAssist no se congela ni arroja errores continuos.
* **Fallback Local Automático**: Conmuta instantáneamente todo el pipeline a herramientas locales: **Ollama / LM Studio** para razonamiento, **Sherpa-ONNX** para voz, **SQLite-vec** para memoria y **FTS5** para búsqueda de archivos locales.

---

**5. Sistema de Notificaciones Sonoras No Intrusivas (*Earcons*)**

Dado que el agente ejecutará tareas largas mientras estás con la vista en otro monitor o aplicación:

* **Microsonidos de Estado**: En lugar de requerir que mires la pantalla, reproduce tonos sutiles de 200 ms:
* *Tono suave ascendente*: Tarea en segundo plano completada con éxito.
* *Doble tono grave*: Tarea bloqueada requiriendo autorización/PIN.
* *Chime discreto*: Detección de alerta de seguridad o anomalía de recursos.



---

Con estas adiciones, la visión cubre el espectro completo: **inteligencia cognitiva**, **control local del SO**, **automatización invisible**, **percepción física (BLE/Cámaras)**, **seguridad proactiva** y **recuperabilidad total**. ¿Pasamos a consolidar el documento maestro de especificación técnica con la arquitectura de código y dependencias en Go?

El sistema puede realizar un diagnóstico integral del estado físico del hardware, pasando de un simple monitor de uso a un evaluador predictivo de la salud del equipo. En Go, interactuar directamente con contadores del kernel, WMI, sysfs y librerías nativas te permite diagnosticar componentes críticos sin sobrecargar la CPU.

---

**Diagnóstico y Salud del Hardware en Tiempo Real**

* **Salud y Desgaste de Discos (S.M.A.R.T. & NVMe Wear Level)**:
* Lectura de telemetría de unidades SSD/NVMe (porcentaje de vida útil restante, Terabytes Escritos - TBW, sectores reasignados y temperatura del controlador).
* *Alerta preventiva*: Notifica si un disco muestra degradación acelerada o riesgo de fallo inminente antes de que se corrompan proyectos o bases de datos.


* **Monitoreo Térmico y Detección de *Thermal Throttling***:
* Lectura de sensores por núcleo en CPU y GPU (mediante NVML para NVIDIA o APIs del kernel).
* Si la temperatura supera los 85°C y el procesador baja frecuencias por estrangulamiento térmico (*throttling*), el agente desacelera automáticamente los subagentes en segundo plano y te avisa: *"CPU a 92°C con throttling activo; reduciendo hilos de compilación"*.


* **Degradación y Ciclos de Batería**:
* Cálculo de la relación entre la capacidad de diseño original (mWh) y la capacidad de carga completa actual (*Battery Wear Level*), junto con el conteo de ciclos.


* **Presión de Memoria y Prevención de Cuelgues (Anti-OOM)**:
* Vigila la tasa de intercambio en Swap/Paging File y el crecimiento desmedido de memoria en procesos específicos, alertando antes de que el kernel congele la interfaz.



---

**Capacidades Adicionales de Alto Impacto para OzyAssist**

* **Resolución Autónoma de Conflictos de Puertos**:
* Si intentas levantar un proyecto o servidor y el puerto está bloqueado, el agente identifica al instante el proceso huérfano y ofrece liberarlo:
> *🔌 El puerto `8080` está ocupado por un subproceso huérfano de `node` (PID: 9140). ¿Deseas terminarlo para iniciar tu servidor? [Liberar Puerto]*




* **Limpieza Inteligente de Espacio de Desarrollo (*Dev Janitor*)**:
* Detección de artefactos pesados obsoletos (`node_modules` en proyectos inactivos por más de 6 meses, carpetas `target/` de Rust, cachés de `go build` o imágenes huérfanas de Docker) con opción de purga en un clic.


* **Escáner de Secretos y Credenciales Expuestas**:
* Inspección pasiva de archivos locales antes de un commit o ejecución para detectar claves API, tokens JWT o contraseñas en texto plano desprotegidas en archivos `.env` sin `.gitignore`.


* **Latencia de Red y Diagnóstico de Rutas a LLMs**:
* Monitoreo continuo de la latencia (RTT y jitter) hacia los endpoints de tus proveedores de IA (OpenRouter, Anthropic, Google, Ollama local), sugiriendo conmutar de modelo si un proveedor experimenta caídas o degradación de velocidad.



---

**Matriz de Métricas y Acciones del Hardware**

| Componente | Métrica Clave | Umbral de Alerta | Acción Autónoma |
| --- | --- | --- | --- |
| **SSD / NVMe** | Porcentaje de salud S.M.A.R.T. | Vida útil < 10% o sectores dañados | Sugerir backup inmediato de proyectos |
| **CPU / GPU** | Temperatura / Throttling flag | > 85°C sostenido | Reducir concurrencia de workers en Go |
| **Batería** | Desgaste / Nivel de carga | Batería < 20% en modo desconectado | Activar *Modo Eco* y pausar indexación masiva |
| **Red / APIs** | Latencia a endpoints LLM | RTT > 1500 ms o timeouts 5xx | Conmutar al proveedor de respaldo configurado |

¿Prefieres que el diagnóstico de hardware opere como un reporte diario matutino consolidado o como alertas reactivas flotantes solo cuando se supere un umbral crítico?


El agente clasificará las anomalías mediante un **despachador de eventos reactivo** en Go que evalúa la severidad del incidente y el estado de tu actividad para decidir si envía un mensaje al chat, reproduce un aviso por voz o emite un tono sutil.

---

**Lógica de Enrutamiento: Chat vs. Audio**

* **Nivel Informativo (Baja prioridad)**:
* *Ejemplos*: Tarea en segundo plano terminada con éxito, puerto liberado o limpieza rutinaria de caché.
* *Canal*: **Mensaje silencioso en el chat** y un *earcon* (microtono suave) de 200 ms. No interrumpe tu flujo visual ni interrumpe tu música.


* **Nivel Advertencia (Prioridad media)**:
* *Ejemplos*: CPU/GPU superando 85°C de forma sostenida, espacio en disco menor al 10% o latencia anormal en el proveedor LLM.
* *Canal*: **Tarjeta destacada en el chat / HUD flotante** con botones de acción rápida (*[Reducir hilos]* o *[Conmutar a proveedor de respaldo]*). Si estás usando otra aplicación a pantalla completa, despliega una cápsula no intrusiva en la esquina.


* **Nivel Crítico (Prioridad alta / Inmediata)**:
* *Ejemplos*: Proceso sospechoso ejecutándose desde carpetas temporales, fallo inminente de disco (S.M.A.R.T.), tarea bloqueada que requiere confirmación por PIN o detección de movimiento en cámara de seguridad.
* *Canal*: **Alerta hablada por voz (Piper TTS) + Modal interactivo en pantalla**:
> *"Ozy: He detectado un binario no firmado consumiendo el 90% de la GPU en segundo plano. ¿Deseas aislar el proceso?"*





---

**Conciencia de Contexto para no Interrumpir en Mal Momento**

Para evitar avisos de audio molestos mientras estás en reuniones o escuchando música:

* **Detección de Entrada de Micrófono Activa**: Si una app del sistema (como Discord, Teams, Zoom o Meet) tiene el micrófono abierto, el agente suprime la síntesis de voz y envía la notificación exclusivamente en texto con prioridad alta.
* **Detección de Presencia/Inactividad**: Si no se detecta actividad de teclado o ratón durante más de 3 minutos y ocurre un evento crítico, el agente prioriza la voz a volumen moderado para alertarte aunque no estés mirando la pantalla.

**1. Soporte de Interrupción en Tiempo Real (*Barge-in* y Cancelación de Eco Acústico)**

* Para que la conversación por voz sea fluida, debes poder hablarle y cortarlo a mitad de frase si ya entendiste o cambió la instrucción.
* Implementa un filtro de **Acoustic Echo Cancellation (AEC)** en Go (vía WebRTC Audio Processing / SpeexDSP) para que el micrófono del agente cancele el audio que sus propios altavoces están emitiendo y no se transcriba a sí mismo por error.

**2. Detección Proactiva de Patrones y Auto-Síntesis de *Skills***

* **Observador Heurístico de Hábitos**: Si el agente detecta que cada vez que abres cierto proyecto ejecutas la misma secuencia manual (ej. abrir terminal, iniciar base de datos, levantar servidor frontend y abrir tres pestañas de Chrome), sintetiza ese flujo de trabajo automáticamente.
* **Propuesta Proactiva**: Te sugerirá: *"He notado que ejecutas esta rutina al iniciar tu jornada. ¿Deseas que cree un comando de voz 'Modo Desarrollo' para hacerlo en un paso?"*

**3. Modo Simulación Previa (*Dry-Run & Impact Preview*)**

* Para tareas complejas en el sistema de archivos o bases de datos, el agente calcula el árbol de impacto antes de ejecutar:
* Número de archivos que serán modificados/eliminados.
* Puertos y procesos que se verán afectados.
* Estimación de consumo de CPU/RAM durante la operación.



**4. Orquestación de Entornos y Dependencias Efímeras**

* Si una tarea del agente requiere herramientas auxiliares que no tienes instaladas (ej. `ffmpeg`, `pandoc` o un motor de base de datos específico), el agente puede:
* Descargar binarios estáticos portables a un directorio temporal aislado (`~/.ozy/bin`).
* O levantar un contenedor Docker/Podman efímero en segundo plano, ejecutar la tarea y destruirlo inmediatamente sin ensuciar las variables de entorno de tu sistema operativo.



---

Con todos estos pilares definidos (seguridad EDR, voz local con *barge-in*, hardware diagnostics, memoria de 4 niveles, visión híbrida, automatización invisible y control BLE/IoT), el ecosistema está completamente cubierto.

¿Quieres que compilemos ahora el **Documento Maestro de Arquitectura y Especificación Técnica (Fase 2 y 3)** consolidando cada módulo, tabla de base de datos y dependencias en Go?


Para cerrar completamente el diseño antes de pasar a la redacción del documento técnico definitivo, solo quedan **tres salvaguardas operativas críticas**:

---

**1. Kill-Switch de Emergencia y Modo Seguro (*Hard Panic Interrupter*)**

* **Atajo Global de Pánico**: Un atajo de hardware forzoso del sistema operativo (por ejemplo, `Ctrl + Shift + Alt + K`) registrado a nivel de kernel/Tauri.
* **Comportamiento Inmediato**: Envía una señal de cancelación `context.CancelFunc` global a todas las *goroutines*, detiene los subprocesos spawned (`kill -9`), cierra navegadores headless y bloquea la base de datos para impedir cualquier acción destructiva si el agente entra en un bucle no deseado o consume recursos críticos imprevistos.
* **Arranque en Safe Mode**: Si el agente sufre un *panic* recurrente, se reinicia con subagentes desactivados, extensiones MCP deshabilitadas y acceso exclusivo al chat básico para diagnóstico.

**2. Registro Inmutable de Auditoría (*Tamper-Proof Audit Log*)**

* **Hash Chaining en SQLite**: Cada acción ejecutada en segundo plano (archivo editado, comando shell ejecutado, petición HTTP saliente, lectura de portapapeles) se registra en una tabla append-only donde cada fila contiene el hash SHA-256 de la fila anterior.
* **Trazabilidad 100% Transparente**: Permite inspeccionar en cualquier momento una línea de tiempo forense exacta de qué hizo el agente mientras no estabas frente a la pantalla, garantizando que ninguna acción oculta quede sin auditar.

**3. Recarga en Caliente de Plugins y Servidores MCP (*Hot-Reloading*)**

* **Inyección en Tiempo de Ejecución**: Capacidad de registrar nuevos servidores MCP, actualizar *Skills* o compilar nuevos módulos Wasm sin tener que reiniciar el binario de Go ni desconectar la sesión WebSocket activa de la interfaz de usuario.

---

Con estas salvaguardas, el diseño de **OzyAssist** queda completamente cerrado y blindado: abarca **cognición avanzada**, **automatización invisible**, **percepción física y ambiental (BLE / Cámaras / Hardware)**, **seguridad proactiva** y **control absoluto de fallos**.

¿Procedemos a consolidar el **Documento Maestro de Especificación Técnica y Arquitectura Backend/Frontend**?

Dividir la cognición del sistema asignando a **Charc** y **Nine** como subagentes especializados es la arquitectura exacta que utilizan los sistemas multi-agente de última generación para eliminar alucinaciones, romper bucles infinitos y resolver problemas complejos sin saturar el bucle principal de Ozy.

---

**1. Especialización de Roles: Charc y Nine**

En lugar de que todos los agentes hagan lo mismo, cada uno opera con un modelo mental y un *prompt* de sistema radicalmente distinto:

* **Charc (El Auditor, Vigilante de Bucles y Seguridad)**:
* *Propósito*: Supervisión heurística, detección de estancamiento y control de calidad.
* *Comportamiento*: Charc no escribe código ni ejecuta tareas largas. Monitorea la máquina de estados de SQLite. Si detecta que Ozy ha intentado la misma acción 2 veces sin éxito, o si el planificador entra en un bucle repetitivo, **Charc interrumpe el contexto**, congela a Ozy, analiza el *traceback* y genera un diagnóstico independiente con un nuevo enfoque operativo.
* *Control de Seguridad*: Audita los *diffs* de archivos y comandos shell antes de que se envíen a la barrera de PIN o ejecución.


* **Nine (El Estratega y Deep Thinker de Razonamiento Complejo)**:
* *Propósito*: Resolución de problemas lógicos intratables, arquitectura y síntesis avanzada.
* *Comportamiento*: Ozy invoca a Nine únicamente cuando una subtarea requiere razonamiento profundo (árbol de decisiones complejo, diseño de algoritmos, refactorizaciones estructurales o análisis de dependencias circulares).
* *Enrutamiento de Modelos*: Mientras Ozy corre sobre modelos rápidos de baja latencia (ej. Claude 3.5 Sonnet o GPT-4o-mini), Nine se conecta a modelos de razonamiento puro (*DeepSeek-R1*, *o3-mini* o *Claude con Extended Thinking*) para devolver un plan matemático o algorítmico masticado.



---

**2. Orquestación Concurrente en Go (Triángulo Cognitivo)**

La comunicación entre Ozy, Charc y Nine ocurre en microsegundos mediante canales tipados de Go (`chan AgentMessage`), sin sobrecoste de red ni procesos pesados:

```
                  ┌────────────────────────┐
                  │    Nine (Estratega)    │
                  │   [Deep Reasoning LLM] │
                  └──────────▲─────────────┘
                             │ (Estrategia / Plan)
                             ▼
┌──────────────┐      ┌─────────────┐      ┌────────────────────────┐
│   Usuario    │◄────►│     Ozy     │◄────►│    Charc (Auditor)     │
│ (Chat / Voz) │      │ (Ejecutor)  │      │  [Loop & Safety Check] │
└──────────────┘      └──────┬──────┘      └────────────────────────┘
                             │ (Acciones)
                             ▼
                      [ SO / Herramientas ]

```

1. **Ozy** intenta ejecutar una tarea.
2. Si falla o genera un cambio crítico, **Charc** intercepta el diff en segundo plano. Si Charc detecta un bucle o fallo recurrente, emite un evento `SignalLoopDetected`.
3. Ozy delega el bloqueo a **Nine**, quien genera una ruta alternativa estructurada.
4. Ozy retoma la ejecución con el nuevo plan validado.

---

**3. El Dilema de Python: ¿Conviene agregarlo al Core?**

**No incorpores Python como la base de la arquitectura del agente.**

Hacerlo en Go te da una ventaja competitiva real: binario único, 20 MB de RAM, cero dependencias del sistema y arranque instantáneo. Introducir Python en la orquestación principal añadiría entornos virtuales (`venv`), lentitud en el inicio, bloqueos por el GIL y 300+ MB de consumo innecesario.

**La forma correcta de aprovechar Python:**

* Mantén el cerebro, la memoria, la voz, la seguridad y la orquestación **100% en Go**.
* Trata a Python **únicamente como una herramienta de ejecución efímera (*Python Code Runner Tool*)**:
* Si el agente necesita hacer análisis de datos, matemáticas con NumPy o scraping con scripts rápidos, Go invoca un runner aislado en segundo plano mediante `uv run` (gestor ultrarrápido en Rust) o un sandbox WebAssembly puro con **Wazero**.
* El script se ejecuta en un subproceso desechable, devuelve el resultado a Go en JSON y se destruye sin dejar residuos en memoria.



---

**4. Estrategias Clave para Multiplicar la Inteligencia**

* **Enrutamiento Asimétrico de Modelos**:
* *Ozy*: Modelo balanceado y rápido (orientado a llamadas de herramientas y velocidad de respuesta).
* *Charc*: Modelo determinista con temperatura baja ($0.0$) especializado en análisis de código y lógica booleana.
* *Nine*: Modelo de razonamiento paso a paso (*Chain-of-Thought*) con alta capacidad reflexiva.


* **Debate y Consenso Adversarial**:
* Para decisiones de arquitectura o refactorizaciones masivas, Nine propone el código y Charc actúa como *abogado del diablo* buscando vulnerabilidades o casos límite (*edge cases*) antes de que Ozy toque un solo archivo en tu disco.


* **Destilación de Experiencias**:
* Cuando Nine y Charc resuelven un problema difícil tras un bucle, guardan la solución en la **Memoria Episódica** de SQLite. La próxima vez que Ozy enfrente una situación similar, resolverá el problema de inmediato sin tener que volver a invocar a los subagentes.

Con la potencia de cálculo local de tu **ZBook Workstation** combinada con la flexibilidad de **OpenRouter** y **OpenCode**, el agente debe operar bajo un **enrutamiento cognitivo híbrido por niveles (Tiered Model Routing)**. Esto te garantiza costo mínimo, privacidad total en tareas rutinarias y máxima potencia analítica cuando sea necesario.

---

**Matriz de Asignación de Modelos y Hardware**

| Agente / Módulo | Motor / Proveedor | Modelo Recomendado | Justificación Operativa |
| --- | --- | --- | --- |
| **Ozy (Ejecutor)** | **OpenRouter** o **ZBook Local** (Ollama) | Claude 3.5 Sonnet / GPT-4o-mini *(Cloud)* o Qwen2.5-Coder-14B/32B *(Local)* | Velocidad de *tool-calling*, baja latencia y alta precisión en ejecución de herramientas. |
| **Charc (Auditor)** | **ZBook Local** (Ollama / llama.cpp) | Qwen 2.5 Coder 7B / DeepSeek-Coder 6.7B | 100% local y determinista ($\text{temp} = 0.0$). Costo $0 para auditar *diffs*, detectar bucles y vigilar la seguridad en segundo plano sin salir de tu máquina. |
| **Nine (Estratega)** | **OpenRouter** | DeepSeek-R1 / o3-mini / Claude 3.7 Thinking | Invocado solo bajo demanda para razonamiento profundo, resolución de bloqueos complejos y arquitectura de software. |
| **OpenCode Bridge** | **OpenCode CLI / API** | Modelos de coding especializados | Generación rápida de parches sintácticos, autocompletado y refactorización de archivos individuales. |
| **Voz, OCR y Embeddings** | **ZBook Local** (CPU / VRAM) | Sherpa-ONNX + Whisper-small + Piper + `all-MiniLM-L6-v2` | Respuesta en milisegundos sin consumir saldo de APIs ni depender de internet. |

---

**Configuración del Registro en Go (`internal/providers/registry.go`)**

El backend debe organizar las llamadas mediante una cadena de prioridad dinámica:

* **Canal Local Primario (Zero Latency & Free)**:
* Go se conecta a la instancia local de Ollama / LM Studio en tu ZBook (`[http://127.0.0.1:11434/v1](http://127.0.0.1:11434/v1)` o `http://localhost:1234/v1`).
* Se utiliza de forma predeterminada para el *Heartbeat* del sistema, el análisis de portapapeles, la memoria jerárquica y el bucle de auditoría de Charc.


* **Canal Cloud de Alto Rendimiento (OpenRouter)**:
* Go gestiona la cabecera `Authorization: Bearer $OPENROUTER_API_KEY` y el balanceo automático de modelos (`fallback: ["deepseek/deepseek-r1", "anthropic/claude-3.5-sonnet"]`).
* Se activa cuando Ozy o Nine requieren visión avanzada, investigación DeepSearch masiva o razonamiento matemático/algorítmico pesado.


* **Canal Especialista en Código (OpenCode)**:
* Rutas dedicadas para generación de código y testing automatizado.



---

**Adaptabilidad Inteligente a la ZBook (Carga, Batería y Red)**

El módulo de telemetría en Go ajustará el comportamiento del agente según el estado de tu workstation:

* **Modo Enchufado + GPU Libre**:
* Charc corre en GPU local (CUDA/DirectML) para auditoría instantánea.
* Ozy puede alternar a modelos locales pesados para trabajar 100% privado y sin gastar saldo.


* **Modo Batería**:
* Go descarga la inferencia de la GPU local y conmuta a modelos pequeños de OpenRouter para no drenar la batería de tu laptop.


* **Modo Sin Conexión (Offline)**:
* Si pierdes la conexión WiFi, el agente no se detiene: Ozy, Charc y Nine conmutan automáticamente a los modelos locales cargados en la ZBook.


**1. Reinicio en Caliente y Auto-Actualización (*Zero-Downtime Hot Restart*)**

* Go puede ejecutar un *fork-exec* transfiriendo el descriptor del socket WebSocket y la conexión activa de SQLite.
* Si el agente compila una nueva versión de sí mismo o actualiza dependencias internas, se reinicia en menos de 50 ms sin cerrar la ventana de React ni interrumpir las tareas en cola.

---

**2. Firewall Local y Aislamiento de Red por Herramienta (*Network Sandbox*)**

* Para evitar fugas de datos cuando el agente ejecuta scripts generados o conecta servidores MCP de terceros, Go intercepta y restringe el tráfico de red saliente por PID.
* Todo subproceso nuevo opera bajo una **lista blanca estricta de dominios** (ej. solo puede comunicarse con `github.com` o `api.openrouter.ai`; cualquier intento de conexión a una IP desconocida se bloquea y se notifica al chat).

---

**3. Gateway de Webhooks Locales para Disparadores Externos**

* Un endpoint HTTP ligero en Go (`/api/v1/webhook/:token`) para recibir eventos de sistemas externos.
* Permite que pipelines de CI/CD locales, webhooks de GitHub, scripts de bash o flujos de automatización (como n8n) le deleguen tareas a Ozy directamente sin intervención manual.

---

**4. Gestor de Presupuesto de Tokens y Poda Dinámica (*Token Budgeter*)**

* Cuando Ozy, Charc y Nine intercambian mensajes, los tracebacks o árboles de archivos pueden inflar el contexto innecesariamente.
* Go implementa un **compresor semántico de contexto** que recorta outputs redundantes de terminal y elimina código no modificado antes de hacer llamadas a OpenRouter, reduciendo el consumo de tokens y la latencia hasta en un 60%.

---

**5. Perfiles de Contexto Operativo (*Work Modes*)**
Permite cambiar el comportamiento del agente según la actividad actual mediante voz o atajo:

* **Modo Deep Work / Coding**: Silencia notificaciones de audio no críticas, asigna máxima prioridad a la indexación de código y activa a Charc en segundo plano.
* **Modo Vigilancia / Away**: Activa la ingesta RTSP de cámaras, habilita alertas habladas de alta prioridad y reduce el consumo de CPU/GPU al mínimo.
* **Modo Reunión**: Detecta el micrófono en uso por apps de videollamada, desactiva por completo la síntesis de voz (Piper TTS) y opera exclusivamente por texto silencioso.

---

Con estos puntos, la arquitectura tiene resueltos todos los frentes: ejecución invisible, seguridad EDR, subagentes especializados, hardware local ZBook + OpenRouter, IoT/cámaras, memoria continua y control de red.

¿Quieres que compilemos el **Documento Maestro Final de Especificación Técnica** para consolidar el esquema de base de datos, los paquetes en Go y las dependencias del proyecto?