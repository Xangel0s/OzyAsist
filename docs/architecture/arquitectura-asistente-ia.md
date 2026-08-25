# Plan de Completitud — OzyAssist
### De MVP parcial (chat + CRUD) a sistema funcional completo

---

## 0. Realidad actual (punto de partida)

Backend ~38% real, frontend solo `chatStore` conectado. Todo lo demás — memoria, agente, MCP, skills, imágenes, sidebar — son stubs o mock local. Esto NO es un defecto del plan original; es simplemente que todavía no hemos llegado a esas fases. El roadmap de 7 fases que diseñamos sigue siendo válido, solo falta ejecutarlo.

**Antes de sumar features nuevas (grafos, OCR, visión):** hay que decidir si entran en el MVP o en una v2. Mi recomendación abajo en la sección 6.

---

## 1. Principio de orden: no construir sobre stubs

Cada fase de aquí en adelante depende de que la anterior esté *realmente* conectada, no solo diseñada. Por eso el orden no es "lo más impresionante primero", sino "lo que bloquea a lo demás primero".

```
Frontend↔Backend real (todos los stores)
        ↓
Memoria (Caps 1 y 2 — vectores)
        ↓
Agente (planner/executor + permisos)
        ↓
Skills + MCP Connectors (CRUD real)
        ↓
Imágenes / OCR
        ↓
Sidebar con visión de pantalla
        ↓
Grafos (evaluar si realmente se necesita)
```

---

## 2. Fase 2f (terminar lo empezado) — Conectar TODO el frontend

Ahora mismo solo `chatStore` habla con el backend real. Faltan:

| Store | Qué falta |
|---|---|
| `projectsStore` | Conectar a `api.projects.*` (backend ya existe, listo para usar) |
| `connectorsStore` | Backend no existe todavía — ver Fase 5 |
| `skillsStore` | Backend no existe todavía — ver Fase 5 |
| `memoryStore` | Reemplazar localStorage por llamadas reales — ver Fase 3 |
| `authStore` | Se queda local por ahora (modo single-user, correcto como está) |
| `search` (SearchModal) | Conectar a `/api/search` real — ver Fase 3 |

**Esfuerzo:** bajo (1-2 días). Es solo conectar `projectsStore`, el resto depende de fases futuras.

**Acción inmediata:** termina `projectsStore` ahora mismo, ya que el backend está listo y es trabajo perdido dejarlo mock.

---

## 3. Fase 3 — Memoria real (Caps 1 y 2)

Esta es la fase que le da sentido a "recuerdos" — sin ella, todo lo demás (agente, skills) opera sin contexto.

### 3.1 — Infraestructura
- Levantar Qdrant (Docker o binario standalone — decidir cuál)
- Conectar `nomic-embed-text-v1.5` vía LM Studio (ya decidido, falta implementar el cliente HTTP hacia LM Studio)
- Implementar `internal/memory/embeddings.go` real (llamada HTTP a LM Studio, no stub)

### 3.2 — Capa 1 (memoria de proyecto, explícita)
- `project_memory.go`: leer/escribir el `.md` en `ProjectRoot`
- Endpoint para que el frontend edite ese `.md` directamente
- Cargar el `.md` completo como contexto en cada chat asociado a un proyecto

### 3.3 — Capa 2 (memoria episódica, vectores)
- `chunker.go` real: trocear texto en 512 tokens / 64 overlap
- `qdrant.go` real: cliente HTTP a Qdrant (insert + search filtrado por `project_id`/`user_id`)
- Pipeline de import: `.md` → chunks → embeddings → Qdrant
- Auto-memoria de chat: cada intercambio se embebe y guarda (evaluar si esto es necesario desde el día 1 o si empiezas solo con import manual + memoria de proyecto)
- Búsqueda semántica: top-5 chunks relevantes inyectados en el prompt de cada mensaje nuevo

### 3.4 — FTS5 (ya tienes la tabla, falta usarla)
- Conectar `SearchModal` del frontend a un endpoint real `/api/search?q=`
- Query híbrida: FTS5 (texto exacto) + Qdrant (semántico) combinados

**Esfuerzo:** alto (1-1.5 semanas). Es la fase más importante de todas — sin esto, "memoria" es una palabra vacía en el proyecto.

**Decisión pendiente antes de empezar:** ¿Qdrant en Docker Compose o binario standalone? Standalone es más simple para un usuario que ni sabe qué es Docker.

---

## 4. Fase 4 — Agent System real

Ya tienes el diseño completo (permisos, sandbox, audit log) — falta la implementación.

| Componente | Qué hacer |
|---|---|
| `planner.go` | Recibe un goal en texto, llama al LLM pidiendo que descomponga en pasos JSON estructurados |
| `executor.go` | Ejecuta cada paso secuencialmente, decide qué tool usar (file_read/write, command, browser) |
| `permissions.go` | Implementar `ValidatePath` real (ya está el pseudocódigo en la arquitectura), chequeo de patrones destructivos |
| `filesystem.go` | Leer/escribir archivos reales dentro del sandbox validado |
| `terminal.go` | Ejecutar comandos con `exec.CommandContext` + timeout real |
| `audit.go` | Insertar cada acción en `agent_actions` de verdad |
| `code_index.go` | Empezar simple: usar `ctags` como subproceso, no un parser AST propio (ahorra semanas) |

