# Auditoría de Hallazgos — OzyAssist

**Fecha:** 2026-07-13  
**Alcance:** Backend (Go), Frontend (React/TypeScript), Base de datos (SQLite)  
**Total hallazgos:** ~60

---

## Tabla de Contenido

- [🔴 Críticos](#-críticos-9)
- [🟠 Altos](#-altos-12)
- [🟡 Medios](#-medios-18)
- [🔵 Bajos](#-bajos-20)

---

## 🔴 Críticos (9)

### C1 — Mensajes no persistidos en flujo de agente

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Líneas** | 110-114, 320-327 |
| **Descripción** | Cuando se dispara el flujo de agente (consent "always" desde `handleChatMessage` línea 110, o "always"/"once" desde `handleConsentResponse` líneas 325/327), el mensaje del usuario **nunca se guarda** en la tabla `messages`. El backend retorna antes de llegar a `handleChatMessageNormal` que es la única ruta que persiste mensajes. El resultado del agente tampoco se persiste. |
| **Impacto** | Al recargar la página, el mensaje del usuario y la respuesta del agente desaparecen del historial del chat. |
| **Fix** | Guardar el mensaje user al inicio de `handleChatMessage` (antes del check de consent), y persistir el resultado del agente en `runAgentTask` vía `db.CreateMessage`. |

---

### C2 — Data race en `delete(clients, client)` bajo read lock

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/websocket.go` |
| **Línea** | 103 |
| **Descripción** | `Broadcast()` adquiere `mu.RLock()` (read lock) pero ejecuta `delete(clients, client)` que es una operación de escritura sobre el mapa `clients`. Ejecución concurrente con `HandleWebSocket` (que adquiere `mu.Lock()`) produce data race. |
| **Impacto** | Crash por mapa concurrente no seguro (`concurrent map iteration and map write`). |
| **Fix** | Usar `mu.Lock()` en lugar de `mu.RLock()` en `Broadcast`, o extraer el delete a un bloque separado con write lock. |

---

### C3 — Data race en `delete(c.buf, key)` bajo read lock en Caps3

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/memory/caps3.go` |
| **Línea** | 55 |
| **Descripción** | `Caps3Store.Get()` adquiere `c.mu.RLock()` pero ejecuta `delete(c.buf, key)` al expirar una entrada — operación de escritura bajo read lock. |
| **Impacto** | Data race con `Caps3Store.Set()` que escribe en el mismo mapa. |
| **Fix** | Adquirir `c.mu.Lock()` antes de hacer `delete`, o mover la limpieza a un método separado con write lock. |

---

### C4 — Concurrent WebSocket write desde dos goroutines

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/websocket.go:90` + `chat.go:374` |
| **Líneas** | `websocket.go:90` (writePump), `chat.go:374` (writeJSON) |
| **Descripción** | `writeJSON()` (llamado desde `readPump`/manejadores) y `writePump()` escriben al mismo `conn.WriteMessage`. La librería `gorilla/websocket` **no es segura para escritura concurrente**. |
| **Impacto** | Data race, potencial corrupción de datos en WebSocket, panic. |
| **Fix** | Centralizar toda escritura a través de `client.Send` channel (writePump). `writeJSON` debe enviar al channel, no escribir directo al conn. |

---

### C5 — Errores `tx.Exec` ignorados en DeleteProject y DeleteChat

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/queries.go` |
| **Líneas** | 139-142, 153-154 |
| **Descripción** | En `DeleteProject` y `DeleteChat`, los retornos de `tx.Exec` (que contienen errores) se descartan completamente. Si una sentencia falla, la transacción igual se commitea parcialmente. |
| **Impacto** | Inconsistencia de datos: un proyecto puede eliminarse pero sus `chats` o `code_graph_edges` quedar huérfanos. |
| **Fix** | Verificar cada `err` después de cada `tx.Exec` y hacer `tx.Rollback()` si alguno falla. |

---

### C6 — `isFirstMsg` off-by-one

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts` |
| **Línea** | 213 |
| **Descripción** | `chat?.messages.length === 1` chequea si es el primer mensaje, pero en ese punto ya se agregaron 2 mensajes (user + assistant placeholder). La condición nunca es `true`, por lo que el assistant placeholder vacío nunca se remueve y `isResponding` nunca se setea a `false` en el flujo de consentimiento. |
| **Impacto** | El usuario ve un bubble assistant vacío detrás del ConsentModal. `isResponding` queda `true`. |
| **Fix** | Cambiar a `chat?.messages.length === 2` o mejor, trackear `isFirstMessage` con una flag separada antes de agregar mensajes. |

---

### C7 — `dismissConsent` no cancela WS session

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts` |
| **Líneas** | 365-367 |
| **Descripción** | `dismissConsent()` setea `consentPending: null` pero no llama `wsClient.cancelStream()`. El `this.session` interno de `ws.ts` queda activo. Al intentar enviar otro mensaje, `wsClient.sendMessage()` rechaza con "Ya hay un streaming en curso". |
| **Impacto** | Usuario bloqueado tras dismiss — no puede enviar más mensajes sin recargar la página. |
| **Fix** | `dismissConsent` debe llamar `wsClient.cancelStream(consentPending.chatId)` antes de limpiar estado. |

---

### C8 — Sin Error Boundary en React

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/main.tsx`, `frontend/src/App.tsx` |
| **Descripción** | No hay ningún React Error Boundary en el árbol de componentes. Si cualquier componente lanza error en render, React desmonta todo el árbol y muestra pantalla en blanco. |
| **Impacto** | Cualquier crash de componente = aplicación inutilizable hasta recargar. |
| **Fix** | Envolver `<App />` (y/o páginas individuales) en un ErrorBoundary con UI de fallback y botón "Recargar". |

---

### C9 — `loadChats` nunca llamado en init

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts` |
| **Líneas** | 97-113 |
| **Descripción** | A diferencia de `projectsStore`, `skillsStore` y `connectorsStore` que llaman `.getState().loadXxx()` al definir el store, `chatStore.loadChats()` nunca se invoca en la inicialización. El array `chats` arranca vacío. |
| **Impacto** | Al recargar la página, el sidebar "Recientes" muestra vacío aunque haya chats existentes. |
| **Fix** | Agregar `useChatStore.getState().loadChats()` tras la definición del store. |

---

## 🟠 Altos (12)

### H1 — Error `GetProject` ignorado → bypass de consentimiento

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Líneas** | 106-117 |
| **Descripción** | Si `db.GetProject(chat.ProjectID)` falla (proyecto eliminado, error DB), el error se ignora completamente y el mensaje cae a `handleChatMessageNormal` — **bypasseando todo el flujo de consentimiento del agente**. |
| **Impacto** | Proyecto eliminado pero con chats activos → comandos de agente se ejecutan sin consentimiento. |
| **Fix** | Si `GetProject` falla, retornar error al cliente en lugar de hacer fallthrough. |

---

### H2 — Path traversal en download de archivos

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/handlers/files.go` |
| **Línea** | 54 |
| **Descripción** | `filepath.Join(uploadDir, c.Param("id"))` usa directamente el parámetro `id` del usuario sin sanitización. `GET /api/files/../../etc/passwd` resuelve fuera del directorio uploads. |
| **Impacto** | Lectura de archivos arbitrarios del sistema. |
| **Fix** | Usar `filepath.Base()` o validar que la ruta resuelta esté dentro de `uploadDir`. |

---

### H3 — Path traversal en `att.ID` de OCR

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Línea** | 60 |
| **Descripción** | `filepath.Join("data/uploads", att.ID)` sin validación de `att.ID`. Un cliente puede enviar `att.ID = "../../etc/passwd"`. |
| **Impacto** | Potencial lectura/OCR de archivos arbitrarios. |
| **Fix** | Validar que `att.ID` no contenga `..` ni separadores de path. |

---

### H4 — Filtro Caps2 sobrescribe filtro de `projectID`

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/memory/caps2.go` |
| **Líneas** | 78-88 |
| **Descripción** | Cuando se proveen `projectID` y `userID`, el `filter["must"]` del segundo sobrescribe al primero. La búsqueda Qdrant solo aplica el filtro de `userID`, perdiendo el filtro de proyecto. |
| **Impacto** | Búsqueda semántica retorna resultados de otros proyectos. |
| **Fix** | Combinar ambos filtros en un solo `filter["must"]` con múltiples condiciones. |

---

### H5 — Directorios excluidos no se saltan en Windows

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/handlers/index.go` |
| **Línea** | 76 |
| **Descripción** | `filepath.ToSlash(relPath)` convierte `\` a `/`, luego `strings.Split(relPath, string(filepath.Separator))` divide por `\`. En Windows, `Separator` es `\` pero `relPath` ya usa `/`. El split devuelve el path completo como un elemento, por lo que `node_modules`, `.git`, etc. nunca se excluyen. |
| **Impacto** | El indexador de proyectos procesa archivos dentro de `node_modules` en Windows, potencialmente millones de archivos. |
| **Fix** | Usar `strings.Split(relPath, "/")` después de `ToSlash`. |

---

### H6 — `cancel()` nunca llamado en flujo normal de streaming

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Líneas** | 125-135 |
| **Descripción** | `context.WithCancel` crea un `cancel()` que solo se invoca desde `CancelStream`. En el flujo normal (channel `chunkCh` se cierra), `cancel()` nunca se llama. El contexto y sus resources quedan hasta GC. |
| **Impacto** | Goroutine/context leak por cada streaming que completa normalmente. |
| **Fix** | Agregar `defer cancel()` después de crear el context, o llamar `cancel()` después del loop de chunks. |

---

### H7 — REST `CreateTask` retorna 200 OK incluso en error

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/handlers/agent.go` |
| **Líneas** | 68-71 |
| **Descripción** | Si `executor.ExecutePlan` retorna error, el handler continúa y retorna 200 con datos parciales en vez de 500 con el error. |
| **Impacto** | Cliente REST asume tarea completada exitosamente cuando en realidad falló. |
| **Fix** | Verificar `err` después de `ExecutePlan` y retornar 500 si es error. |

---

### H8 — Sin ping/pong en WebSocket

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/websocket.go` |
| **Líneas** | 51-86 |
| **Descripción** | No se configura `conn.SetPingHandler` ni `conn.SetPongHandler`. Si la conexión queda half-open (network partition), `ReadMessage` bloquea para siempre. |
| **Impacto** | Goroutine leak por cada conexión que se cae silenciosamente. |
| **Fix** | Configurar ping/pong handlers con timeout (`conn.SetReadDeadline`). |

---

### H9 — Foreign Keys pueden quedar OFF si migración 003 falla

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/migrations/003_agent_status.up.sql` |
| **Líneas** | 1, 20 |
| **Descripción** | La migración hace `PRAGMA foreign_keys = OFF` al inicio y `ON` al final. Si falla entre medio (ej: DROP TABLE成功, ALTER TABLE falla), foreign_keys quedan OFF permanentemente en la única conexión. |
| **Impacto** | Integridad referencial silenciosamente perdida para toda la sesión. |
| **Fix** | Usar `defer` para restaurar `foreign_keys = ON`, o evitar desactivarlas usando `ALTER TABLE` directamente. |

---

### H10 — Faltan índices en varias tablas

| Tabla | Columna | Query | Archivo:Línea |
|-------|---------|-------|---------------|
| `messages` | `chat_id` | `WHERE chat_id = ? ORDER BY created_at` | `queries.go:425` |
| `memory_entries` | `user_id` | `WHERE user_id = ? ORDER BY created_at DESC` | `queries.go:259` |
| `skills` | `user_id` | `WHERE user_id = ? ORDER BY created_at DESC` | `queries.go:314` |
| `connectors` | `user_id` | `WHERE user_id = ? ORDER BY created_at DESC` | `queries.go:359` |
| `chats` | `created_at` | `ORDER BY created_at DESC` | `queries.go:63` |
| `projects` | `created_at` | `ORDER BY created_at DESC` | `queries.go:103` |

**Impacto:** Full table scan + filesort en cada consulta. Degrada con datos crecientes.

---

### H11 — Consent "no" deja assistant message vacío

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts` |
| **Líneas** | 355-357 |
| **Descripción** | Cuando `resolveConsent("no")` envía la decisión, el backend responde con `done` pero sin `text` events. `onDone` solo setea `isResponding: false` sin escribir contenido en el assistantMsg vacío. |
| **Impacto** | El usuario ve un bubble assistant permanente y vacío. |
| **Fix** | `onDone` debe setear contenido fallback si `pendingContent` está vacío (ej: "[Acción rechazada]"). |

---

### H12 — `createChat` sin await en ChatPage

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/components/Chat/ChatPage.tsx` |
| **Líneas** | 38-41 |
| **Descripción** | `createChat("chat")` es async pero no se usa `await`. Si falla (retorna null), `activeChatId` nunca se actualiza y el mensaje del usuario se pierde silenciosamente. |
| **Impacto** | Mensaje perdido sin feedback al usuario. |
| **Fix** | Usar `chatId = await createChat("chat")` y manejar null. |

---

## 🟡 Medios (18)

### M1 — OCR no procesado en path de agente

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Líneas** | 138 vs 334 |
| **Descripción** | `processAttachments` se llama solo en `handleChatMessageNormal` (línea 138). `runAgentTask` (línea 334) nunca procesa attachments. Imágenes adjuntadas en mensajes que disparan agente no reciben OCR. |
| **Fix** | Llamar `processAttachments` al inicio de `runAgentTask` o antes del check de consentimiento. |

### M2 — Error `GetMessages` silencioso

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/api/ws/chat.go` |
| **Líneas** | 162-164 |
| **Descripción** | Si `db.GetMessages` falla, solo se loggea. El chat continúa con historial vacío, sin notificar al usuario. |
| **Fix** | Retornar error al cliente si no se puede cargar el historial. |

### M3 — Doble query a `GetProject` en `CreateAndExecuteTask`

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/agent/taskrunner.go` |
| **Líneas** | 17-24, 38-46 |
| **Descripción** | El proyecto se consulta dos veces (para instructions + sandbox) en la misma función. Resultados pueden ser inconsistentes si el proyecto cambia entre queries. |
| **Fix** | Cachear el resultado de la primera query y reusarlo. |

### M4 — `MemoryEntry.CreatedAt` tipo `string` en vez de `time.Time`

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/models/memory.go:11`, `queries.go:271` |
| **Descripción** | Único modelo que usa `string` para `CreatedAt`. Todos los demás usan `time.Time`. |
| **Fix** | Cambiar a `time.Time` y actualizar el scan. |

### M5 — `AgentTask.CompletedAt` nunca seleccionado

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/queries.go:399-401` |
| **Descripción** | `completed_at` se escribe (`CURRENT_TIMESTAMP` en `UpdateTaskStatus`) pero no se incluye en el SELECT de `GetTask`. El campo siempre es `nil`. |
| **Fix** | Agregar `completed_at` al SELECT de `GetTask`. |

### M6 — `AgentAction` modelo write-only

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/models/agent_task.go:18-28` |
| **Descripción** | `CreateAgentAction` escribe, `ConfirmAction` actualiza, pero no hay SELECT que lea `agent_actions`. |
| **Fix** | Agregar query de lectura si es necesaria para UI de auditoría. |

### M7 — Sin paginación en listados

| Endpoint | Tabla | Archivo:Línea |
|----------|-------|---------------|
| `ListChats` | chats | `queries.go:61` |
| `ListProjects` | projects | `queries.go:101` |
| `ListSkills` | skills | `queries.go:314` |
| `ListConnectors` | connectors | `queries.go:359` |
| `GetMessages` | messages | `queries.go:425` |

**Fix:** Agregar `LIMIT ? OFFSET ?` con valores predeterminados razonables.

### M8 — Variable global `DB` exportada

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/sqlite.go:20`, `router.go:101` |
| **Descripción** | `var DB *sql.DB` es exportada (mayúscula). Código externo (`router.go:101`) ejecuta SQL directamente sin pasar por `queries.go`. |
| **Fix** | Hacerla privada y exponer solo las funciones de `queries.go`. |

### M9 — `completed_at` se setea incondicionalmente

| Campo | Valor |
|-------|-------|
| **Archivo** | `backend/internal/db/queries.go:393` |
| **Descripción** | `UpdateTaskStatus` setea `completed_at = CURRENT_TIMESTAMP` en **cada** actualización, incluso para transiciones no-terminales (ej: "running" → "planning"). |
| **Fix** | Solo setear `completed_at` cuando el nuevo status es terminal ("completed", "failed", "cancelled"). |

### M10 — Search modal no cambia de vista

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/components/Search/SearchModal.tsx:74-77` |
| **Descripción** | `handleSelectChat` llama `setActiveChat` pero no cambia `activeView`. Buscar un chat code-mode y clickearlo deja al usuario en la vista anterior. |
| **Fix** | Pasar `mode` como parámetro y llamar `setActiveView(mode === "code" ? "code" : "chat")`. |

### M11 — Tree fetch failure muestra spinner permanente

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/components/Code/ProjectContext.tsx:51-55` |
| **Descripción** | El catch de `api.projects.tree` setea `treeData = null` (igual que estado inicial). La UI no distingue entre "cargando" y "error". |
| **Fix** | Usar estado `treeError` separado para mostrar mensaje de error con retry. |

### M12 — `regenerateMessage` ignora pending consent

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts:272-296` |
| **Descripción** | Si hay un `consentPending` activo, `regenerateMessage` llama `sendMessage` que vuelve a disparar `onConsentRequired`. Loop infinito. |
| **Fix** | Agregar `if (get().consentPending) return;` al inicio de `regenerateMessage`. |

### M13 — WS session no se limpia al recibir `consent_required`

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/services/ws.ts:84` |
| **Descripción** | `consent_required` no setea `this.session = null` (a diferencia de `done`/`error`). La session del `sendMessage` original queda activa indefinidamente. |
| **Fix** | Setear `this.session = null` después de despachar `onConsentRequired`. |

### M14 — `sendConsentResponse` falla silenciosamente si WS cerrado

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/services/ws.ts:121` |
| **Descripción** | Si WS está cerrado, `sendConsentResponse` hace `return` sin notificar. El usuario ve el modal pero su decisión se pierde. |
| **Fix** | Retornar booleano, y `resolveConsent` debe mostrar error al usuario si falla. |

### M15 — `updateChatTitle` no persiste al backend

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/chatStore.ts:90-95` |
| **Descripción** | Solo actualiza estado local. Al recargar, el título vuelve al original. |
| **Fix** | Agregar `await api.chats.update(chatId, { name: title })`. |

### M16 — Sin loading states

| Componente | Archivo |
|-----------|---------|
| SkillsPage | `components/Skills/SkillsPage.tsx` |
| ConnectorsPage | `components/Connectors/ConnectorsPage.tsx` |
| `loadMessages` | `store/chatStore.ts:372-389` |

**Impacto:** Usuario no ve indicador de carga en Skills, Connectors, ni al cambiar de chat.

### M17 — Project deletion no limpia chats locales

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/store/projectsStore.ts:79-89` |
| **Descripción** | Al eliminar proyecto, se remueve de `projects` pero no se actualiza `projectId` en los chats del `chatStore`. Los chats code-mode siguen referenciando el proyecto eliminado. |
| **Fix** | Disparar actualización de chats con projectId proyectado = null en chatStore. |

### M18 — `cancelStream` existe pero nunca se invoca

| Campo | Valor |
|-------|-------|
| **Archivo** | `frontend/src/services/ws.ts:130` |
| **Descripción** | El método `cancelStream` está implementado pero ninguna parte del frontend lo llama. |
| **Fix** | Integrarlo en los flujos de cancelación (dismissConsent, cambio de chat, etc.). |

---

## 🔵 Bajos (20+)

| # | Hallazgo | Archivo | Línea |
|---|----------|---------|-------|
| B1 | Tipos `any` en lugar de tipos concretos (9+ ocurrencias) | `chatStore.ts:100,339,351,375`, `ws.ts:53,113`, `api.ts:104,145,162-173`, `AnalyzeProjectStep.tsx:63` | Varias |
| B2 | `as any` para `source: "chat"` bypassea union type | `store/chatStore.ts` | 351 |
| B3 | `dangerouslySetInnerHTML` en CodeArtifact | `components/Code/CodeArtifact.tsx` | 58 |
| B4 | `document.execCommand` deprecado en MenuBar | `components/Layout/MenuBar.tsx` | 67-74 |
| B5 | `sidebarMode` state toggled pero nunca leído | `components/Layout/Sidebar.tsx` | 25 |
| B6 | Botones decorativos sin handlers (TopAppBar) | `components/Layout/TopAppBar.tsx` | 11-61 |
| B7 | Botón "Editar" sin onClick (ProjectDetail) | `components/Code/ProjectDetail.tsx` | 52-53 |
| B8 | Botones "Adjuntar archivo" no funcionales | `CodeInput.tsx:45-49`, `ChatInput.tsx:43-48`, `ChatPage.tsx:106-108` | Varias |
| B9 | ModelSelector estático sin API integration | `components/Common/ModelSelector.tsx` | - |
| B10 | Hardcoded "Medio"/"Alto" labels por provider | `components/Chat/ChatPage.tsx` | 183 |
| B11 | `memoryStore.addEntry` para respuestas vacías | `store/chatStore.ts` | 239-244 |
| B12 | `updateProfileMd` con null user resetea auth state | `store/authStore.ts` | 30 |
| B13 | `importFromMd` no awaited en MemoryImportStep | `components/Onboarding/MemoryImportStep.tsx` | 29-31 |
| B14 | Tauri-only dialog falla silenciosamente en browser | `components/Code/AnalyzeProjectStep.tsx` | 40-41 |
| B15 | Sin validación visual en WelcomeScreen | `components/Onboarding/WelcomeScreen.tsx` | 52-58 |
| B16 | Magic number 3 para onboarding steps | `components/Layout/Sidebar.tsx` | 125 |
| B17 | Sin keyboard accessibility en ConsentModal (Escape, click-outside) | `components/Code/ConsentModal.tsx` | - |
| B18 | Sin arrow-key navigation en SearchModal | `components/Search/SearchModal.tsx` | - |
| B19 | `EnsureDefaultUser` ignora error de primer Scan | `queries.go` | 14 |
| B20 | Modelo `User.DefaultProvider`/`DefaultModel` nunca leídos | `models/user.go:9-10` | - |
| B21 | COALESCE redundante cuando columna tiene DEFAULT | `queries.go:90,102,315,335,360,426` | Varias |
| B22 | LIKE sin escapar `%`/`_` en búsqueda FTS | `queries.go` (SearchChats/SearchProjects) | Varias |
| B23 | Sin graceful shutdown para streams activos | `cmd/server/main.go` | - |
| B24 | `handlers/agent.go` permission override es inverso | `handlers/agent.go:41-44` | - |
| B25 | `handlers/sidevar.go` filename typo | `handlers/sidevar.go` | - |

---

## Estadísticas

| Prioridad | Cantidad |
|-----------|----------|
| 🔴 Críticos | 9 |
| 🟠 Altos | 12 |
| 🟡 Medios | 18 |
| 🔵 Bajos | 25 |
| **Total** | **64** |

---

*Generado el 2026-07-13 mediante auditoría estática de código fuente.*
