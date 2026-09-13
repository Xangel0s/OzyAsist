# Especificación Técnica Maestra y Roadmap de Implementación — OzyAssist Core

Este documento consolida la arquitectura definitiva, el esquema de base de datos, el árbol de paquetes en Go y las fases de desarrollo para **OzyAssist**, estructurado para operar de forma autónoma, segura y multimodal en tu ZBook Workstation con soporte híbrido en la nube.

---

## 1. Arquitectura Cognitiva Triádica y Enrutamiento de Modelos

El cerebro del sistema se divide en tres roles asimétricos comunicados mediante canales Go (`chan AgentMessage`), evitando bucles infinitos y alucinaciones:

```
                  ┌────────────────────────┐
                  │    Nine (Estratega)    │
                  │   [Deep Reasoning LLM] │
                  └──────────▲─────────────┘
                             │ (Rutas alternativas / Arquitectura)
                             ▼
┌──────────────┐      ┌─────────────┐      ┌────────────────────────┐
│   Usuario    │◄────►│     Ozy     │◄────►│    Charc (Auditor)     │
│ (Chat / Voz) │      │ (Ejecutor)  │      │  [Loop & Safety Check] │
└──────────────┘      └──────┬──────┘      └────────────────────────┘
                             │ (Acciones en background)
                             ▼
                    [ SO / PTY / CDP / MCP ]

```

### Matriz de Asignación por Hardware y Nivel de Carga

| Rol / Módulo            | Motor / Proveedor              | Modelo Recomendado                      | Temperatura | Modo Batería / Eco             |
| ----------------------- | ------------------------------ | --------------------------------------- | ----------- | ------------------------------ |
| **Ozy (Ejecutor)**      | OpenRouter / Ollama Local      | Claude 3.5 Sonnet / Qwen2.5-Coder-14B   | $0.2$       | Qwen2.5-Coder-7B (Local)       |
| **Charc (Auditor/EDR)** | ZBook Local (Ollama/llama.cpp) | Qwen 2.5 Coder 7B / DeepSeek-Coder      | $0.0$       | Qwen 2.5 Coder 3B (Local)      |
| **Nine (Estratega)**    | OpenRouter                     | DeepSeek-R1 / o3-mini / Claude Thinking | $0.6$       | Invocado solo bajo bloqueo     |
| **OpenCode Bridge**     | OpenCode API / CLI             | Modelos de síntesis de parches          | $0.1$       | Fallback a script local        |
| **Audio (STT/TTS)**     | ZBook Local (CGO / CPU)        | Sherpa-ONNX + Whisper + Piper           | —           | Inferencia ultra-ligera en CPU |

---

## 2. Estructura de Paquetes en Go (`backend/internal/`)

El backend desacopla el ciclo de vida del agente, la telemetría, los controladores del SO y la percepción física:

