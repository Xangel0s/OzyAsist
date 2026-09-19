# AGENTS.md — OzyAssist Guidelines & Instructions

## Project Overview
OzyAssist es un asistente autónomo de escritorio, código y cowork para Windows inspirado en Claude Desktop, Manus y Claude Code. Cuenta con un núcleo en Go de alto rendimiento, arquitectura Zero-Docker con SQLite en Go puro (`modernc.org/sqlite`), memoria híbrida en RAM, terminal con bucle de auto-reparación (*Self-Healing*), interfaz de consola TUI reactiva en Bubble Tea con sistema de colas, y soporte para un ecosistema multi-proveedor (Mistral AI, KiloCode Gateway, Cohere, Groq, OpenAI, Anthropic, DeepSeek y modelos locales).

## Architecture & Tech Stack
- **Core / Backend**: Go (Bubble Tea TUI / Gin REST / gorilla/websocket / pure-Go SQLite / local RAM embeddings).
- **Zero-Docker Policy**: No Docker containers. Pure-Go SQLite driver, in-memory caching, and local vector search.
- **Interfaces**:
  - `cmd/ozy`: CLI / TUI interactiva basada en Bubble Tea con menú de inicio retro (años 80-90) navegable con flechas y soporte de mouse (`↑`/`↓`/`Enter`: Iniciar conversación, Historial de conversaciones, Configuraciones, Salir), pantalla de historial de chats persistidos en SQLite con buscador en tiempo real, reanudación y borrado interactivo (`d` + confirmación), panel de diagnóstico BIOS, transición fluida al chat agéntico moderno directo con encabezado sobrio de sesión (`#shortID · <Tema>`), **Ventana Lateral de Contexto Interactiva (Sidebar)** colapsable con `Ctrl+B` o `/context` (telemetría de tokens vs límite de modelo con barra de progreso, métricas de RAM en Go puro `runtime.MemStats`, conteo de proyectos en RAM y recuerdos activos en SQLite), **Gestor Visual Interactivo de Memorias** (`/memmgr`, `/memories gui`) con buscador en tiempo real, teclas `a` para agregar preferencias/stack, `d` para eliminar con confirmación `s`/`n`, y comandos de chat `/forget <num|id>` y `/forgetall`, **Chat Refinado de Alto Contraste** (usuario en gris neutro `#888888`, agente con encabezado `O Ozy:` en Neon Lime `#d1f107`, sin colores azules/cyan `#00d2d3`), **Playground Plegado por Defecto** (`showThinking: false`: no muestra ninguna traza de ejecución mientras está colapsado, desplegable con `Ctrl+T` o `/thinking`), barra de estado inferior no redundante, auto-limpieza de sesiones vacías en SQLite, barra de entrada con badge `[OZY]` en Neon Lime `#d1f107`, comandos `/new` y `/history`, colas de mensajes, soporte nativo de mouse (`tea.WithMouseCellMotion`) y superficies atenuadas sin emojis.
  - `cmd/server`: Servidor HTTP y WebSocket para integraciones cliente y streaming reactivo.
  - `cmd/ozyctl`: Herramienta de utilidades y administración.
- **Cognitive Architecture & Subagents**:
  - **OZY**: Ejecutor de acciones y llamadas a herramientas del sistema/MCP.
  - **CHARC**: Auditor de calidad que valida precondiciones y audita resultados antes de confirmarlos.
  - **NINE**: Estratega cognitivo que descompone tareas complejas y sintetiza respuestas finales.
  - **DREAMER**: Subagente cognitivo asíncrono para consolidación de memoria continua, deduplicación y auto-refinado del perfil de usuario (`users.profile_md`).
- **Hybrid In-Memory KV Cache**:
  - Motor de caché clave-valor en RAM pura (`MemoryKVStore`) con expiración por TTL y recolección automática.
  - Conector opcional a Redis (`RedisKVStore`) con protocolo RESP nativo en Go puro si se define `REDIS_URL` o `REDIS_ADDR`.
  - Fallback transparente y silencioso a RAM en caso de desconexión o entorno Zero-Docker.
- **Universal Host Path Discovery & In-Memory Index**:
  - Escaneo BFS de bajo impacto (profundidad 3) de unidades del host (`C:`, `D:`, etc.) omitiendo carpetas pesadas (`node_modules`, `.git`, etc.).
  - Detección autónoma de proyectos mediante marcadores (`.git`, `go.mod`, `package.json`, `Cargo.toml`, etc.).
  - Mapa invertido de tokens en memoria RAM para resolución de rutas y proyectos en microsegundos sin requerir embeddings densos ni Ollama.
  - Inyección dinámica de la topología real de proyectos del host en el System Prompt.
