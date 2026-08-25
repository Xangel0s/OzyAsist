# Arquitectura — OzyAssist (Backend Go + Gin)
### v2 — Actualizado con SQLite, Agent System, y Memory en 3 capas

---

## 1. Decisiones cerradas en esta iteración

| Decisión | Resolución |
|---|---|
| Base de datos | **SQLite** (no MongoDB) — binario único, cero fricción de instalación |
| Auth | **Sin auth por defecto** — modo single-user local. JWT se activa solo si se habilita modo servidor/multiusuario a futuro |
| Agent System | **Sí, incluido** — con modelo de permisos por niveles (no todo o nada) |
| Orden de fases | **Memory antes que Agent** — el agente necesita contexto semántico para ser útil |

---

## 2. Estructura de carpetas (actualizada)

```
backend/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   │   ├── router.go
│   │   ├── middleware/
│   │   │   ├── auth.go             # no-op en modo local; activo en modo servidor
│   │   │   └── ratelimit.go
│   │   └── handlers/
│   │       ├── chat.go
│   │       ├── projects.go
│   │       ├── skills.go
│   │       ├── connectors.go
│   │       ├── memory.go
│   │       ├── search.go
│   │       ├── models.go
│   │       ├── files.go
│   │       ├── agent.go
│   │       └── sidebar.go
│   ├── providers/
│   │   ├── interface.go
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   ├── openrouter.go
│   │   └── lmstudio.go
│   ├── memory/
│   │   ├── embeddings.go
│   │   ├── qdrant.go
│   │   ├── chunker.go
│   │   ├── memory_import.go
│   │   ├── episodic.go             # NUEVO — memoria episódica de tareas del agente
│   │   ├── project_memory.go       # NUEVO — lectura/escritura del .md vivo por proyecto
│   │   └── store.go
│   ├── agent/
│   │   ├── planner.go
│   │   ├── executor.go
│   │   ├── permissions.go          # NUEVO — niveles de permiso, validación de paths
│   │   ├── audit.go                # NUEVO — log de acciones del agente
│   │   ├── browser.go
│   │   ├── filesystem.go
│   │   ├── terminal.go
│   │   ├── code_index.go           # NUEVO — indexado de símbolos/funciones (no código crudo)
│   │   └── screenshot.go
│   ├── db/
│   │   ├── sqlite.go                # NUEVO — reemplaza mongo.go
│   │   ├── migrations/              # NUEVO — schema versionado (golang-migrate)
│   │   └── models/
│   │       ├── user.go
│   │       ├── chat.go
│   │       ├── message.go
│   │       ├── project.go
│   │       ├── skill.go
│   │       ├── connector.go
│   │       ├── memory.go
│   │       ├── agent_task.go
│   │       └── agent_action.go      # NUEVO — auditoría de acciones
│   ├── websocket/
│   │   ├── hub.go
│   │   └── client.go
│   ├── skills/
│   │   └── executor.go
│   ├── connectors/
│   │   └── mcp_client.go
│   └── search/
│       └── engine.go                # FTS5 (SQLite) + vector (Qdrant) híbrido
├── go.mod
├── Dockerfile
└── docker-compose.yml                # solo Qdrant queda en Docker; SQLite es un archivo local
```

---

## 3. Persistencia: SQLite

- Driver: `modernc.org/sqlite` (pure Go, sin cgo — mantiene el binario portable).
- Búsqueda full-text nativa vía extensión **FTS5** (ya viene en SQLite moderno), usada en `search/engine.go` combinada con la búsqueda vectorial de Qdrant.
- Migraciones versionadas con `golang-migrate` desde el día uno, para no arrastrar deuda de schema.
- Un solo archivo `.db` en el directorio de datos del usuario (`~/.ozyassist/ozy.db` en Linux/Mac, `%APPDATA%\OzyAssist\ozy.db` en Windows) — respaldo trivial: copiar un archivo.
- Qdrant sigue siendo el único servicio que requiere Docker (o binario standalone de Qdrant, que también existe sin Docker si se quiere simplificar aún más a futuro).

