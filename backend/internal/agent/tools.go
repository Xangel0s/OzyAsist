package agent

import (
	"encoding/json"

	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/providers"
)

// AgentTools son las herramientas nativas del agente disponibles para el LLM en Modo Code.
// El InputSchema sigue el formato JSON Schema estándar (agnóstico de provider).
// La capa de providers se encarga de traducir al formato de cada API (Anthropic, OpenAI, etc.)
var AgentTools = []providers.ToolDef{
	{
		Name:        "read_file",
		Description: "Lee el contenido completo de un archivo del proyecto. Úsalo para entender el código antes de modificarlo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto (ej: src/main.go)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "write_file",
		Description: "Crea o sobreescribe completamente un archivo del proyecto. Incluye el contenido completo del archivo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto"
				},
				"content": {
					"type": "string",
					"description": "Contenido completo del archivo"
				}
			},
			"required": ["path", "content"]
		}`),
	},
	{
		Name:        "run_command",
		Description: "Ejecuta un comando en la consola de Windows (PowerShell / Git / CLI). Si el comando es sobre otro proyecto (ej: 'crmgeofal', 'cotizador'), especifica el nombre o ruta en 'cwd'.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Comando a ejecutar (ej: 'git log -n 5 --oneline', 'git status', 'dir')"
				},
				"cwd": {
					"type": "string",
					"description": "Directorio de trabajo o nombre del proyecto donde ejecutar el comando (ej: 'crmgeofal', 'C:\\Users\\User\\Documents\\crmgeofal')"
				}
			},
			"required": ["command"]
		}`),
	},
	{
		Name:        "list_files",
		Description: "Lista archivos del proyecto con un patrón glob. Útil para explorar la estructura del proyecto.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "Patrón glob (ej: **/*.go, src/**/*.ts, *.json)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de resultados a devolver (default: 50)"
				}
			},
			"required": ["pattern"]
		}`),
	},
	{
		Name:        "search_text",
		Description: "Busca texto o expresiones regulares dentro de los archivos del proyecto (similar a ripgrep). Devuelve archivo, línea y contenido.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Texto o regex a buscar"
				},
				"include": {
					"type": "string",
					"description": "Patrón glob para filtrar archivos (ej: *.go, *.ts)"
				},
				"case_sensitive": {
					"type": "boolean",
					"description": "Si la búsqueda es sensible a mayúsculas (default: false)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de coincidencias (default: 30)"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "apply_diff",
		Description: "Aplica un diff unificado (formato unified diff) a un archivo. Úsalo para modificaciones quirúrgicas sin reescribir el archivo completo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo a parchear"
				},
				"diff": {
					"type": "string",
					"description": "Diff en formato unified diff estándar (--- a/ +++ b/ @@ ...)"
				}
			},
			"required": ["path", "diff"]
		}`),
	},
	{
		Name:        "web_search",
		Description: "Busca en la web información técnica actualizada, paquetes, documentación, noticias o soluciones a errores utilizando fuentes confiables de internet.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Término de búsqueda en internet"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "deep_search",
		Description: "Realiza una investigación profunda en internet dividiendo la consulta en sub-búsquedas, extrayendo fuentes confiables y generando síntesis fundamentada con citas.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Pregunta o tema complejo para investigación exhaustiva en internet"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "web_fetch",
		Description: "Descarga e inspecciona directamente el contenido de cualquier página web o URL en vivo (HTTP/HTTPS). Extrae título, metadatos, encabezados, OpenGraph y texto Markdown legible. Úsalo SIEMPRE que el usuario mencione un dominio (ej: peruflack.com), sitio web, artículo o para comprobar si un sitio está activo y qué contiene.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "URL o nombre de dominio a inspeccionar (ej: 'https://peruflack.com' o 'peruflack.com')"
				},
				"max_length": {
					"type": "integer",
					"description": "Límite opcional de caracteres de contenido a extraer (default: 4000)"
				}
			},
			"required": ["url"]
		}`),
	},
	{
		Name:        "web_dns_lookup",
		Description: "Consulta registros DNS (A, AAAA, CNAME, MX, TXT) y resuelve las direcciones IP de cualquier dominio en internet para verificar si existe, a qué servidor apunta (Vercel, AWS, Cloudflare, etc.) y si está activo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"domain": {
					"type": "string",
					"description": "Nombre de dominio a consultar (ej: 'peruflack.com')"
				}
			},
			"required": ["domain"]
		}`),
	},
	{
		Name:        "os_get_desktop",
		Description: "Obtiene la lista exacta y completa de todos los archivos, accesos directos e iconos del escritorio de Windows de forma nativa (<1ms). Úsalo SIEMPRE para saber qué hay en el escritorio.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {}
		}`),
	},
	{
		Name:        "os_list_apps",
		Description: "Consulta directamente el registro de Windows para buscar y listar aplicaciones/software instalados (ej: 'League of Legends', 'VS Code', 'Chrome', 'Docker').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"filter": {
					"type": "string",
					"description": "Filtro o nombre de la aplicación a buscar (dejar vacío para listar todas)"
				}
			}
		}`),
	},
	{
		Name:        "os_explore",
		Description: "Navega y explora carpetas del sistema de archivos de forma nativa retornando metadatos estructurados (archivos, tamaños, fechas, subdirectorios).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta absoluta o relativa a explorar (ej: C:\\Users\\User\\Documents o .)"
				},
				"depth": {
					"type": "integer",
					"description": "Profundidad máxima de recursión (default: 1)"
				}
			}
		}`),
	},
	{
		Name:        "os_find_files",
		Description: "Busca archivos de forma nativa e instantánea por nombre o patrón en un directorio del sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"root": {
					"type": "string",
					"description": "Directorio raíz donde buscar (ej: C:\\Users\\User o .)"
				},
				"pattern": {
					"type": "string",
					"description": "Nombre de archivo o patrón a buscar (ej: *.png, report, ozymetas)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo de resultados (default: 30)"
				}
			}
		}`),
	},
	{
		Name:        "os_active_windows",
		Description: "Enumera las ventanas visibles y aplicaciones abiertas actualmente en la pantalla del usuario.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {}
		}`),
	},
	{
		Name:        "os_system_info",
		Description: "Obtiene métricas nativas de hardware y telemetría en tiempo real (uso de CPU, RAM libre/usada, espacio en disco).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {}
		}`),
	},
	{
		Name:        "os_create_dir",
		Description: "Crea una carpeta o estructura de directorios en el sistema de archivos de forma nativa e instantánea.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta de la carpeta a crear (ej: C:\\Users\\User\\Documents\\MiProyecto o ./nueva_carpeta)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_move_item",
		Description: "Mueve o renombra un archivo o carpeta a una nueva ubicación en el sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src": {
					"type": "string",
					"description": "Ruta de origen del archivo o carpeta"
				},
				"dst": {
					"type": "string",
					"description": "Ruta de destino del archivo o carpeta"
				}
			},
			"required": ["src", "dst"]
		}`),
	},
	{
		Name:        "os_copy_item",
		Description: "Copia un archivo o carpeta a una nueva ubicación en el sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src": {
					"type": "string",
					"description": "Ruta de origen del archivo o carpeta"
				},
				"dst": {
					"type": "string",
					"description": "Ruta de destino"
				}
			},
			"required": ["src", "dst"]
		}`),
	},
	{
		Name:        "os_delete_item",
		Description: "Elimina un archivo o carpeta. Por defecto lo envía de forma segura a la Papelera de reciclaje de Windows.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta del archivo o carpeta a eliminar"
				},
				"permanent": {
					"type": "boolean",
					"description": "true para eliminar permanentemente, false para enviar a la Papelera (default: false)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_organize_folder",
		Description: "Organiza y clasifica automáticamente todos los archivos de una carpeta (como Escritorio o Descargas) agrupándolos en subcarpetas temáticas (Documentos, Imágenes, Videos, Música, Instaladores, Código, etc.).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta de la carpeta a organizar (ej: C:\\Users\\User\\Desktop o C:\\Users\\User\\Downloads)"
				}
			},
			"required": ["path"]
		}`),
	},

	{
		Name:        "os_focus_window",
		Description: "Trae al frente y enfoca una ventana visible de Windows por su Handle (HWND).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"hwnd": {
					"type": "integer",
					"description": "Handle numérico de la ventana obtenido con os_active_windows"
				}
			},
			"required": ["hwnd"]
		}`),
	},
	{
		Name:        "os_kill_process",
		Description: "Cierra o termina un proceso o aplicación en ejecución por su PID o por su nombre (ej: 'notepad', 'notepad.exe', 'calc').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"pid": {
					"type": "integer",
					"description": "Process ID (PID) del proceso a terminar"
				},
				"name": {
					"type": "string",
					"description": "Nombre de la aplicación o ejecutable a terminar (ej: 'notepad.exe', 'calc.exe', 'chrome.exe')"
				},
				"force": {
					"type": "boolean",
					"description": "Forzar terminación inmediata (default: true)"
				}
			}
		}`),
	},
	{
		Name:        "os_hardware_control",
		Description: "Ajusta configuraciones físicas del sistema operativo (brillo o volumen).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"setting": {
					"type": "string",
					"description": "El ajuste a modificar: 'brightness' o 'volume'"
				},
				"value": {
					"type": "integer",
					"description": "El valor del 0 al 100"
				}
			},
			"required": ["setting", "value"]
		}`),
	},
	{
		Name:        "os_power_state",
		Description: "Cambia el estado de energía de la computadora.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"state": {
					"type": "string",
					"description": "El estado deseado: 'shutdown', 'restart', o 'sleep'"
				}
			},
			"required": ["state"]
		}`),
	},
	{
		Name:        "os_launch_app",
		Description: "Lanza o abre una aplicación o programa en Windows. Permite abrir cualquier programa instalado (ej: 'antigravity', 'code', 'notepad', 'calc', 'chrome', 'explorer', 'brave', 'docker') y opcionalmente pasar una ruta de archivo, carpeta o proyecto que la aplicación debe abrir.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"appName": {
					"type": "string",
					"description": "Nombre de la aplicación a ejecutar (ej: 'antigravity', 'code', 'notepad', 'calc', 'chrome', 'explorer')"
				},
				"path": {
					"type": "string",
					"description": "Ruta de archivo, carpeta o proyecto a abrir con la aplicación (ej: 'crmgeofal', 'C:\\Users\\User\\Documents\\crmgeofal', 'test.txt')"
				},
				"args": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Argumentos adicionales de línea de comandos para la aplicación"
				}
			},
			"required": ["appName"]
		}`),
	},
	{
		Name:        "os_run_command",
		Description: "Ejecuta un comando en la consola de Windows (PowerShell / Git / CLI del sistema) y devuelve la salida (stdout, stderr y código de salida). Úsalo para inspeccionar repositorios Git ('git log', 'git status', 'git diff'), ejecutar scripts, compilaciones y diagnósticos. IMPORTANTE: Si la consulta es sobre otro proyecto (ej: 'crmgeofal', 'cotizador'), pasa su nombre o ruta en 'cwd' para no ejecutarlo erróneamente en el directorio de OzyAssist.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Comando de consola a ejecutar (ej: 'git log -n 5 --oneline', 'git status', 'dir', 'whoami')"
				},
				"cwd": {
					"type": "string",
					"description": "Directorio de trabajo o nombre del proyecto donde ejecutar el comando (ej: 'crmgeofal', 'C:\\Users\\User\\Documents\\crmgeofal'). Ozy resuelve automáticamente proyectos en Documents."
				}
			},
			"required": ["command"]
		}`),
	},
	{
		Name:        "os_draft_email",
		Description: "Redacta un correo electrónico formal o informal y abre automáticamente la ventana del cliente de correo (Outlook, Thunderbird, Windows Mail, o Gmail/Outlook Web en el navegador) con el destinatario, asunto y cuerpo pre-cargados para revisión y envío.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"to": {
					"type": "string",
					"description": "Dirección de correo del destinatario principal (ej: 'cliente@empresa.com')"
				},
				"subject": {
					"type": "string",
					"description": "Asunto del correo electrónico"
				},
				"body": {
					"type": "string",
					"description": "Cuerpo o contenido completo del correo electrónico redactado"
				},
				"cc": {
					"type": "string",
					"description": "Dirección o direcciones con copia (CC) opcional"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "desktop_app", "gmail", "outlook_web"],
					"description": "Cliente a usar: 'auto' (cliente predeterminado del SO), 'gmail' (pestaña de redacción en Gmail Web), o 'outlook_web'"
				},
				"from_account": {
					"type": "string",
					"description": "Correo o nombre del perfil del navegador (ej: 'zastuto5@gmail.com', 'Tu Chrome', 'Default') para usar esa sesión autenticada directamente sin pedir contraseña"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la ventana de composición inmediatamente en pantalla (default: true)"
				}
			},
			"required": ["to", "subject", "body"]
		}`),
	},
	{
		Name:        "os_draft_whatsapp",
		Description: "Redacta un mensaje para WhatsApp y abre la conversación (en WhatsApp Web o en la app de escritorio de Windows) con el número del destinatario y el texto pre-cargado listo para enviar con 1 clic.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"phone": {
					"type": "string",
					"description": "Número de teléfono con código de país (ej: '+51999888777' o '51999888777')"
				},
				"text": {
					"type": "string",
					"description": "Mensaje completo a enviar"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "web", "desktop"],
					"description": "Cliente: 'auto' (wa.me universal), 'web' (WhatsApp Web), 'desktop' (app de Windows)"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la ventana inmediatamente (default: true)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "os_draft_telegram",
		Description: "Redacta un mensaje para Telegram y abre el chat (Telegram Desktop o Telegram Web) con el usuario o canal y el texto pre-cargado listo para enviar con 1 clic.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"recipient": {
					"type": "string",
					"description": "Nombre de usuario (@usuario) o número telefónico del destinatario"
				},
				"text": {
					"type": "string",
					"description": "Mensaje completo a enviar"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "desktop", "web"],
					"description": "Cliente: 'auto' (t.me universal), 'desktop' (app de Windows tg://), 'web' (Telegram Web)"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la app inmediatamente (default: true)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "telegram_send_message",
		Description: "Envía un mensaje de Telegram 100% en segundo plano sin abrir ninguna ventana de navegador o app, utilizando la API oficial de Telegram Bot (TELEGRAM_BOT_TOKEN y TELEGRAM_CHAT_ID).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Texto del mensaje a enviar vía Telegram Bot API"
				},
				"chat_id": {
					"type": "string",
					"description": "ID de chat opcional (si se omite, usa el del archivo .env)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "browser_list_profiles",
		Description: "Detecta e inspecciona todos los perfiles de navegador instalados en el sistema (Google Chrome, Microsoft Edge, Brave) junto con sus cuentas asociadas (correos de Google/Microsoft, nombres y directorios de perfil) para abrir sesiones y correos autenticados sin pedir contraseñas.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_get_clipboard",
		Description: "Obtiene y lee el texto que el usuario tiene copiado en el portapapeles de Windows en este momento.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_set_clipboard",
		Description: "Copia texto o contenido al portapapeles de Windows para que el usuario pueda pegarlo inmediatamente con Ctrl+V.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Texto a copiar al portapapeles"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "os_notify",
		Description: "Envía una notificación emergente (Toast banner) nativa de Windows en la esquina de la pantalla con título y mensaje.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "Título de la notificación (ej: 'OzyAssist', 'Tarea Completada')"
				},
				"message": {
					"type": "string",
					"description": "Mensaje o cuerpo de la notificación"
				}
			},
			"required": ["message"]
		}`),
	},
	{
		Name:        "os_read_document",
		Description: "Lee y extrae el texto completo de documentos locales en formatos PDF (.pdf), Excel (.xlsx, .xlsm), Word (.docx), CSV (.csv) y texto plano (.txt, .md, .json, .log). Úsalo para inspeccionar contratos, hojas de cálculo, reportes o cotizaciones.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta absoluta o relativa al documento a leer"
				},
				"max_length": {
					"type": "integer",
					"description": "Máximo de caracteres a extraer (default: 8000)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_schedule_alarm",
		Description: "Programa un recordatorio o alarma con notificación en Windows y aviso sonoro/por voz para dentro de un tiempo ('10m', '30s', '1h') o a una hora fija ('15:30').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"time_in": {
					"type": "string",
					"description": "Tiempo para la alarma: relativo ('10m', '30s', '1h', '45m') o específico ('15:30')"
				},
				"message": {
					"type": "string",
					"description": "Mensaje o asunto del recordatorio"
				}
			},
			"required": ["time_in", "message"]
		}`),
	},
	{
		Name:        "os_list_alarms",
		Description: "Consulta y lista todos los recordatorios y alarmas pendientes actualmente activas en el sistema.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "browser_open_groq",
		Description: "Abre la consola de API Keys de Groq (https://console.groq.com/keys) en Google Chrome con el perfil autenticado del usuario para obtener la clave con 1 solo clic.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_setup_groq_key",
		Description: "Valida y guarda automáticamente una API Key de Groq (gsk_...) leída del portapapeles o parámetro en el archivo .env, activando el motor de voz y Hey Ozy inmediatamente sin reiniciar.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"api_key": {
					"type": "string",
					"description": "Clave opcional de Groq (si se omite, Ozy la detectará automáticamente de lo que tengas copiado en el portapapeles)"
				}
			}
		}`),
	},
	{
		Name:        "os_take_screenshot",
		Description: "Captura una imagen PNG del monitor principal de Windows en tiempo real para ver qué hay en pantalla.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {}
		}`),
	},
	{
		Name:        "os_mouse_click",
		Description: "Mueve el cursor y hace clic de mouse nativo en coordenadas específicas (X, Y) de la pantalla del usuario.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"x": {
					"type": "integer",
					"description": "Coordenada horizontal X en píxeles"
				},
				"y": {
					"type": "integer",
					"description": "Coordenada vertical Y en píxeles"
				},
				"button": {
					"type": "string",
					"enum": ["left", "right", "double"],
					"description": "Tipo de clic: 'left' (clic izquierdo), 'right' (clic derecho) o 'double' (doble clic)"
				}
			},
			"required": ["x", "y"]
		}`),
	},
	{
		Name:        "os_type_text",
		Description: "Escribe texto o pulsa teclas de forma simulada en la ventana activa o campo enfocado actualmente.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Texto a escribir o teclas"
				},
				"press_enter": {
					"type": "boolean",
					"description": "Si es true, presiona la tecla Enter al finalizar de escribir (default: false)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "os_create_excel",
		Description: "Crea o genera un archivo de hoja de cálculo nativo de Excel (.xlsx) con tablas, múltiples hojas y cabeceras estilizadas en verde neón (#D1F107). Úsalo para reportes, consolidados, presupuestos o exportación de datos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta destino del archivo Excel (ej: 'Desktop/reporte_ventas.xlsx', 'crm_data.xlsx')"
				},
				"title": {
					"type": "string",
					"description": "Título de la hoja principal (ej: 'Ventas 2026', 'Inventario')"
				},
				"headers": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Lista de nombres de columnas (ej: ['ID', 'Cliente', 'Total', 'Fecha'])"
				},
				"rows": {
					"type": "array",
					"items": {
						"type": "array",
						"items": {"type": "string"}
					},
					"description": "Matriz bidimensional con los valores de cada fila"
				}
			},
			"required": ["path", "headers", "rows"]
		}`),
	},
	{
		Name:        "os_query_db",
		Description: "Ejecuta consultas SQL de solo lectura (SELECT, PRAGMA) en bases de datos SQLite locales (.db, .sqlite). Bloquea sentencias destructivas. Úsalo para auditar datos, clientes, tareas o métricas de proyectos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"db_path": {
					"type": "string",
					"description": "Ruta a la base de datos (ej: 'backend/data/ozyassist.db', 'crmgeofal/app.db')"
				},
				"query": {
					"type": "string",
					"description": "Consulta SQL SELECT o PRAGMA a ejecutar"
				},
				"max_rows": {
					"type": "integer",
					"description": "Máximo de filas a devolver (default: 50)"
				}
			},
			"required": ["db_path", "query"]
		}`),
	},
	{
		Name:        "os_analyze_screen",
		Description: "Captura la pantalla completa de Windows y realiza un análisis visual inteligente con IA multimodal. Úsalo para diagnosticar errores visuales, ventanas emergentes, gráficas o interfaces.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"question": {
					"type": "string",
					"description": "Pregunta o instrucción de análisis sobre lo que se ve en la pantalla (ej: '¿qué error se muestra en pantalla?', 'describe las gráficas abiertas')"
				}
			}
		}`),
	},
	{
		Name:        "os_watchdog",
		Description: "Inicia, detiene o lista monitores proactivos en segundo plano (Watchdog) para vigilar puertos locales (ej: '8080', '3000') o procesos. Emite notificaciones Toast automáticas si el servicio cae o se recupera.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"description": "Acción a realizar: 'start', 'stop', 'list'"
				},
				"type": {
					"type": "string",
					"description": "Tipo de objetivo para 'start': 'port' (puerto de red) o 'process' (proceso de Windows)"
				},
				"target": {
					"type": "string",
					"description": "Objetivo a vigilar (ej: '8080', '3000', 'docker.exe', 'backend.exe')"
				},
				"interval": {
					"type": "integer",
					"description": "Intervalo de comprobación en segundos (default: 10)"
				},
				"id": {
					"type": "string",
					"description": "ID del monitor a detener (para acción 'stop')"
				},
				"alert_msg": {
					"type": "string",
					"description": "Mensaje personalizado de alerta al detectar caída"
				}
			},
			"required": ["action"]
		}`),
	},
}

// VoiceAgentTools es un subconjunto m\u00ednimo de herramientas para el Modo Voz.
// Solo incluye tools OS nativas, sin tools de c\u00f3digo (read_file, write_file, etc.)
// Esto reduce el payload del schema de ~3000 tokens a ~700 tokens, acelerando
// el tiempo de primera respuesta en modelos locales peque\u00f1os.
var VoiceAgentTools = []providers.ToolDef{
	{
		Name:        "os_get_desktop",
		Description: "Lista los archivos e iconos del escritorio de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_list_apps",
		Description: "Busca aplicaciones instaladas en Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"filter":{"type":"string","description":"Nombre a buscar"}}}`),
	},
	{
		Name:        "os_active_windows",
		Description: "Lista las ventanas abiertas actualmente.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},

	{
		Name:        "os_system_info",
		Description: "Telemetr\u00eda de hardware en tiempo real: CPU, RAM, disco.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_explore",
		Description: "Explora el contenido de una carpeta.",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta a explorar"}},"required":["path"]}`),
	},
	{
		Name:        "os_organize_folder",
		Description: "Organiza autom\u00e1ticamente los archivos de una carpeta.",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta de la carpeta a organizar"}},"required":["path"]}`),
	},
	{
		Name:        "web_search",
		Description: "Busca informaci\u00f3n en internet.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string","description":"T\u00e9rmino de b\u00fasqueda"}},"required":["query"]}`),
	},
	{
		Name:        "os_hardware_control",
		Description: "Ajusta el brillo o volumen del PC (ej: 'volume' 50, 'brightness' 100).",
		InputSchema: mustJSON(`{"type":"object","properties":{"setting":{"type":"string"},"value":{"type":"integer"}},"required":["setting","value"]}`),
	},
	{
		Name:        "os_power_state",
		Description: "Apaga, reinicia o suspende la PC ('shutdown', 'restart', 'sleep').",
		InputSchema: mustJSON(`{"type":"object","properties":{"state":{"type":"string"}},"required":["state"]}`),
	},
	{
		Name:        "os_launch_app",
		Description: "Lanza o abre una aplicación en la PC (soporta 'antigravity', 'code', 'calc', 'notepad', etc.) y opcionalmente una ruta de carpeta o proyecto.",
		InputSchema: mustJSON(`{"type":"object","properties":{"appName":{"type":"string"},"path":{"type":"string"}},"required":["appName"]}`),
	},
	{
		Name:        "os_run_command",
		Description: "Ejecuta un comando en consola de Windows (PowerShell/Git/CLI). Úsalo para 'git log', 'git status', 'dir', etc.",
		InputSchema: mustJSON(`{"type":"object","properties":{"command":{"type":"string","description":"Comando a ejecutar"}},"required":["command"]}`),
	},
	{
		Name:        "deep_search",
		Description: "Investigación profunda en internet con citas grounded y reporte completo.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string","description":"Tema a investigar"}},"required":["query"]}`),
	},
	{
		Name:        "os_draft_email",
		Description: "Redacta un correo electrónico formal y abre la ventana del cliente de correo en pantalla para revisar y enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"to":{"type":"string","description":"Destinatario"},"subject":{"type":"string","description":"Asunto"},"body":{"type":"string","description":"Cuerpo del correo"}},"required":["to","subject","body"]}`),
	},
	{
		Name:        "os_draft_whatsapp",
		Description: "Redacta un mensaje de WhatsApp y abre la conversación con el mensaje listo para enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"phone":{"type":"string","description":"Teléfono"},"text":{"type":"string","description":"Mensaje"}},"required":["text"]}`),
	},
	{
		Name:        "os_draft_telegram",
		Description: "Redacta un mensaje de Telegram y abre el chat listo para enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"recipient":{"type":"string","description":"Usuario o número"},"text":{"type":"string","description":"Mensaje"}},"required":["text"]}`),
	},
	{
		Name:        "os_get_clipboard",
		Description: "Lee el contenido actual del portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_set_clipboard",
		Description: "Copia texto al portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	},
	{
		Name:        "os_notify",
		Description: "Muestra una notificación emergente en la pantalla de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`),
	},
	{
		Name:        "os_read_document",
		Description: "Lee y extrae texto de un documento local (PDF, Excel, Word, CSV, TXT).",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta al archivo"}},"required":["path"]}`),
	},
	{
		Name:        "os_schedule_alarm",
		Description: "Programa una alarma o recordatorio ('10m', '30s', '1h', '15:30').",
		InputSchema: mustJSON(`{"type":"object","properties":{"time_in":{"type":"string"},"message":{"type":"string"}},"required":["time_in","message"]}`),
	},
	{
		Name:        "os_list_alarms",
		Description: "Lista los recordatorios activos pendientes.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_kill_process",
		Description: "Cierra o termina un programa o proceso por nombre (ej: 'notepad', 'calc') o por PID.",
		InputSchema: mustJSON(`{"type":"object","properties":{"name":{"type":"string"},"pid":{"type":"integer"}}}`),
	},
}

// GetActiveTools returns the tools to inject into the LLM, including dynamic MCP tools.
// En voiceMode devuelve únicamente VoiceAgentTools para garantizar baja latencia y prefill instantáneo.
func GetActiveTools(voiceMode bool) []providers.ToolDef {
	if voiceMode {
		return VoiceAgentTools
	}

	tools := make([]providers.ToolDef, len(AgentTools))
	copy(tools, AgentTools)

	// Dynamic MCP tools (solo en modo texto / consola para proteger latencia y tokens en voz)
	mcpTools := mcp.DefaultRegistry.GetAllTools()
	for name, mcpTool := range mcpTools {
		// Convert MCP input schema to JSON RawMessage
		schemaBytes, _ := json.Marshal(mcpTool.InputSchema)

		tools = append(tools, providers.ToolDef{
			Name:        name,
			Description: mcpTool.Description + " [MCP]",
			InputSchema: json.RawMessage(schemaBytes),
		})
	}

	return tools
}

func mustJSON(s string) json.RawMessage {
	// Comprimir el JSON eliminando espacios innecesarios
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic("tools.go: invalid JSON schema: " + err.Error())
	}
	raw, _ := json.Marshal(v)
	return json.RawMessage(raw)
}