- **Self-Healing Loop**:
  - Intercepción de `stderr` tras `os_run_command`.
  - Detección de dependencias ausentes (ej. `ModuleNotFoundError`) y auto-instalación silenciosa (`pip install <pkg>`).
  - Hasta 3 reintentos autónomos sin intervención del usuario.
- **Message Queuing & Prioritization**:
  - Encolado de mensajes entrantes cuando el agente está ocupado (`[COLA] N mensajes`).
  - Cancelación rápida con `Esc` o `/cancel`.
  - Envío prioritario con interrupción inmediata vía `/now <orden>`.
  - Consulta y limpieza de cola vía `/queue` y `/clearqueue`.
- **User Profile & Continuous Memory (Auto-Learning)**:
  - Ficha de perfil persistente en SQLite (`users.profile_md`), inyectada automáticamente en el System Prompt con caché en RAM.
  - Memoria continua en `user_memories` con motor de búsqueda semántica y FTS5 BM25.
  - Auto-aprendizaje en segundo plano tras cada turno con `FactExtractor.ExtractAndPersistAsync`.
  - Herramientas nativas del agente: `remember_fact`, `search_memory`, `update_user_profile`.
  - Comandos TUI sobrios y sin emojis: `/profile`, `/memories`, `/remember <hecho>`, `/forget <num|id>`, `/forgetall`, `/memmgr`, `/context`, `/dream`, `/paths`, `/scan`.
- **Smart Path Resolver**:
  - Resolución inteligente de rutas de usuario en microsegundos consultando el registro en RAM antes del disco.
  - Protección estricta: nunca resolver rutas del usuario hacia el directorio de trabajo del servidor (`CWD`).
- **Error-Sensitive Dialog Inspection & Anti-Hallucination (`os_detect_dialogs`)**:
  - Detección Win32 nativa de cuadros modales y popups de error (`#32770`, MessageBox, alertas de aplicaciones).
  - Extracción en microsegundos del texto literal de controles hijos (`Static`, `Edit`, `Button`) y clasificación por severidad (`ERROR`, `WARNING`, `INFO`).
  - Directiva estricta CHARC: nunca alucinar o afirmar que una aplicación abrió "sin errores" sin auditar activamente el estado del sistema.
- **Native Pure-Go PDF Engine (`os_create_pdf`, `os_convert_to_pdf`)**:
  - Generación directa y estilización de documentos PDF profesionales (`github.com/jung-kurt/gofpdf`) bajo la política Zero-Docker.
  - Soporte de barras de acento institucional en Neon Lime, metadatos, títulos, párrafos con viñetas y tablas con alternancia de colores.
  - Conversión de fuentes Markdown, TXT, CSV y JSON a PDF estructurado sin recurrir a renombrado de archivos.
- **Native Pure-Go Word Engine (`os_create_docx`)**:
  - Generación directa de documentos Microsoft Word `.docx` válidos en formato OpenXML empaquetado en ZIP en Go puro.
  - Encabezados, estilos, viñetas y tablas estructuradas compatibles con Word, Office 365, LibreOffice y Google Docs.
- **Native Archive Engine (`os_compress_zip`, `os_extract_zip`)**:
  - Empaquetado recursivo y extracción segura con validación anti *Zip Slip* (path traversal) sin herramientas externas.
- **In-File Content Search Engine (`os_search_content`)**:
  - Grep nativo en disco para búsqueda de cadenas literales y expresiones regulares omitiendo directorios masivos (`node_modules`, `.git`, `venv`).
- **Network Port Inspector (`os_port_inspector`)**:
  - Inspección de puertos TCP en escucha, asociación a PID y nombre de proceso, con opción de liberación forzada (`kill`).
- **Native Web Downloader (`os_download_file`)**:
  - Descarga directa y segura de archivos y recursos web hacia carpetas locales de usuario con auto-detección de nombre.
- **Graceful Window & High-Integrity Process Control (`os_close_window`, `os_kill_process`)**:
  - Cierre elegante de ventanas visibles mediante mensajes Win32 `WM_CLOSE` (0x0010) y `SC_CLOSE` al manejador `HWND`.
  - Mapeo multi-idioma de nombres coloquiales a ejecutables reales (`"administrador de tareas"` -> `Taskmgr.exe`, `"bloc de notas"` -> `notepad.exe`, `"calculadora"` -> `calc.exe` / `CalculatorApp.exe`, etc.).
  - Terminación multinivel con fallback a WMI / CIM (`Get-CimInstance Win32_Process | Invoke-CimMethod -MethodName Terminate`) para finalizar procesos elevados con permisos de administrador sin bloqueos de `Acceso denegado`.
  - Directiva estricta anti-alucinación: prohibido confirmar cierres de ventanas sin ejecutar la herramienta correspondiente.