### Esquema base (tablas principales)

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    profile_md TEXT,
    default_provider TEXT,
    default_model TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT NOT NULL,
    root_path TEXT,                    -- workdir para modo Sandboxed del agente
    instructions_md TEXT,               -- memoria de proyecto (capa 1)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chats (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    project_id TEXT REFERENCES projects(id),
    mode TEXT CHECK(mode IN ('chat','code')),
    provider TEXT,
    model TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT REFERENCES chats(id),
    role TEXT CHECK(role IN ('user','assistant','tool')),
    content TEXT,
    attachments_json TEXT,              -- array serializado
    tool_calls_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE VIRTUAL TABLE messages_fts USING fts5(content, content=messages, content_rowid=id);

CREATE TABLE skills (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT,
    description TEXT,
    trigger_pattern TEXT,
    execution_type TEXT CHECK(execution_type IN ('script','prompt_template','api_call')),
    config_json TEXT
);

CREATE TABLE connectors (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT,
    type TEXT,
    endpoint TEXT,
    auth_config_json TEXT
);

CREATE TABLE agent_tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    chat_id TEXT REFERENCES chats(id),
    goal TEXT,
    plan_json TEXT,                     -- pasos planificados
    status TEXT CHECK(status IN ('planning','running','completed','failed','cancelled')),
    permission_level TEXT CHECK(permission_level IN ('read_only','sandboxed','trusted')),
    summary TEXT,                        -- resumen final, se embebe para memoria episódica
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE TABLE agent_actions (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES agent_tasks(id),
    action_type TEXT CHECK(action_type IN ('file_read','file_write','command_exec','browser_action')),
    target TEXT,                         -- path o comando
    details_json TEXT,
    requires_confirmation BOOLEAN,
    confirmed_by_user BOOLEAN,
    result TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Agent System — modelo de permisos

### Niveles

```go
package agent

type PermissionLevel string

const (
    LevelReadOnly   PermissionLevel = "read_only"
    LevelSandboxed  PermissionLevel = "sandboxed"
    LevelTrusted    PermissionLevel = "trusted"
)

type PermissionConfig struct {
    Level             PermissionLevel
    ProjectRoot       string   // requerido en modo Sandboxed
    CommandTimeout    time.Duration // default 30s
    DestructivePatterns []string    // rm, DROP, git push --force, etc.
}
```

### Reglas por nivel

**`read_only`** (default para tareas de análisis/consulta)
- El agente puede leer archivos y código.
- Cualquier `file_write` o `command_exec` se rechaza antes de intentarse.
- Uso típico: "explícame cómo funciona este módulo", "revisa este código y dame feedback".

**`sandboxed`** (default para tareas de desarrollo activo)
- El agente opera exclusivamente dentro de `ProjectRoot`.
- Validación de path obligatoria en `filesystem.go`:

```go
func ValidatePath(requested, root string) error {
    absRequested, err := filepath.Abs(requested)
    if err != nil { return err }
    absRoot, err := filepath.Abs(root)
    if err != nil { return err }
    if !strings.HasPrefix(absRequested, absRoot) {
        return errors.New("path fuera del sandbox permitido")
    }
    return nil
}
```
- Comandos de terminal corren con timeout y sin privilegios elevados.
- Uso típico: "implementa esta función", "corre los tests", "refactoriza este archivo".

**`trusted`** (opt-in explícito por sesión, nunca default)
- Sin restricción de path.
- Cualquier comando que matchee un patrón destructivo (`rm -rf`, `DROP TABLE`, `git push --force`, `sudo`, etc.) se marca `requires_confirmation = true` y se muestra como preview/diff en el chat antes de ejecutarse — no corre hasta que confirmes.
- Uso típico: tareas que cruzan múltiples proyectos o requieren acceso más amplio al sistema.

### Reglas duras, independientes del nivel

1. Todo comando de terminal tiene timeout configurable (default 30s), sin excepción.
2. Toda acción (`file_read`, `file_write`, `command_exec`, `browser_action`) se registra en `agent_actions` — historial completo y auditable.
3. Nada corre con privilegios de administrador/root salvo autorización explícita por esa sesión específica (no persiste entre sesiones).
4. El nivel de permiso se define por **proyecto o por tarea**, no globalmente — puedes tener un proyecto en `sandboxed` y otro en `read_only` simultáneamente.

---

## 5. Sistema de memoria (3 capas)

### Capa 1 — Memoria de proyecto (explícita, editable)
- Un archivo `.md` vivo por proyecto, similar a `CLAUDE.md` — convenciones, decisiones de arquitectura, estado actual.
- Se guarda en `projects.instructions_md` y también existe como archivo físico dentro del `ProjectRoot` para que lo edites directamente con tu editor si prefieres.
- El agente lo carga completo al iniciar cualquier tarea sobre ese proyecto — no depende de búsqueda semántica, es contexto garantizado.

### Capa 2 — Memoria episódica (vectores, Qdrant)
- Al completar (o fallar) una `agent_task`, se genera un resumen: qué se pidió, qué se hizo, qué archivos se tocaron, resultado.
- Ese resumen se embebe y se guarda en Qdrant con `source: episodic`, `project_id`, `task_id`.
- Al iniciar una tarea nueva, se hace búsqueda semántica filtrada por proyecto → se inyectan los 3-5 episodios más relevantes como contexto ("¿ya hice algo parecido antes?").

### Capa 3 — Memoria de trabajo (scratch, en memoria RAM, no persistida)
- Mientras una tarea multi-paso está `running`, el `planner`/`executor` mantienen el plan actual y resultados intermedios en memoria del proceso.
- Solo se persiste a las capas 1 y 2 cuando la tarea termina (`completed`/`failed`) — evita ensuciar la memoria de largo plazo con ruido de pasos intermedios.

### Indexado de código (no código crudo en vectores)
- `agent/code_index.go` indexa por símbolo/función (usando un parser AST ligero en Go, o `ctags` como fallback simple).
- Se embebe una **descripción** de qué hace cada función/archivo, no el código fuente completo.
- Esto permite consultas tipo "dónde está la lógica de rate limiting" sin cargar el repo completo en cada prompt, ahorrando contexto y tokens.

---

## 6. Roadmap de implementación (reordenado)

| Fase | Contenido |
|---|---|
| **Fase 1 — Scaffolding** | Estructura Go, SQLite + migraciones, modelos, router, modo local sin auth |
| **Fase 2 — Chat funcional** | Provider router (Anthropic + OpenAI + OpenRouter), WebSocket streaming, CRUD chats/messages, FTS5 |
| **Fase 3 — Memoria** | Qdrant, embeddings, chunking, memory import, memoria de proyecto (capa 1), memoria episódica (capa 2) |
| **Fase 4 — Agent System** | Planner/executor, permisos por nivel, filesystem sandboxed, terminal con timeout, code_index, audit log |
| **Fase 5 — MCP + Skills** | Cliente MCP, skill executor, connector CRUD |
| **Fase 6 — Browser + Sidebar Assist** | Browser control (chromedp/rod), screen capture, sugerencias contextuales — evaluado como *opcional* dado el peso de dependencias |
| **Fase 7 — Modelos locales** | LM Studio provider, embeddings locales |

**Nota sobre Fase 6:** dado el costo de instalar Chromium (~300MB) y la complejidad de automatización de navegador, esta fase queda marcada como candidata a *feature flag opcional* — se activa solo si el usuario la necesita, para no inflar la instalación base de quien solo quiere el asistente de código.

---

## 7. Pendientes a validar antes de Fase 1

- [ ] Confirmar `modernc.org/sqlite` vs `mattn/go-sqlite3` (pure Go vs cgo — impacta cross-compilation)
- [ ] Definir lista inicial de `DestructivePatterns` para el nivel `trusted`
- [ ] Decidir si Qdrant corre embebido/standalone o sigue en Docker Compose
- [ ] Definir formato exacto del resumen episódico (¿plantilla fija o generado libremente por el LLM?)

---

*Documento actualizado — Julio 2026. Reemplaza la sección de persistencia y agent system del plan original.*
