---
name: ozy-debug-log
description: >-
  Use this skill when debugging OzyAssist issues: reading backend server logs,
  analyzing WebSocket errors, diagnosing frontend console errors, or understanding
  why a feature isn't working. Provides log format guide, key log patterns to search,
  and diagnostic commands.
---

# OzyAssist Debug & Log Analysis — Skill

## Backend Log Patterns

### Normal Startup Sequence
```
SQLite inicializado: ...data/ozyassist.db
[EDR Watchdog] Monitoreo continuo iniciado
[Smart Clipboard] Vigilante de portapapeles iniciado
Servidor iniciado en puerto 8080
[GIN-debug] Listening and serving HTTP on :8080
```

### Normal Chat Request
```
[GIN] 200 | GET  /api/chats
[GIN] 200 | GET  /api/models/available
[GIN] 200 | GET  /ws              ← WebSocket upgrade (long-lived)
[INFO] [provider/model] Running chat completion...
[INFO] [provider/model] Streaming response...
[GIN] 200 | GET  /ws              ← Connection closed (shows duration)
```

### Warning: Qdrant Not Running (Expected — OK to ignore)
```
Caps2 Qdrant insert error (will skip): dial tcp [::1]:6333: connectex: No se puede establecer una conexión
```
This is **normal** — Qdrant is optional. Memory falls back to SQLite.

### Error: Provider Not Configured
```
Provider no disponible: no providers registered
```
→ User needs to configure LM Studio/Ollama in Settings → Proveedores LLM.

### Error: WebSocket Connection Reset
```
wsarecv: Se ha anulado una conexión establecida por el software en su equipo host
```
→ Client disconnected (browser closed tab, network interruption). Normal on Windows.

## Reading Daemon Task Logs

```powershell
# View last N lines of server log (replace task-XXXX with actual task ID)
Get-Content "C:\Users\User\.gemini\antigravity-ide\brain\<conv-id>\.system_generated\tasks\task-XXXX.log" -Tail 50

# Watch live (like tail -f)
Get-Content "...\task-XXXX.log" -Wait -Tail 20
```

## Frontend Console Error Patterns

| Console Message | Cause | Fix |
|-----------------|-------|-----|
| `WebSocket connection failed: ERR_CONNECTION_REFUSED` | Backend not running | Start `ozy-server.exe` |
| `Watchdog timeout triggered in OzyLiveOverlay` | Model took >90s | Check if LM Studio is responding |
| `SpeechSynthesis error: interrupted` | TTS cancelled mid-speech | Normal — `stopSpeaking()` was called |
| `SpeechSynthesis error: not-allowed` | Mic permission denied | User must allow mic in browser |
| `Cannot read properties of null (reading 'id')` | activeChatId is null | Ensure `createChat` succeeded |

## Diagnostic Commands

```powershell
# Is the server running?
Test-NetConnection localhost -Port 8080

# What's on port 8080?
netstat -ano | findstr :8080

# Quick API health check
Invoke-RestMethod http://localhost:8080/health

# List available models
Invoke-RestMethod http://localhost:8080/api/models/available | ConvertTo-Json

# Check git for recent changes
git -C c:\Users\User\Documents\ozyAsis log --oneline -10

# Verify binary is up to date
(Get-Item "c:\Users\User\Documents\ozyAsis\backend\ozy-server.exe").LastWriteTime
```

## WebSocket Debug (Browser DevTools)

1. Open DevTools → **Network** tab → filter `WS`
2. Click the `ws` connection
3. **Messages** subtab shows all frames:
   - `→` outgoing (client to server)
   - `←` incoming (server to client — check for `type: "error"`)

Look for:
- `{"type":"agent:thinking"}` — model received request
- `{"type":"message:delta","content":"..."}` — streaming token
- `{"type":"agent:completed"}` — finished
- `{"type":"error","error":"..."}` — something went wrong

## Performance Profiling

```powershell
# Check memory usage of server process
Get-Process ozy-server | Select-Object CPU, WorkingSet, PrivateMemorySize

# Check if LM Studio is responding
Invoke-RestMethod "http://localhost:1234/v1/models" -TimeoutSec 5

# Check Ollama
Invoke-RestMethod "http://localhost:11434/api/tags" -TimeoutSec 5
```