- **Windows Service Inspector & Controller (`os_service_manager`)**:
  - Consulta y administración nativa de servicios de Windows (`list`, `status`, `start`, `stop`, `restart`) con filtrado en tiempo real.
- **Host Docker Container Manager (`os_docker_manager`)**:
  - Control e inspección del daemon de Docker local y contenedores (`status`, `list`, `logs`, `start`, `stop`, `restart`, `stats`).
- **Intelligent Log & Event Analyzer (`os_analyze_logs`)**:
  - Extracción y análisis de trazas, excepciones y pánicos en archivos locales `.log`/`.txt` con filtrado regex y lectura del Visor de Eventos de Windows (`Get-WinEvent`).
- **Instant Shell Launch Engine (`procShellExecuteW`)**:
  - Invocación nativa delegada al Shell del explorador de Windows para abrir ejecutables elevados (`Taskmgr.exe`, `SystemSettings.exe`), herramientas y documentos al primer intento sin errores de elevación 740.
- **Dynamic Window Snapping & Tiling (`os_tile_windows`)**:
  - Acomodo y división de ventanas en pantalla en tiempo real (`left`, `right`, `maximize`, `minimize`, `restore`, `center`, `show_desktop`) mediante APIs Win32 `MoveWindow` y `ShowWindow` calculando el área de trabajo del monitor (`SPI_GETWORKAREA`).
- **Autonomous Disk Purge & Sentinel (`os_disk_cleaner`)**:
  - Análisis y purga segura de archivos temporales huérfanos (`%TEMP%`, cachés de compilación) protegiendo procesos activos, con opción de vaciado de Papelera y telemetría de MB liberados.
- **Audio Output & Master Volume Controller (`os_audio_device`)**:
  - Detección, listado y conmutación de dispositivos de salida de sonido (altavoces, auriculares Bluetooth, FxSound), lectura del volumen maestro en % y estado Mute, ajuste de volumen y silenciamiento nativo vía WASAPI CoreAudio (`IAudioEndpointVolume`).
- **Physical Hardware, USB & Sensor Inspector (`os_hardware_inspector`)**:
  - Detección y clasificación de puertos y dispositivos físicos USB conectados (memorias, hubs, teclados, mouse, Bluetooth, cámaras, discos).
  - Auditoría de periféricos sensibles en uso activo (cámara web y micrófono) consultando Windows `CapabilityAccessManager\ConsentStore` para reportar en tiempo real si transmiten y qué proceso/aplicación los ocupa.
  - Telemetría de hardware profunda: modelo y núcleos de CPU con carga %, memoria RAM total/usada/libre, desglose de almacenamiento por volumen (C:, D:, G:), tarjetas GPU NVIDIA/Intel con VRAM y temperatura en °C (`nvidia-smi`), temperatura térmica ACPI del sistema y estado de batería (CA/descarga).
- **Wi-Fi & Network Inspector (`os_wifi_manager`)**:
  - Telemetría en vivo de interfaces inalámbricas (SSID, señal %, canal, tipo de radio, velocidad Mbps) y escaneo de redes disponibles.
- **Persistent Task Scheduler (`os_schedule_task`)**:
  - Creación, listado y eliminación de tareas programadas persistentes en segundo plano mediante `schtasks.exe` nativo de Windows.

## Development & Execution Commands
- **TUI Interactiva**: `cd backend && go run cmd/ozy/main.go`
- **Compilar Binario**: `cd backend && go build -o ozy.exe ./cmd/ozy/.`
- **Servidor Web**: `cd backend && go run cmd/server/main.go`
- **Tests**:
  - Proveedores: `cd backend && go test ./internal/providers/... -v`
  - Sistema & Paths: `cd backend && go test ./internal/system/... -v`
  - Agente & Tríada: `cd backend && go test ./internal/agent/... -v`
  - TUI & Colas: `cd backend && go test ./internal/tui/... -v`
  - Suite completa: `cd backend && go test ./internal/...`

## Brand Theme Colors
- Primary Brand Color: Vibrant Electric Neon Lime `#d1f107` (`bg-[#d1f107] text-[#181e00] font-bold`).
- Dark Surface Colors: Surface Dim `#131313`, Surface Container `#1e1e1e` / `#222222`.
- Queue Indicator Color: Warm Amber `#f1c40f`.

## Code Principles & Rules
1. **SOLID & DRY**: Código modular, desacoplado y reutilizable.
2. **Robust Error Handling**: Registro claro de errores y recuperación sin caídas del proceso.
3. **Security & Path Safety**: Prevenir path traversal, validar rutas contra carpetas del sistema y ejecutar SQLite en modo solo lectura para consultas exploratorias.
4. **Zero-Docker Constraint**: Mantener la dependencia en librerías Go puras (`modernc.org/sqlite`) sin exigir CGO ni herramientas externas.