**Importante:** el planner necesita la memoria episódica (Fase 3) para no repetir errores/decisiones — por eso va después, no antes.

**Esfuerzo:** alto (1.5-2 semanas). Es la fase más compleja técnicamente, especialmente el executor con manejo de errores por paso.

**Empieza en `sandboxed` únicamente.** No implementes `trusted` hasta que `read_only` y `sandboxed` estén probados con casos reales — es donde vive el riesgo de que el agente borre algo.

---

## 5. Fase 5 — Skills + MCP Connectors (CRUD real)

Ahora mismo el frontend tiene UI completa pero todo local — hay que construir el backend real detrás.

### Skills
- `skills/executor.go`: ejecutar un skill según su `execution_type` (script/prompt_template/api_call)
- CRUD real en `handlers/skills.go` (reemplazar los 501)
- Conectar `skillsStore` del frontend

### MCP Connectors
- `connectors/mcp_client.go`: implementar el protocolo MCP real (hay SDKs de referencia, no reinventar el wire format)
- CRUD real en `handlers/connectors.go`
- Conectar `connectorsStore` del frontend

**Esfuerzo:** medio (1 semana). MCP tiene spec pública, así que es más "integrar" que "diseñar".

---

## 6. Features nuevas que mencionaste — evaluación honesta

### Imágenes + OCR fallback
**Viable y razonable para el MVP.** Plan concreto:
- Si el modelo activo soporta visión nativa (GPT-4o, Claude con imágenes) → mandar la imagen directo en el mensaje, sin OCR.
- Si no la soporta → correr OCR local (Tesseract vía binding Go, o un modelo OCR chico en LM Studio) y mandar el texto extraído como contexto.
- Esto es una capa relativamente aislada — se puede meter en paralelo a la Fase 5 sin bloquear nada.
- **Esfuerzo:** medio (3-5 días).

### Sidebar con visión de pantalla ("computer use")
**Viable, pero es la pieza más pesada de todas.** Necesitas:
- Captura de pantalla periódica desde Tauri (permisos de OS)
- Envío al backend, que lo manda a un modelo con visión
- Esto es esencialmente construir tu propio "Claude Computer Use" — funcional, pero espera que tome más tiempo del que parece a primera vista, sobre todo afinando qué tan seguido capturar sin saturar tokens/costos.
- **Esfuerzo:** alto (1-2 semanas), y depende de que Fase 4 (agente) ya esté sólida si quieres que además pueda *actuar*, no solo observar.

### Grafos (knowledge graph)
**Mi recomendación honesta: NO para el MVP.** Ya lo habíamos descartado al inicio por buena razón — los vectores (Fase 3) resuelven la mayoría de los casos de "recordar contexto relacionado". Un grafo de conocimiento agrega complejidad de modelado (¿qué son los nodos? ¿qué son las aristas? ¿cómo se actualizan sin corromper relaciones viejas?) que no se justifica hasta que tengas evidencia real de que los vectores se quedan cortos. Si después de usar el sistema unas semanas sientes que falta razonamiento relacional explícito (tipo "este bug está relacionado con esa decisión de arquitectura de hace 2 meses"), ahí lo evaluamos con datos reales en vez de especulación.

---

## 7. Orden final recomendado (con esfuerzo estimado)

| # | Fase | Esfuerzo | Bloquea a |
|---|---|---|---|
| 1 | Terminar Fase 2f (`projectsStore` conectado) | 1-2 días | Todo lo demás en frontend |
| 2 | Fase 3 — Memoria (Qdrant + embeddings + FTS5 real) | 1-1.5 semanas | Agente, Skills, calidad de todo el sistema |
| 3 | Fase 4 — Agent System (solo read_only + sandboxed) | 1.5-2 semanas | Sidebar con acción real |
| 4 | Fase 5 — Skills + MCP real | 1 semana | — |
| 5 | Imágenes + OCR fallback | 3-5 días | — (paralelo a Fase 5 si quieres) |
| 6 | Sidebar con visión de pantalla | 1-2 semanas | Requiere Fase 4 sólida si quieres acción, no solo observación |
| 7 | Grafos | — | Evaluar después de usar el sistema, no antes |

**Total estimado hasta tener "Claude Desktop con memoria real, agente, skills, MCP e imágenes":** aproximadamente 6-8 semanas de trabajo enfocado, sin contar sidebar con visión (que sumaría 1-2 semanas más) ni grafos.

---

## 8. Próximo paso inmediato

Terminar `projectsStore` → backend ya existe, es trabajo casi gratis dejarlo mock más tiempo. Después, arrancar Fase 3 (memoria) de lleno, porque es la que le da sentido real a "recuerdos" y desbloquea todo lo que sigue.

---

*Documento generado a partir de la auditoría del sistema — Julio 2026.*