```text
backend/
├── cmd/
│   ├── server/
│   │   └── main.go                 # Arranque, DI, WAL SQLite y WebSocket Hub[cite: 1]
│   └── ozyctl/
│       └── main.go                 # CLI rápida comunicada por Named Pipe / Unix Socket
├── internal/
│   ├── agent/                      # Orquestación triádica y control de bucles[cite: 1]
│   │   ├── ozy.go                  # Bucle ejecutor principal y llamadas a herramientas
│   │   ├── charc.go                # Auditor local: detección de bucles y filtrado de diffs
│   │   ├── nine.go                 # Estratega de deep thinking para desbloqueo de tareas
│   │   ├── self_heal.go            # Bucle de auto-recuperación ante tracebacks
│   │   └── verifier.go             # Verificación heurística de resultados (exit codes, DOM)
│   ├── taskengine/                 # Máquina de estados desacoplada en segundo plano[cite: 3]
│   │   ├── state_machine.go        # Transiciones de estado persistidas en SQLite[cite: 3]
│   │   ├── worker_pool.go          # Pool concurrente de goroutines no bloqueantes
│   │   └── scheduler.go            # Priorización de tareas y planificador cron proactivo[cite: 3]
│   ├── system/                     # Control nativo del Sistema Operativo[cite: 3]
│   │   ├── telemetry.go            # gopsutil + S.M.A.R.T. + Térmico + Batería[cite: 3]
│   │   ├── window_manager.go       # UI Automation / Win32 IPC invisible (sin mover cursor)[cite: 3]
│   │   ├── pty_pool.go             # Terminales virtuales en background con replay asciinema
│   │   └── undo_engine.go          # Shadow snapshots y rollback universal
│   ├── security/                   # Watchdog EDR y aislamiento de red
│   │   ├── watchdog.go             # Detección de binarios anómalos, rutas temp y minería
│   │   ├── firewall.go             # Restricción de tráfico saliente por PID (lista blanca)
│   │   └── audit_trail.go          # Registro inmutable con hash chaining SHA-256
│   ├── browser/                    # Automatización web headless y relay[cite: 3]
│   │   ├── cdp_client.go           # Chromedp en modo headless invisible[cite: 3]
│   │   ├── extension_bridge.go     # Relay WebSocket con extensión de Chrome activa
│   │   └── session_clone.go        # Clonación segura de cookies/perfil en caso de bloqueo
│   ├── memory/                     # Memoria jerárquica de 4 niveles[cite: 1]
│   │   ├── working.go              # Scratchpad volátil de subtarea activa
│   │   ├── semantic.go             # Grafo de conocimiento + SQLite-vec + FTS5 híbrido[cite: 1]
│   │   ├── episodic.go             # Registro de tracebacks y soluciones previas[cite: 1]
│   │   ├── procedural.go           # Biblioteca de Skills y playbooks autogenerados[cite: 1]
│   │   └── compactor.go            # Consolidación nocturna/idle en background
│   ├── search/                     # Motor de investigación[cite: 1]
│   │   ├── deepsearch.go           # Sub-queries paralelas, crawling y reranker
│   │   └── citations.go            # Anclaje estricto (grounding) y contrato de fuentes
│   ├── perception/                 # Visión y Hardware ambiental
│   │   ├── screen_capture.go       # Captura de ventanas tapadas vía framebuffer GPU
│   │   ├── camera_rtsp.go          # Ingesta RTSP (gortsplib) + YOLOv8 local + VLM
│   │   └── bluetooth.go            # tinygo.org/x/bluetooth para dispositivos BLE / GATT
│   ├── voice/                      # Audio local con baja latencia[cite: 1]
│   │   ├── wakeword.go             # Sherpa-ONNX "Hey Ozy" en CGO (<1% CPU)[cite: 3]
│   │   ├── stt.go                  # Whisper local con VAD integrado[cite: 3]
│   │   ├── tts.go                  # Piper TTS neuronal (<200 ms)[cite: 3]
│   │   └── aec_filter.go           # Cancelación de eco acústico para soporte Barge-in
│   └── connectors/                 # Integraciones externas[cite: 1]
│       ├── google/                 # OAuth2 PKCE + Gmail (Drafts/Send) + Calendar + Drive[cite: 3]
│       ├── mcp/                    # Orquestador dinámico stdio/SSE + Hot-Reloading[cite: 1]
│       └── sandbox_wasm.go         # Wazero runtime para scripts de Python/código efímero

```

---

## 3. Esquema de Base de Datos y Persistencia (SQLite WAL Mode)

Inicialización del motor para alta concurrencia:

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;

```

### Migración `009_background_engine.up.sql`

```sql
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

