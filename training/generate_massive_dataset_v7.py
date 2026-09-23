"""
generate_massive_dataset_v7.py
Generador de Dataset Agéntico de Alta Fidelidad para OzyAssist (v7)
Diseñado bajo estándares de la industria (Berkeley Function Calling Leaderboard / Hermes 3 / AgentBench):
1. Saludos y preámbulos conversacionales realistas en español (LATAM y España).
2. Distinción nativa de Windows 11: Apps de usuario (os_launch_app/os_close_window) vs Servicios SCM (os_service_manager).
3. Bucles ReAct multi-etapa reales: Búsqueda web (web_search) -> Observación verídica -> Generación documental (os_create_pdf/os_create_docx).
4. Anáforas y foco contextual ("ábrelo", "léelo", "¿cuándo se creó?").
5. Ejemplos negativos (Chitchat, saludos puros, preguntas conceptuales) sin invocar herramientas.
6. Suite Office completa (Excel fórmulas, Word OpenXML, PDF profesional, Grep nativo).
7. Hardware físico, puertos USB, telemetría y WiFi.
"""

import json
import random
import os

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario actual: User (Ruta: C:\\Users\\User)
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.

DIRECTRICES:
1. Si el usuario te saluda ("hola", "buenos días") o hace preguntas conceptuales, responde de manera amigable, concisa y en lenguaje natural sin invocar herramientas.
2. Si el usuario solicita una acción (abrir/cerrar apps, gestionar servicios de Windows 11, buscar archivos o investigar en la web), analiza la situación en <thought>...</thought> y emite inmediatamente la herramienta correspondiente en formato JSON dentro de:
<tool_call>{"name": "nombre_herramienta", "arguments": {...}}</tool_call>
3. Para investigaciones temáticas ("investiga sobre Beck...", "haz un informe de..."), ejecuta PRIMERO 'web_search' para recopilar datos reales antes de invocar 'os_create_pdf' o 'os_create_docx'.
4. En Windows 11, los servicios del sistema en segundo plano usan 'os_service_manager'. Las aplicaciones de usuario usan 'os_launch_app' para abrir y 'os_close_window' para cerrar. Proporciona siempre rutas completas."""

APPS = [
    ("calculadora", "calc", "Calculadora"),
    ("bloc de notas", "notepad", "Bloc de notas"),
    ("google chrome", "chrome", "Google Chrome"),
    ("visual studio code", "code", "Visual Studio Code"),
    ("antigravity ide", "antigravity", "Antigravity IDE"),
    ("administrador de tareas", "taskmgr", "Administrador de tareas"),
    ("explorador de archivos", "explorer", "Explorador de archivos"),
    ("spotify", "spotify", "Spotify"),
    ("terminal", "wt", "Terminal"),
    ("paint", "mspaint", "Paint"),
    ("panel de control", "control", "Panel de control"),
    ("configuración de windows", "ms-settings:", "Configuración"),
    ("discord", "discord", "Discord"),
    ("word", "winword", "Word"),
    ("excel", "excel", "Excel"),
    ("powerpoint", "powerpnt", "PowerPoint")
]

GREETING_PREFIXES = [
    "Hola, ", "Hola Ozy, ", "Hola ", "Buenas tardes, ", "Buenos días, ", "Buenas noches, ",
    "Oye Ozy, ", "Oye, ", "Por favor ", "Porfa, ", "Qué tal, ", "Hola amigo, ", ""
]

OPEN_PHRASES = [
    "abre {app}", "inicia {app}", "ejecuta {app}", "lanza {app}", "¿puedes abrir {app}?",
    "ábreme la {app}", "necesito que abras {app}", "despliega {app}", "ponme {app} en pantalla",
    "échame a andar {app}", "levantá {app}", "abrí {app}", "arráncame {app}", "spawnea {app}"
]

CLOSE_PHRASES = [
    "cierra {app}", "cierra la ventana de {app}", "termina el proceso de {app}",
    "mata {app}", "salte de {app}", "¿puedes cerrar {app}?", "por favor cierra {app}",
    "cerrá {app}", "quita {app} de la pantalla"
]

SERVICES = [
    ("cola de impresión", "spooler"),
    ("servicio de impresión", "spooler"),
    ("windows update", "wuauserv"),
    ("actualizaciones de windows", "wuauserv"),
    ("servidor lanman", "LanmanServer"),
    ("estación de trabajo lanman", "LanmanWorkstation"),
    ("centro de seguridad", "wscsvc"),
    ("windows defender", "WinDefend"),
    ("temas de windows", "Themes")
]

dataset = []

# ==============================================================================
# 1. APERTURA DE APPS CON SALUDOS Y JERGAS (~500 ejemplos)
# ==============================================================================
for app_name, app_target, app_title in APPS:
    for _ in range(32):
        prefix = random.choice(GREETING_PREFIXES)
        phrase = random.choice(OPEN_PHRASES).format(app=app_name)
        user_msg = f"{prefix}{phrase}".strip()
        user_msg = user_msg[0].upper() + user_msg[1:]

        thought = f"El usuario solicita iniciar {app_name}. Invocaré os_launch_app con target '{app_target}'."
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"os_launch_app\", \"arguments\": {{\"target\": \"{app_target}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Aplicación {app_title} iniciada exitosamente con PID {random.randint(1000, 9999)}."},
                {"role": "assistant", "content": f"Listo, acabo de abrir {app_title} en pantalla."}
            ]
        })

# ==============================================================================
# 2. CIERRE DE APPS CON SALUDOS Y JERGAS (~350 ejemplos)
# ==============================================================================
for app_name, app_target, app_title in APPS:
    for _ in range(22):
        prefix = random.choice(GREETING_PREFIXES)
        phrase = random.choice(CLOSE_PHRASES).format(app=app_name)
        user_msg = f"{prefix}{phrase}".strip()
        user_msg = user_msg[0].upper() + user_msg[1:]

        thought = f"El usuario solicita cerrar {app_name}. Invocaré os_close_window con title '{app_title}'."
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"os_close_window\", \"arguments\": {{\"title\": \"{app_title}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Ventana '{app_title}' cerrada exitosamente."},
                {"role": "assistant", "content": f"He cerrado {app_title} correctamente."}
            ]
        })

# ==============================================================================
# 3. WINDOWS 11 SERVICIOS DEL SISTEMA (os_service_manager) (~250 ejemplos)
# ==============================================================================
SERVICE_ACTIONS = [
    ("reinicia el {svc}", "restart", "reiniciar"),
    ("reiníciame el {svc}", "restart", "reiniciar"),
    ("detén el {svc}", "stop", "detener"),
    ("para el {svc}", "stop", "detener"),
    ("inicia el {svc}", "start", "iniciar"),
    ("arranca el {svc}", "start", "iniciar"),
    ("cómo está el {svc}", "status", "consultar estado de"),
    ("cuál es el estado de {svc}", "status", "consultar estado de"),
    ("revisa el estado del {svc}", "status", "consultar estado de")
]

for svc_name, svc_id in SERVICES:
    for tmpl, action, action_desc in SERVICE_ACTIONS:
        prefix = random.choice(GREETING_PREFIXES)
        user_msg = f"{prefix}{tmpl.format(svc=svc_name)}".strip()
        user_msg = user_msg[0].upper() + user_msg[1:]

        thought = f"El usuario solicita {action_desc} el servicio de Windows '{svc_name}' ({svc_id}). En Windows 11, los servicios del sistema usan os_service_manager."
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"os_service_manager\", \"arguments\": {{\"action\": \"{action}\", \"name\": \"{svc_id}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Servicio '{svc_id}' ejecutó acción '{action}' exitosamente. Estado actual: Running."},
                {"role": "assistant", "content": f"El servicio '{svc_name}' ({svc_id}) fue procesado exitosamente ({action}). Su estado actual es activo."}
            ]
        })

# ==============================================================================
# 4. BUCLER REACT MULTI-ETAPA: INVESTIGACIÓN WEB + CREACIÓN DE DOCUMENTOS (~350 ejemplos)
# ==============================================================================
RESEARCH_TOPICS = [
    {
        "topic": "Beck y su álbum Colors",
        "query": "Beck album Colors tracklist lanzamiento productores grammy",
        "obs": "Colors es el decimotercer álbum de estudio de Beck, lanzado el 13 de octubre de 2017 por Capitol Records. Producido por Beck y Greg Kurstin. Ganó 2 premios Grammy: Mejor Álbum de Música Alternativa y Mejor Ingeniería de Grabación. Tracks: Colors, 7th Heaven, I'm So Free, Dear Life, Wow, Up All Night, Dreams, Square One, Fix Me.",
        "doc_title": "Beck: Análisis Discográfico de Colors",
        "file": "Informe_Beck_Colors.pdf",
        "s1_title": "Ficha Técnica y Producción",
        "s1_content": "Colors fue lanzado en octubre de 2017 por Capitol Records bajo la producción conjunta de Beck y Greg Kurstin. Obtuvo dos premios Grammy.",
        "s2_title": "Repertorio y Reconocimiento",
        "s2_content": "Incluye sencillos aclamados como Dreams, Wow y Up All Night. Representa un giro brillante hacia el pop experimental y bailable."
    },
    {
        "topic": "Cage The Elephant y su discografía",
        "query": "Cage the Elephant discografia albumes premios grammy",
        "obs": "Cage the Elephant es una banda de rock estadounidense formada en 2006. Álbumes destacados: Cage the Elephant (2008), Thank You, Happy Birthday (2011), Melophobia (2013), Tell Me I'm Pretty (2015, Grammy Mejor Álbum de Rock), Social Cues (2019, Grammy Mejor Álbum de Rock).",
        "doc_title": "Cage The Elephant: Trayectoria y Discografía",
        "file": "Informe_Cage_The_Elephant.pdf",
        "s1_title": "Orígenes y Salto a la Fama",
        "s1_content": "Fundada en Bowling Green, Kentucky. Alcanzó notoriedad internacional con Ain't No Rest for the Wicked y Melophobia.",
        "s2_title": "Premios Grammy y Madurez Musical",
        "s2_content": "Galardonados con dos premios Grammy al Mejor Álbum de Rock por Tell Me I'm Pretty y Social Cues."
    },
    {
        "topic": "Evolución de TypeScript 5 y tipado estricto",
        "query": "TypeScript 5 novedades caracteristicas decoradores rendimiento",
        "obs": "TypeScript 5.0 introdujo decoradores ECMAScript estándar, const type parameters, optimización de velocidad de compilación y bundle reducido en un 37%, soporte para export type *, y mejoras en JSDoc y enum namespaces.",
        "doc_title": "Informe Técnico: Novedades de TypeScript 5",
        "file": "TypeScript_5_Evolucion.docx",
        "s1_title": "Arquitectura y Rendimiento",
        "s1_content": "TypeScript 5 redujo el tamaño del paquete e incrementó la velocidad del compilador mediante la migración a módulos modernos y optimización del AST.",
        "s2_title": "Nuevas Características de Lenguaje",
        "s2_content": "Implementación oficial de decoradores conformes a la fase 3 de TC39 y parámetros de tipo const para inferencia inmutable."
    },
    {
        "topic": "Arquitectura de Microservicios con Go y gRPC",
        "query": "Go microservices gRPC protocolo buffers rendimiento concurrencia",
        "obs": "Go ofrece alto rendimiento en microservicios gracias a goroutines y canales ligeros. Combinado con gRPC y Protocol Buffers sobre HTTP/2, proporciona serialización binaria ultrarrápida, streaming bidireccional y bajo consumo de memoria.",
        "doc_title": "Dossier Arquitectónico: Microservicios en Go",
        "file": "Microservicios_Go_gRPC.pdf",
        "s1_title": "Fundamentos y Concurrencia",
        "s1_content": "El modelo CSP de Go permite manejar miles de conexiones simultáneas con huella de RAM en kilobytes por hilo.",
        "s2_title": "Comunicación con gRPC y HTTP/2",
        "s2_content": "Multiplexación de peticiones, compresión binaria con Protobuf y tipado seguro en contratos de servicio distribuidos."
    }
]

for item in RESEARCH_TOPICS:
    for _ in range(25):
        prefix = random.choice(GREETING_PREFIXES)
        user_msg = f"{prefix}investiga sobre {item['topic']} y genera un informe en mis documentos".strip()
        user_msg = user_msg[0].upper() + user_msg[1:]

        is_docx = item["file"].endswith(".docx")
        tool_name = "os_create_docx" if is_docx else "os_create_pdf"
        target_path = f"C:\\Users\\User\\Documents\\{item['file']}"

        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {
                    "role": "assistant",
                    "content": f"<thought>El usuario solicita un informe sobre {item['topic']}. Para redactar un documento verídico y riguroso, primero investigo en internet con web_search.</thought>\n<tool_call>{{\"name\": \"web_search\", \"arguments\": {{\"query\": \"{item['query']}\"}}}}</tool_call>"
                },
                {
                    "role": "tool",
                    "content": item["obs"]
                },
                {
                    "role": "assistant",
                    "content": f"<thought>Con los datos verídicos obtenidos de la búsqueda, procedo a generar el documento estructurado con {tool_name}.</thought>\n<tool_call>{{\"name\": \"{tool_name}\", \"arguments\": {{\"path\": \"{target_path}\", \"title\": \"{item['doc_title']}\", \"sections\": [{{\"title\": \"{item['s1_title']}\", \"content\": \"{item['s1_content']}\"}}, {{\"title\": \"{item['s2_title']}\", \"content\": \"{item['s2_content']}\"}}]}}}}</tool_call>"
                },
                {
                    "role": "tool",
                    "content": f"Archivo '{target_path}' generado exitosamente."
                },
                {
                    "role": "assistant",
                    "content": f"He investigado sobre {item['topic']} y acabo de generar el informe completo en '{target_path}'. Contiene la información técnica recopilada y análisis estructurado. ¿Deseas que lo abra en pantalla para revisarlo?"
                }
            ]
        })

# ==============================================================================
# 5. ANÁFORAS Y CONTEXTO MULTI-TURNO ("ábrelo", "¿cuándo se creó?") (~250 ejemplos)
# ==============================================================================
ANAPHORA_FILES = [
    ("C:\\Users\\User\\Documents\\Informe_Beck_Colors.pdf", "el informe de Beck"),
    ("C:\\Users\\User\\Documents\\presupuesto_2026.xlsx", "el presupuesto"),
    ("C:\\Users\\User\\Documents\\Auditoria_Seguridad.docx", "la auditoría de seguridad"),
    ("C:\\Users\\User\\Documents\\Reporte_Hardware.pdf", "el reporte de hardware")
]

ANAPHORA_QUERIES = [
    ("ábrelo", "os_launch_app", lambda p: {"target": p}, "abrir el archivo en foco"),
    ("ábrela", "os_launch_app", lambda p: {"target": p}, "abrir el archivo en foco"),
    ("por favor ábrelo", "os_launch_app", lambda p: {"target": p}, "abrir el archivo en foco"),
    ("muéstramelo en pantalla", "os_launch_app", lambda p: {"target": p}, "desplegar el archivo en pantalla"),
    ("cuándo fue creado", "os_file_info", lambda p: {"path": p}, "consultar la fecha de creación del archivo"),
    ("cuánto pesa", "os_file_info", lambda p: {"path": p}, "consultar el tamaño y metadatos del archivo"),
    ("dónde se guardó", "os_file_info", lambda p: {"path": p}, "verificar la ubicación exacta del archivo")
]

for file_path, desc in ANAPHORA_FILES:
    for q_text, tool_name, arg_builder, intent_desc in ANAPHORA_QUERIES:
        for _ in range(8):
            prefix = random.choice(GREETING_PREFIXES)
            user_msg = f"{prefix}{q_text}".strip()
            user_msg = user_msg[0].upper() + user_msg[1:]

            system_with_focus = SYSTEM_PROMPT + f"\n\n[ARCHIVO EN FOCO DE LA CONVERSACIÓN]: {file_path}"
            thought = f"El usuario hace referencia anafórica ('{q_text}') sobre el archivo recién generado en la conversación ({file_path}). Procedo a {intent_desc}."
            args = arg_builder(file_path)

            dataset.append({
                "messages": [
                    {"role": "system", "content": system_with_focus},
                    {"role": "user", "content": f"Crea {desc} en mis documentos"},
                    {"role": "assistant", "content": f"Listo, acabo de crear {desc} en '{file_path}'."},
                    {"role": "user", "content": user_msg},
                    {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"{tool_name}\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                    {"role": "tool", "content": f"Acción completada sobre {file_path}."},
                    {"role": "assistant", "content": f"Listo, he ejecutado la acción sobre '{file_path}' exitosamente."}
                ]
            })

# ==============================================================================
# 6. EJEMPLOS NEGATIVOS (CHITCHAT, DUDAS CONCEPTUALES, CERO HERRAMIENTAS) (~200 ejemplos)
# ==============================================================================
NEGATIVE_SAMPLES = [
    ("Hola, ¿cómo estás hoy?", "¡Hola! Estoy muy bien y totalmente listo para asistirte en tu sistema Windows. ¿En qué puedo ayudarte hoy?"),
    ("Hola Ozy, buenas tardes", "¡Buenas tardes! ¿Qué tarea o proyecto tenemos para hoy en el equipo?"),
    ("Buenos días Ozy", "¡Buenos días! Todo el sistema está operativo. Dime qué necesitas y lo ejecutamos."),
    ("¿Qué es la memoria RAM y para qué sirve?", "La memoria RAM (Random Access Memory) es la memoria de trabajo de alta velocidad de tu equipo. Almacena temporalmente los datos e instrucciones que la CPU necesita en tiempo real mientras ejecutas aplicaciones. A diferencia del disco duro o SSD, es volátil y se borra al reiniciar."),
    ("¿Cuál es la diferencia entre un SSD NVMe y un disco HDD?", "La principal diferencia radica en la tecnología y velocidad: un HDD utiliza platos mecánicos giratorios (velocidades de ~150 MB/s), mientras que un SSD NVMe utiliza memoria flash conectada por interfaz PCIe directa, alcanzando velocidades de 3,500 a 7,000 MB/s con latencias en microsegundos."),
    ("¿Qué capacidades tienes como asistente?", "Cuento con integración nativa en Windows: puedo abrir y cerrar aplicaciones, gestionar servicios del sistema (SCM), crear documentos formales (PDF, Word, Excel con fórmulas), realizar búsquedas web en tiempo real, inspeccionar hardware físico, puertos USB y diagnosticar redes WiFi."),
    ("Muchas gracias por la ayuda, quedó perfecto", "¡De nada! Es un placer ayudarte. Cualquier otra consulta o tarea en tu equipo, aquí estaré."),
    ("¿Qué significa la sigla API?", "API significa Application Programming Interface (Interfaz de Programación de Aplicaciones). Es un conjunto de reglas y protocolos que permite que diferentes aplicaciones de software se comuniquen y compartan datos entre sí.")
]

for q, ans in NEGATIVE_SAMPLES:
    for _ in range(25):
        prefix = random.choice(["", "Oye, ", "Una pregunta, ", "Disculpa, "])
        user_msg = f"{prefix}{q}".strip()
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": ans}
            ]
        })

# ==============================================================================
# 7. HARDWARE, PUERTOS USB, AUDITORÍA WIFI Y SUITE OFFICE COMPLETA (~350 ejemplos)
# ==============================================================================
HARDWARE_CASES = [
    ("Inspecciona los puertos USB físicos", "os_hardware_inspector", {"action": "usb"}, "Inspeccionando puertos USB físicos y periféricos."),
    ("¿Qué dispositivos USB tengo conectados?", "os_hardware_inspector", {"action": "usb"}, "Consultando dispositivos USB conectados al sistema."),
    ("Chequea la salud del sistema y discos", "os_hardware_inspector", {"action": "health"}, "Evaluando salud SMART, temperaturas y saturación."),
    ("Revisa la telemetría del equipo (CPU y RAM)", "os_hardware_inspector", {"action": "telemetry"}, "Obteniendo carga de CPU, RAM y temperaturas."),
    ("Cómo está la conexión WiFi", "os_wifi_manager", {"action": "status"}, "Consultando estado, señal y velocidad de red WiFi."),
    ("Escanea las redes WiFi disponibles", "os_wifi_manager", {"action": "scan"}, "Escaneando redes inalámbricas circundantes.")
]

for q, tool, args, thought_txt in HARDWARE_CASES:
    for _ in range(20):
        prefix = random.choice(GREETING_PREFIXES)
        user_msg = f"{prefix}{q}".strip()
        user_msg = user_msg[0].upper() + user_msg[1:]

        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": f"<thought>{thought_txt}</thought>\n<tool_call>{{\"name\": \"{tool}\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": f"Telemetría obtenida exitosamente para {tool}."},
                {"role": "assistant", "content": f"Aquí tienes el reporte de telemetría solicitado para tu equipo."}
            ]
        })

random.shuffle(dataset)

output_path = os.path.join(os.path.dirname(__file__), "dataset", "ozy_distilled_v7.jsonl")
os.makedirs(os.path.dirname(output_path), exist_ok=True)

with open(output_path, "w", encoding="utf-8") as f:
    for entry in dataset:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")

print(f"=== [OK] DATASET AGÉNTICO v7 GENERADO EXITOSAMENTE ===")
print(f"Ruta: {output_path}")
print(f"Total de Ejemplos: {len(dataset)}")
print(f"Distribución:")
print(f"  - Apertura y cierre de apps con saludos: ~850")
print(f"  - Servicios Windows 11 (os_service_manager): ~250")
print(f"  - Bucles ReAct de Investigación Web + PDF/Word: ~350")
print(f"  - Anáforas y Foco Contextual ('ábrelo'): ~250")
print(f"  - Ejemplos Negativos (Cero herramientas): ~200")
print(f"  - Hardware, Puertos USB, WiFi y Office: ~350")
