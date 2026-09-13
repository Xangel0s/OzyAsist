---
name: ozy-build-deploy
description: >-
  Use this skill whenever you need to build, rebuild, or redeploy OzyAssist after making
  code changes. Covers the complete build pipeline: Go backend compilation, React/Vite
  frontend build, dist copy, server restart, and browser hard-refresh procedure.
  Also covers port management and process cleanup.
---

# OzyAssist Build & Deploy — Skill

## Full Rebuild & Restart Procedure

Run this sequence after ANY code change that needs to be tested:

```powershell
# 1. Stop running server (if any daemon task is active, kill it first)
Get-Process -Name "ozy-server" -ErrorAction SilentlyContinue | Stop-Process -Force
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue |
  ForEach-Object { Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue }

# 2. Build backend (from repo root)
cd backend
go build -o ozy-server.exe ./cmd/server/main.go
# Expected: exits code 0, no output

# 3. Build frontend
cd ..\frontend
npm run build
# Expected: "✓ built in Xs" at the end

# 4. Copy dist to backend
cd ..
Copy-Item -Recurse -Force "frontend\dist" "backend\dist"

# 5. Start server (daemon)
cd backend
.\ozy-server.exe
# Expected log: "Servidor iniciado en puerto 8080"
```

## Backend-Only Change (No Frontend Change)
```powershell
# Stop existing server, rebuild Go, restart — no npm build needed
cd backend
go build -o ozy-server.exe ./cmd/server/main.go && .\ozy-server.exe
```

## Frontend-Only Change
```powershell
cd frontend
npm run build
cd ..
Copy-Item -Recurse -Force "frontend\dist" "backend\dist"
# No need to restart server — static files served from disk
# User must do Ctrl+Shift+R in browser (hard refresh)
```

## Quick Validation After Deploy

```powershell
# 1. Health check
Invoke-RestMethod http://localhost:8080/health

# 2. Check backend log for startup errors
# (look at the running daemon task log)

# 3. Type check frontend without building
cd frontend; npx tsc --noEmit
```

## Browser Hard Refresh
After every frontend rebuild, the browser MUST do a **hard refresh** to bypass cache:
- **Windows/Linux**: `Ctrl + Shift + R`
- **Mac**: `Cmd + Shift + R`

The new JS bundle gets a new hash (e.g., `index-CoUq8Y7Q.js`). If the old hash is still visible in Network tab, the browser is serving a cached version.

## Port Conflicts
If port 8080 is busy:
```powershell
# Find what's using port 8080
netstat -ano | findstr :8080

# Kill by PID
Stop-Process -Id <PID> -Force
```

## Common Build Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `cannot find package` | Missing Go import | `go mod tidy` |
| `CGO_ENABLED=0` + sqlite | SQLite requires CGO | Ensure gcc is in PATH (MinGW/MSYS2) |
| `npm run build` TS errors | TypeScript type error | Fix error, then rebuild |
| `EADDRINUSE :8080` | Previous server still running | Kill process on port 8080 |
| Browser shows old UI | Cache not cleared | `Ctrl+Shift+R` in browser |

## Environment Requirements
- Go 1.22+ with CGO enabled (sqlite3 requires gcc)
- Node.js 18+ for Vite build
- GCC in PATH (for `go-sqlite3`): verify with `gcc --version`
