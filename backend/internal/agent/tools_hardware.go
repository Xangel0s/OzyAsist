package agent

import (
	"github.com/ozyassist/backend/internal/providers"
)

var HardwareTools = []providers.ToolDef{
	{
		Name:        "os_port_inspector",
		Description: "Inspecciona qué proceso (PID y ejecutable) tiene ocupado un puerto TCP de red (ej: 3000, 8080, 5432) o lista todos los puertos de red en escucha activa. ÚNICAMENTE para puertos de red/sockets lógicos. NUNCA usar para puertos físicos ni dispositivos USB (para USB o periféricos usa 'os_hardware_inspector'). Permite liberar el puerto inmediatamente si se especifica kill: true.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"port": {
					"type": "integer",
					"description": "Número de puerto TCP a consultar (ej: 3000, 8080). Si se omite, lista todos los puertos en escucha."
				},
				"kill": {
					"type": "boolean",
					"description": "true para terminar forzosamente el proceso que ocupa el puerto y liberarlo inmediatamente"
				}
			}
		}`),
	},
	{
		Name:        "os_service_manager",
		Description: "Inspecciona y administra servicios de Windows (listar servicios, consultar estado detallado, iniciar, detener o reiniciar).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["list", "status", "start", "stop", "restart"],
					"description": "Acción a realizar sobre los servicios (default: 'list')"
				},
				"name": {
					"type": "string",
					"description": "Nombre del servicio de Windows a consultar o controlar (ej: 'wuauserv', 'docker', 'Spooler')"
				},
				"filter": {
					"type": "string",
					"description": "Filtro opcional para buscar servicios por nombre al listar (ej: 'docker', 'sql', 'update')"
				}
			}
		}`),
	},
	{
		Name:        "os_docker_manager",
		Description: "Administra e inspecciona contenedores Docker en el host (ver estado del motor Docker, listar contenedores, leer logs, iniciar, detener, reiniciar o ver consumo de CPU/RAM).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "list", "logs", "start", "stop", "restart", "stats"],
					"description": "Acción a ejecutar con Docker (default: 'status')"
				},
				"container": {
					"type": "string",
					"description": "Nombre o ID del contenedor (requerido para 'logs', 'start', 'stop', 'restart')"
				},
				"tail": {
					"type": "integer",
					"description": "Número de líneas de log a obtener (default: 50)"
				}
			}
		}`),
	},
	{
		Name:        "os_analyze_logs",
		Description: "Analiza y extrae errores, excepciones y trazas críticas de archivos de registro locales o del Visor de Eventos de Windows.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"source": {
					"type": "string",
					"description": "Ruta de un archivo de log local (.log, .txt), o 'application', 'system', 'windows-events' para eventos de Windows"
				},
				"lines": {
					"type": "integer",
					"description": "Cantidad de líneas a inspeccionar (default: 60, máx: 300)"
				},
				"filter": {
					"type": "string",
					"description": "Palabra clave o expresión regular de búsqueda (ej: 'error', 'panic', 'timeout')"
				},
				"severity": {
					"type": "string",
					"enum": ["ERROR", "WARNING", "ALL"],
					"description": "Filtro de severidad predefinido (default: 'ERROR')"
				}
			}
		}`),
	},
	{
		Name:        "os_disk_cleaner",
		Description: "Analiza y purga de forma segura archivos temporales huérfanos (%TEMP%) y vacía opcionalmente la papelera de reciclaje para recuperar espacio en disco.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["analyze", "clean"],
					"description": "'analyze' solo calcula el espacio recuperable; 'clean' purga los archivos seguros (default: 'analyze')"
				},
				"empty_recycle_bin": {
					"type": "boolean",
					"description": "Si es true, también vacía la papelera de reciclaje de Windows"
				}
			}
		}`),
	},
	{
		Name:        "os_wifi_manager",
		Description: "Inspecciona la conexión Wi-Fi actual (nombre SSID, calidad de señal %, velocidad, canal, autenticación) o escanea redes disponibles en el entorno.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["status", "networks"],
					"description": "'status' muestra la conexión actual detallada; 'networks' escanea redes visibles (default: 'status')"
				}
			}
		}`),
	},
	{
		Name:        "os_hardware_inspector",
		Description: "Inspecciona el hardware físico de la máquina: salud integral SMART de discos/temperaturas/alertas ('health'), puertos físicos USB y periféricos conectados ('devices'), detección de cámara web o micrófono en uso activo ('in_use'), y telemetría de CPU, RAM, discos, GPU, temperatura y batería ('telemetry').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["health", "devices", "in_use", "telemetry"],
					"description": "'health' (o 'salud', 'audit') audita la salud SMART de discos, umbrales térmicos y saturación; 'devices' lista puertos físicos USB y periféricos; 'in_use' verifica si la cámara web o micrófono están activos; 'telemetry' reporta CPU, RAM, discos, GPU, temperatura y batería (default: 'health')"
				}
			}
		}`),
	},
	{
		Name:        "os_network_diagnostics",
		Description: "Diagnóstico de red y conectividad: prueba de latencia (ping en ms a 1.1.1.1 o host destino), información de adaptadores e IPs locales/puerta de enlace, y vaciado de caché DNS de Windows (flush DNS).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["test", "flush_dns", "ip_info"],
					"description": "'test' (ping) mide latencia y pérdida de paquetes; 'flush_dns' limpia la caché DNS; 'ip_info' muestra IPs locales y gateway (default: 'test')"
				},
				"host": {
					"type": "string",
					"description": "Host o IP de destino para la prueba de ping (default: '1.1.1.1')"
				}
			}
		}`),
	},
	{
		Name:        "os_process_sentinel",
		Description: "Audita y administra procesos en Windows: 'top_memory' lista los procesos que consumen más memoria RAM; 'top_cpu' lista los de mayor carga de CPU; 'kill' finaliza un proceso rebelde por PID ('process_id') o nombre ('name').",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["top_memory", "top_cpu", "kill"],
					"description": "Acción a realizar: 'top_memory', 'top_cpu' o 'kill' (default: 'top_memory')"
				},
				"top_n": {
					"type": "integer",
					"description": "Cantidad de procesos a mostrar (default: 10, máx: 30)"
				},
				"process_id": {
					"type": "integer",
					"description": "PID del proceso a finalizar (requerido para 'kill' si no se indica 'name')"
				},
				"name": {
					"type": "string",
					"description": "Nombre del ejecutable a finalizar (ej: 'notepad', 'chrome')"
				}
			}
		}`),
	},
}
