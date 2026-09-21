package agent

import (
	"github.com/ozyassist/backend/internal/providers"
)

var DesktopTools = []providers.ToolDef{
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
		Name:        "os_close_window",
		Description: "Cierra elegantemente una ventana visible enviando el mensaje Win32 WM_CLOSE a su manejador (HWND) o cerrando el proceso correspondiente por título o PID. Úsala siempre que el usuario pida cerrar un programa o ventana visible (ej: 'cierra el administrador de tareas', 'cierra la calculadora', etc.).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"hwnd": {
					"type": "integer",
					"description": "Handle numérico de la ventana obtenido con os_active_windows"
				},
				"title": {
					"type": "string",
					"description": "Título o nombre de la ventana a cerrar (ej: 'Administrador de tareas', 'Calculadora', 'Bloc de notas')"
				},
				"pid": {
					"type": "integer",
					"description": "Process ID (PID) asociado a la ventana"
				}
			}
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
	{
		Name:        "os_detect_dialogs",
		Description: "Inspecciona cuadros de diálogo modales, popups del sistema (#32770) y ventanas de mensajes o errores en pantalla. Extrae el texto literal del error y los botones interactivos. OBLIGATORIO: Úsalo siempre después de abrir un archivo o lanzar una aplicación para verificar si Windows o la aplicación mostró algún error.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"app_filter": {
					"type": "string",
					"description": "Filtro opcional por título de ventana o nombre de proceso (ej: 'acrobat', 'adobe', 'excel', 'word', 'error'). Si está vacío, devuelve todos los diálogos activos."
				}
			}
		}`),
	},
	{
		Name:        "os_tile_windows",
		Description: "Acomoda, divide o reposiciona ventanas en la pantalla (mitad izquierda, mitad derecha, maximizar, restaurar, centrar o minimizar todo para ver escritorio).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "Nombre o título de la ventana a mover (ej: 'Administrador de tareas', 'Google Chrome', 'Visual Studio Code')"
				},
				"hwnd": {
					"type": "integer",
					"description": "Handle numérico de la ventana (opcional si se especifica title)"
				},
				"layout": {
					"type": "string",
					"enum": ["left", "right", "maximize", "minimize", "restore", "center", "show_desktop"],
					"description": "Disposición deseada de la ventana (default: 'maximize')"
				}
			}
		}`),
	},
	{
		Name:        "os_audio_device",
		Description: "Controla el sistema de audio de Windows: consulta el nivel de volumen master actual (0-100%) y estado Mute, ajusta el nivel de volumen, silencia/reactiva sonido, lista dispositivos de salida o cambia el dispositivo de salida predeterminado.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["list", "set", "get_volume", "set_volume", "mute", "unmute"],
					"description": "Acción a realizar: 'get_volume' para ver el volumen actual y si está silenciado; 'set_volume' para ajustar nivel; 'mute' o 'unmute' para silenciar/reactivar; 'list' para ver endpoints; 'set' para cambiar endpoint (default: 'list')"
				},
				"level": {
					"type": "number",
					"description": "Porcentaje de volumen maestro a establecer (0 a 100, requerido para set_volume)"
				},
				"mute": {
					"type": "boolean",
					"description": "true para silenciar el sonido, false para reactivarlo (para action: 'mute')"
				},
				"name": {
					"type": "string",
					"description": "Nombre o fragmento del dispositivo a seleccionar (requerido para 'set', ej: 'auriculares', 'altavoces')"
				}
			}
		}`),
	},
	{
		Name:        "os_schedule_task",
		Description: "Crea, lista o elimina tareas programadas persistentes en Windows (schtasks.exe) que se ejecutan automáticamente en segundo plano.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["list", "create", "delete"],
					"description": "Acción a ejecutar: 'list', 'create' o 'delete' (default: 'list')"
				},
				"name": {
					"type": "string",
					"description": "Nombre de la tarea (ej: 'BackupProyectos', 'CleanLogs')"
				},
				"command": {
					"type": "string",
					"description": "Comando o script completo a ejecutar (requerido para 'create')"
				},
				"schedule": {
					"type": "string",
					"enum": ["DAILY", "HOURLY", "ONLOGON", "WEEKLY"],
					"description": "Frecuencia de ejecución (default: 'DAILY')"
				},
				"time": {
					"type": "string",
					"description": "Hora de ejecución en formato 24h 'HH:mm' (ej: '20:00', default: '09:00')"
				}
			}
		}`),
	},
	{
		Name:        "os_power_profile",
		Description: "Consulta o cambia el plan de energía de Windows (Equilibrado, Alto Rendimiento, Economizador) y ajusta el nivel de brillo de la pantalla en % (0 a 100).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "set_plan", "set_brightness"],
					"description": "'status' consulta plan activo y brillo; 'set_plan' cambia el plan; 'set_brightness' cambia el brillo (default: 'status')"
				},
				"plan": {
					"type": "string",
					"enum": ["balanced", "high_performance", "power_saver"],
					"description": "Nombre del plan a activar ('balanced', 'high_performance', 'power_saver')"
				},
				"brightness": {
					"type": "integer",
					"description": "Porcentaje de brillo de pantalla a establecer (0 a 100)"
				}
			}
		}`),
	},
	{
		Name:        "os_toast_notify",
		Description: "Envía una notificación interactiva nativa Toast en el Centro de Notificaciones de Windows 10/11 (tarjeta emergente en la esquina inferior derecha) para alertar al usuario.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "Título de la notificación (ej: 'OzyAssist: Alerta de Hardware')"
				},
				"message": {
					"type": "string",
					"description": "Texto del mensaje o alerta"
				}
			},
			"required": ["message"]
		}`),
	},
	{
		Name:        "os_keyboard_layout",
		Description: "Inspecciona la distribución de teclado activa ('status'), lista las distribuciones instaladas ('list') o cambia la distribución de teclado de Windows al instante ('set' con 'latam', 'spain', 'us' o código KLID hex ej. '0000080A').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "list", "set"],
					"description": "Acción a realizar: 'status' (ver activo), 'list' (ver disponibles) o 'set' (cambiar teclado)"
				},
				"layout": {
					"type": "string",
					"description": "Distribución deseada para 'set' (ej: 'latam', 'spain', 'us', o KLID hex ej: '0000080A')"
				}
			}
		}`),
	},
	{
		Name:        "os_startup_manager",
		Description: "Administra las aplicaciones que inician automáticamente al arrancar sesión en Windows. 'list' muestra programas en HKCU\\Run, HKLM\\Run y carpeta Startup; 'add' registra una aplicación en el inicio; 'remove' la retira.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["list", "add", "remove"],
					"description": "Acción a realizar: 'list', 'add' o 'remove' (default: 'list')"
				},
				"name": {
					"type": "string",
					"description": "Nombre de la aplicación o entrada de inicio"
				},
				"command": {
					"type": "string",
					"description": "Ruta o comando ejecutable a iniciar con Windows (requerido para 'add')"
				}
			}
		}`),
	},
	{
		Name:        "os_notification_focus",
		Description: "Consulta o configura el estado de las notificaciones emergentes de Windows y el modo silencioso/no molestar ('status' consulta el estado; 'set' con enabled: true/false las activa o silencia).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "set"],
					"description": "Acción: 'status' para ver el estado, 'set' para cambiarlo"
				},
				"enabled": {
					"type": "boolean",
					"description": "true para activar notificaciones normalmente, false para silenciarlas (modo no molestar)"
				}
			}
		}`),
	},
	{
		Name:        "os_get_clipboard",
		Description: "Lee y obtiene el texto actualmente copiado en el portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_set_clipboard",
		Description: "Copia o escribe una cadena de texto directamente en el portapapeles de Windows.",
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
		Name:        "os_media_control",
		Description: "Controla la reproducción multimedia global de Windows (Spotify, YouTube, VLC, navegador): pausar/reanudar ('play_pause'), siguiente pista ('next'), pista anterior ('previous') o detener ('stop').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["play_pause", "next", "previous", "stop"],
					"description": "Comando multimedia a ejecutar (default: 'play_pause')"
				}
			}
		}`),
	},
	{
		Name:        "os_display_config",
		Description: "Consulta o conmuta la configuración de pantallas y monitores externos en Windows: 'status' lista resoluciones y monitores conectados; 'extend' amplía el escritorio; 'clone' duplica la pantalla; 'internal' solo pantalla de la PC; 'external' solo segunda pantalla.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "extend", "clone", "internal", "external"],
					"description": "Modo de pantalla a configurar (default: 'status')"
				}
			}
		}`),
	},
}