CREATE TABLE IF NOT EXISTS task_steps (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    agent_assigned TEXT NOT NULL DEFAULT 'ozy', -- 'ozy', 'charc', 'nine'
    action_type TEXT NOT NULL,                  -- 'shell_exec', 'mcp_call', 'cdp_action', 'file_patch', 'ble_write', 'google_draft'
    payload TEXT NOT NULL,
    requires_pin BOOLEAN DEFAULT 0,
    pin_authorized BOOLEAN DEFAULT 0,
    verification_rule TEXT,                     -- JSON: {"type": "exit_code", "expected": 0}
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

### Migración `010_hierarchical_memory.up.sql`

```sql
-- Memoria Semántica (Hechos atómicos persistentes extraídos de audio/texto)
CREATE TABLE IF NOT EXISTS user_memories (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    category TEXT NOT NULL,                     -- 'preferencia', 'stack', 'hardware', 'regla'
    content TEXT NOT NULL,
    embedding BLOB,                             -- Vector 384-dim (all-MiniLM-L6-v2)
    access_count INTEGER DEFAULT 1,
    last_recalled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Memoria Episódica (Historial de tracebacks y soluciones de Charc/Nine)
CREATE TABLE IF NOT EXISTS episodic_traces (
    id TEXT PRIMARY KEY,
    error_signature TEXT NOT NULL,              -- Hash del tipo de error / traceback
    context_snippet TEXT NOT NULL,
    solution_patch TEXT NOT NULL,
    resolved_by TEXT NOT NULL,                  -- 'nine', 'charc_patch', 'self_heal'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS user_memories_fts USING fts5(content, category);

```

### Migración `011_security_and_audit.up.sql`

```sql
-- Registro inmutable con encadenamiento criptográfico (Hash Chaining)
CREATE TABLE IF NOT EXISTS audit_trail_chained (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    agent TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    record_hash TEXT NOT NULL
);

-- Snapshots para Time-Travel y Rollback
CREATE TABLE IF NOT EXISTS shadow_snapshots (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content_before BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

```

---

## 4. Protocolos de Comunicación y Contratos de Eventos

### Evento WebSocket: Alerta EDR / Diagnóstico Hardware

```json
{
  "type": "system:alert",
  "data": {
    "level": "critical",
    "channel": "voice_and_modal",
    "category": "security_watchdog",
    "title": "Proceso Sospechoso Detectado",
    "message": "Se detectó el ejecutable xmr_worker.exe corriendo desde AppData\\Temp consumiendo 92% de GPU.",
    "pid": 14208,
    "actions": [
      { "id": "kill_process", "label": "Terminar Proceso", "style": "danger" },
      { "id": "quarantine", "label": "Aislar Archivo", "style": "secondary" }
    ]
  }
}
```

### Evento WebSocket: DeepSearch con Citaciones

```json
{
  "type": "deepsearch:result",
  "data": {
    "query": "Mejores prácticas para WAL mode en SQLite con Go",
    "content": "Para maximizar la concurrencia, SQLite debe configurarse en modo WAL junto con PRAGMA busy_timeout [1]. Esto previene errores de bloqueo en múltiples goroutines [2].",
    "sources": [
      {
        "id": 1,
        "domain": "sqlite.org",
        "title": "Write-Ahead Logging",
        "url": "https://sqlite.org/wal.html"
      },
      {
        "id": 2,
        "domain": "github.com/mattn/go-sqlite3",
        "title": "Concurrency Guide",
        "url": "https://github.com/mattn/go-sqlite3"
      }
    ]
  }
}
```

---

## 5. Roadmap de Desarrollo y Sprints de Ejecución

```
  [Sprint 1: Motor & Triada] ──► [Sprint 2: Seguridad & Hardware] ──► [Sprint 3: Voz, Visión & BLE]
               │                                │                                │
               ▼                                ▼                                ▼
   Máquina estados SQLite             Watchdog EDR + S.M.A.R.T.         Sherpa-ONNX + Barge-in
   Ozy + Charc + Nine                 Firewall PID + Sandbox Wasm       YOLOv8 RTSP + Tinygo BLE
   Browser CDP + Extension            DeepSearch Grounding              Tauri HUD + Smart Clipboard

```

| Sprint       | Enfoque Principal                     | Entregables Clave                                         |
| ------------ | ------------------------------------- | --------------------------------------------------------- |
| **Sprint 1** | **Core Asíncrono y Tríada Cognitiva** | • Tablas SQLite `agent_tasks` y `task_steps` con WAL.<br> |

<br>• Implementación de canales Go para Ozy, Charc y Nine.<br>

<br>• Chromedp Headless y puente con extensión de Chrome.<br>

<br>• Bucle de auto-reparación y máquina de estados. |
| **Sprint 2** | **Seguridad EDR, Memoria y DeepSearch** | • Demonio EDR (`watchdog.go`) y telemetría S.M.A.R.T./térmica.<br>

<br>• Memoria jerárquica en 4 niveles con SQLite-vec + FTS5.<br>

<br>• Motor de DeepSearch concurrente con citas estructuradas.<br>

<br>• Sandbox Wasm (`wazero`) y Time-Travel Undo Engine. |
| **Sprint 3** | **Multimodalidad, IoT y Despliegue** | • Wake Word ("Hey Ozy") con Sherpa-ONNX y AEC (Barge-in).<br>

<br>• Ingesta RTSP para cámaras de seguridad y módulo BLE GATT.<br>

<br>• HUD flotante en Tauri con smart clipboard y atajo de pánico.<br>

<br>• Empaquetado en binario único con `//go:embed`. |

El diseño reúne todas las especificaciones necesarias para ejecutar la construcción del sistema en tu entorno de desarrollo.
