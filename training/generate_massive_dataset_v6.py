"""
generate_massive_dataset_v6.py
Generador Masivo de Dataset de Alta Diversidad para OzyAssist (v6)
Cubre integralmente:
1. Suite Office Completa:
   - os_create_excel (.xlsx con tablas y formato Ozy #D1F107)
   - os_create_docx (.docx OpenXML con secciones y tablas)
   - os_create_pdf (.pdf formal con estilo institucional)
   - os_search_content (búsqueda de contenido/grep dentro de informes)
2. Búsqueda nativa ultrarrápida (< 100 ms) de cualquier archivo con telemetría explícita
3. Flujos multi-turno con anáforas ("ábrelo", "¿cuándo se creó?", "¿cuánto pesa?")
4. Jergas de LATAM y España (México, Perú, Colombia, Argentina, España, SysAdmin)
5. Hardware físico, puertos USB, sensores de privacidad (cámara web/micrófono), salud SMART y temperaturas
6. Configuración del sistema Windows (WiFi, volumen, acomodo de ventanas, limpieza de disco C)
7. Resolución multi-etapa de problemas reales de rendimiento y almacenamiento
"""

import json
import random
import os

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario actual: User (Ruta: C:\\Users\\User)
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.

DIRECTRICES:
1. Analiza brevemente la situación antes de actuar dentro de <thought>...</thought>.
2. Para ejecutar cualquier acción en Windows (apps, búsqueda web, reportes PDF/Excel/Word, hardware, wifi, buscar archivos), invoca la herramienta correspondiente en formato JSON dentro de:
<tool_call>{"name": "nombre_herramienta", "arguments": {...}}</tool_call>
3. Cuando la herramienta retorne su resultado, responde en lenguaje natural de manera concisa, fluida, precisa y amigable.
4. Para consultas sobre archivos, proporciona siempre la ruta completa exacta y ofrece el siguiente paso lógico."""

# ==============================================================================
# 1. CATÁLOGO DE APLICACIONES Y VARIACIONES DE APERTURA (os_launch_app)
# ==============================================================================
APPS = [
    ("calculadora", "calc", "calc.exe"),
    ("bloc de notas", "notepad", "notepad.exe"),
    ("google chrome", "chrome", "chrome.exe"),
    ("visual studio code", "code", "Code.exe"),
    ("antigravity ide", "antigravity", "antigravity.exe"),
    ("administrador de tareas", "taskmgr", "Taskmgr.exe"),
    ("explorador de archivos", "explorer", "explorer.exe"),
    ("spotify", "spotify", "Spotify.exe"),
    ("terminal", "wt", "wt.exe"),
    ("paint", "mspaint", "mspaint.exe"),
    ("panel de control", "control", "control.exe"),
    ("configuración de windows", "ms-settings:", "SystemSettings.exe"),
    ("discord", "discord", "Discord.exe"),
    ("word", "winword", "WINWORD.EXE"),
    ("excel", "excel", "EXCEL.EXE"),
    ("powerpoint", "powerpnt", "POWERPNT.EXE")
]

OPEN_VERBS = [
    # Neutro
    "Abre {app}", "Por favor abre {app}", "Abre la {app}", "¿Puedes abrir {app}?",
    "Inicia {app}", "Iníciame {app}", "Inicia la aplicación de {app}",
    "Ejecuta {app}", "Lanza {app}", "Arranca {app}", "Ábreme {app}",
    "Quiero abrir {app}", "Despliega {app}", "¿Me abres {app}?",
    
    # México / Centroamérica
    "Échame a andar {app}", "Ponme {app} en pantalla", "Levántame {app}",
    "Haz el paro de abrir {app}", "Ábrete {app}", "Tira {app}", "Chécate y abre {app}",

    # Perú / Colombia / Caribe
    "Ábrete esa nota de {app}", "Pásame {app} al toque", "Ábreme esa vaina de {app}",
    "Ponme a correr {app}", "Lánzate {app} rápido",

    # Cono Sur (Argentina, Chile, Uruguay)
    "Levantá {app}", "Abrí {app}", "Fijate si abrís {app}", "Poné {app}",
    "Haceme la gauchada de abrir {app}",

    # España
    "Arráncame el programa de {app}", "Abre el {app} tío", "Lanza la aplicación {app}",

    # Jerga Técnica / SysAdmin
    "Spawnea el proceso de {app}", "Invoca una nueva instancia de {app}",
    "Lanza el binario de {app}", "Ejecuta el target {app}"
]

dataset = []

# 1. Apertura de apps con jergas (~480 ejemplos)
for app_name, app_target, app_exe in APPS:
    for tmpl in OPEN_VERBS:
        user_msg = tmpl.format(app=app_name)
        thought = f"El usuario solicita iniciar {app_name}. Invocaré os_launch_app con target '{app_target}'."
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_msg},
                {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"os_launch_app\", \"arguments\": {{\"target\": \"{app_target}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Aplicación {app_exe} iniciada exitosamente con PID {random.randint(1000, 9999)}."},
                {"role": "assistant", "content": f"Listo, acabo de abrir {app_name} en pantalla."}
            ]
        })

# ==============================================================================
# 2. SUITE OFFICE COMPLETA (EXCEL, WORD, PDF, GREP)
# ==============================================================================

# 2A. EXCEL (.xlsx) - os_create_excel
EXCEL_SCENARIOS = [
    (
        "Presupuesto_Q4_2026.xlsx",
        "Presupuesto Q4 2026",
        ["Concepto", "Monto USD", "Responsable", "Estado"],
        [["Servidores Cloud", "1450.00", "DevOps", "Aprobado"], ["Licencias Software", "890.00", "IT", "Pendiente"], ["Infraestructura Red", "620.00", "Sistemas", "Aprobado"]],
        [
            "Crea una hoja de Excel en Documentos llamada Presupuesto_Q4_2026.xlsx con los gastos de infraestructura",
            "Arma un excel con el presupuesto del cuarto trimestre en Presupuesto_Q4_2026.xlsx",
            "Genera un archivo xlsx de presupuesto en Documentos con columnas Concepto, Monto, Responsable y Estado",
            "Hazme un excel Presupuesto_Q4_2026.xlsx al toque con los costos de servidores y licencias",
            "Ármame un libro de cálculo Presupuesto_Q4_2026.xlsx en Excel",
            "Quiero que crees una hoja de cálculo en Documentos llamada Presupuesto_Q4_2026.xlsx",
            "Exporta a Excel los costos en Presupuesto_Q4_2026.xlsx por favor"
        ]
    ),
    (
        "Inventario_Equipos_IT.xlsx",
        "Inventario de Equipos",
        ["ID", "Dispositivo", "Usuario Asignado", "Ubicación", "RAM (GB)"],
        [["IT-001", "Laptop HP ZBook", "Carlos M.", "Piso 3", "32"], ["IT-002", "Dell Precision", "Ana R.", "Remoto", "64"], ["IT-003", "ThinkPad P16", "Javier L.", "Piso 2", "32"]],
        [
            "Genera un Excel en Documentos llamado Inventario_Equipos_IT.xlsx con la lista de laptops y workstations",
            "Crea un libro de cálculo Inventario_Equipos_IT.xlsx con los equipos de cómputo de la empresa",
            "Échate una hoja de Excel Inventario_Equipos_IT.xlsx para registrar los dispositivos asignados",
            "Ármate un excel con el inventario de máquinas en Documentos/Inventario_Equipos_IT.xlsx",
            "Crea un archivo .xlsx para el inventario de TI"
        ]
    ),
    (
        "Balance_Ventas_Mensual.xlsx",
        "Balance de Ventas",
        ["Mes", "Ventas Totales", "Costo Operativo", "Margen Neto"],
        [["Julio", "$45,200", "$18,500", "59%"], ["Agosto", "$52,800", "$21,100", "60%"], ["Septiembre", "$61,400", "$23,900", "61%"]],
        [
            "Crea un archivo Excel Balance_Ventas_Mensual.xlsx con las métricas de ingresos de los últimos meses",
            "Ármame una hoja en Excel llamada Balance_Ventas_Mensual.xlsx con las ventas y costos",
            "Exporta las ventas a un archivo Excel Balance_Ventas_Mensual.xlsx en Documentos",
            "Genera una hoja de cálculo de balance de ventas en Documentos"
        ]
    )
]

for fname, title, headers, rows, prompts in EXCEL_SCENARIOS:
    full_path = f"C:\\Users\\User\\Documents\\{fname}"
    for p in prompts:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": p},
                {"role": "assistant", "content": f"<thought>El usuario solicita crear una hoja de cálculo nativa de Excel (.xlsx). Usaré os_create_excel.</thought>\n<tool_call>{{\"name\": \"os_create_excel\", \"arguments\": {{\"path\": \"{full_path}\", \"title\": \"{title}\", \"headers\": {json.dumps(headers)}, \"rows\": {json.dumps(rows)}}}}}</tool_call>"},
                {"role": "tool", "content": f"📊 === ARCHIVO EXCEL GENERADO EXITOSAMENTE ===\n• Archivo: {full_path}\n• Hojas: 1 ({title})\n• Filas: {len(rows)}\n• Estilo: Diseño OzyAssist con cabeceras en Verde Neón (#D1F107)"},
                {"role": "assistant", "content": f"He generado la hoja de cálculo de Excel exitosamente:\n- **Archivo:** `{full_path}`\n- **Título:** {title}\n- **Registros:** {len(rows)} filas con estilo nativo OzyAssist.\n\n¿Deseas que abra el archivo en pantalla para revisarlo?"}
            ]
        })

# 2B. WORD (.docx) - os_create_docx
WORD_SCENARIOS = [
    (
        "Plan_Lanzamiento_OzyAssist_v6.docx",
        "Plan de Lanzamiento OzyAssist v6",
        "Estrategia de Despliegue y Arquitectura",
        [
            {"title": "1. Resumen Ejecutivo", "content": "OzyAssist v6 incorpora un motor autónomo con soporte integral para la suite Office, arquitectura de búsqueda instantánea y diagnóstico de hardware físico.", "bullets": ["Latencia sub-100ms en búsqueda", "Zero-Docker nativo en Go", "Soporte multi-dialecto"]},
            {"title": "2. Hitos de Implementación", "content": "El cronograma contempla validación en entornos reales de Windows y estabilización de inferencia local."}
        ],
        {"headers": ["Fase", "Fecha", "Objetivo"], "rows": [["Fase 1", "21/09/2026", "Compilación y fine-tuning v6"], ["Fase 2", "22/09/2026", "Pruebas de estrés Office y HW"]]},
        [
            "Redacta un documento en Word sobre el plan de lanzamiento de OzyAssist v6 y guárdalo en Documentos",
            "Crea un archivo Word Plan_Lanzamiento_OzyAssist_v6.docx con la propuesta técnica y el cronograma",
            "Arma un documento formal en Word llamado Plan_Lanzamiento_OzyAssist_v6.docx",
            "Escribe un informe en formato .docx con el plan de despliegue de OzyAssist",
            "Genera un Word formal para el plan de lanzamiento v6 en Documentos",
            "Prepara un archivo docx con el plan estratégico de lanzamiento"
        ]
    ),
    (
        "Contrato_Servicios_Software.docx",
        "Contrato Marco de Servicios de Software",
        "Términos y Condiciones Legales",
        [
            {"title": "Cláusula Primera: Objeto", "content": "El presente contrato establece las bases para la prestación de servicios de consultoría, arquitectura y desarrollo de sistemas autónomos de escritorio.", "bullets": ["Confidencialidad estricta", "Propiedad intelectual reservada", "SLA del 99.9%"]},
            {"title": "Cláusula Segunda: Entregables", "content": "La entrega de módulos se realizará de acuerdo al cronograma técnico validado por ambas partes."}
        ],
        {"headers": ["Entregable", "Plazo", "Importe"], "rows": [["Módulo Agéntico Core", "15 días", "$4,500"], ["Integración Win32 & Office", "10 días", "$3,500"]]},
        [
            "Prepara un contrato formal en Word en Documentos llamado Contrato_Servicios_Software.docx",
            "Genera un documento Word con el contrato de prestación de servicios",
            "Redáctame un .docx con las cláusulas de servicios de software",
            "Arma un contrato en formato Word para servicios de software"
        ]
    )
]

for fname, title, subtitle, sections, table, prompts in WORD_SCENARIOS:
    full_path = f"C:\\Users\\User\\Documents\\{fname}"
    for p in prompts:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": p},
                {"role": "assistant", "content": f"<thought>El usuario solicita crear un documento formal en Microsoft Word (.docx). Invocaré os_create_docx.</thought>\n<tool_call>{{\"name\": \"os_create_docx\", \"arguments\": {{\"path\": \"{full_path}\", \"title\": \"{title}\", \"subtitle\": \"{subtitle}\", \"sections\": {json.dumps(sections)}, \"table\": {json.dumps(table)}}}}}</tool_call>"},
                {"role": "tool", "content": f"📝 === ARCHIVO WORD (.docx) GENERADO EXITOSAMENTE ===\n• Archivo: {full_path}\n• Título: {title}\n• Secciones: {len(sections)}\n• Formato: OpenXML estándar compatible con Microsoft Word"},
                {"role": "assistant", "content": f"He creado el documento de Word formalmente en:\n`{full_path}`\n\nIncluye {len(sections)} secciones estructuradas y la tabla de datos en formato OpenXML.\n¿Deseas que lo abra en Word para revisarlo?"}
            ]
        })

# 2C. PDF (.pdf) - os_create_pdf
PDF_SCENARIOS = [
    (
        "Informe_Ejecutivo_Infraestructura.pdf",
        "Informe Ejecutivo de Infraestructura",
        "Auditoría y Rendimiento de Servidores",
        [
            {"title": "1. Estado Operativo", "content": "Todos los servicios críticos operan dentro de los umbrales de latencia y disponibilidad exigidos.", "bullets": ["CPU promedio: 18%", "Memoria RAM: 45%", "Almacenamiento NVMe: Salud 100%"]},
            {"title": "2. Recomendaciones", "content": "Se aconseja mantener la política Zero-Docker y la compresión continua de memoria."}
        ],
        [
            "Crea un informe ejecutivo en PDF llamado Informe_Ejecutivo_Infraestructura.pdf en Documentos",
            "Genera un reporte PDF con la auditoría de infraestructura y servidores",
            "Saca un PDF ejecutivo con métricas de rendimiento en Documentos",
            "Arma un informe en PDF de infraestructura en Documentos"
        ]
    )
]

for fname, title, subtitle, sections, prompts in PDF_SCENARIOS:
    full_path = f"C:\\Users\\User\\Documents\\{fname}"
    for p in prompts:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": p},
                {"role": "assistant", "content": f"<thought>El usuario solicita un informe en formato PDF. Invocaré os_create_pdf.</thought>\n<tool_call>{{\"name\": \"os_create_pdf\", \"arguments\": {{\"path\": \"{full_path}\", \"title\": \"{title}\", \"subtitle\": \"{subtitle}\", \"sections\": {json.dumps(sections)}}}}}</tool_call>"},
                {"role": "tool", "content": f"📄 === ARCHIVO PDF GENERADO EXITOSAMENTE ===\n• Archivo: {full_path}\n• Título: {title}\n• Secciones: {len(sections)}\n• Estado: Válido (formato nativo PDF-1.3)"},
                {"role": "assistant", "content": f"Informe PDF generado con éxito en:\n`{full_path}`\n\n¿Deseas que lo abra en tu visor de PDFs predeterminado?"}
            ]
        })

# 2D. BÚSQUEDA DE CONTENIDO DENTRO DE INFORMES - os_search_content (Grep)
GREP_CASES = [
    ("presupuesto", "C:\\Users\\User\\Documents", "Presupuesto_Q4_2026.xlsx", "Fila 2: Servidores Cloud | $1450.00"),
    ("lanzamiento", "C:\\Users\\User\\Documents", "Plan_Lanzamiento_OzyAssist_v6.docx", "Hito 1: Lanzamiento programado para septiembre"),
    ("latencia", "C:\\Users\\User\\Documents", "Informe_Ejecutivo_Infraestructura.pdf", "Latencia promedio inferior a 100ms"),
    ("API_KEY", "C:\\Users\\User\\Documents\\ozyAsis", ".env", "GROQ_API_KEY=gsk_...")
]

for term, sdir, hit_file, hit_preview in GREP_CASES:
    for q in [
        f"Busca dentro de los informes dónde se menciona '{term}'",
        f"Rastrea en los documentos la palabra '{term}'",
        f"Haz un grep en {sdir} buscando '{term}'",
        f"¿En qué archivo está escrito '{term}'?"
    ]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": q},
                {"role": "assistant", "content": f"<thought>El usuario desea buscar texto o términos dentro de los archivos. Invocaré os_search_content.</thought>\n<tool_call>{{\"name\": \"os_search_content\", \"arguments\": {{\"pattern\": \"{term}\", \"dir\": \"{sdir}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== COINCIDENCIAS DE CONTENIDO ({sdir}) ===\n1. Archivo: {hit_file}\n   Línea: {hit_preview}"},
                {"role": "assistant", "content": f"Encontré coincidencias para **'{term}'** en:\n- **Archivo:** `{hit_file}`\n- **Extracto:** `{hit_preview}`\n\n¿Deseas abrir el archivo o inspeccionar el contenido completo?"}
            ]
        })

# ==============================================================================
# 3. ARCHIVOS, TEMAS Y BÚSQUEDA (<100MS) Y ANÁFORAS
# ==============================================================================
TOPICS = [
    ("roblox", "RobloxInfo.pdf", "informe sobre roblox"),
    ("cage the elephant", "la_banda_cage_the_elephant_y_sus_albumes_reporte.pdf", "reporte de discografía de cage the elephant"),
    ("presupuesto 2026", "Presupuesto_Q4_2026.xlsx", "hoja de balance del presupuesto"),
    ("contrato alquiler", "Contrato_Servicios_Software.docx", "documento legal de servicios"),
    ("plan lanzamiento", "Plan_Lanzamiento_OzyAssist_v6.docx", "plan de lanzamiento"),
    ("ventas trimestrales", "Balance_Ventas_Mensual.xlsx", "análisis de ventas mensuales"),
    ("infraestructura", "Informe_Ejecutivo_Infraestructura.pdf", "informe de infraestructura"),
    ("inventario de laptops", "Inventario_Equipos_IT.xlsx", "tabla de inventario técnico"),
    ("credenciales api", ".env", "archivo de variables de entorno")
]

FIND_PATTERNS = [
    "cuál es la ruta y ubicación de {topic}",
    "dónde está el archivo de {topic}",
    "busca el {topic}",
    "localízame el documento sobre {topic}",
    "rastrea el fichero de {topic}",
    "fíjate dónde guardé el {topic}",
    "dónde quedó metido el archivo de {topic}",
    "en qué carpeta tengo el {topic}",
    "búscame cualquier reporte que hable de {topic}",
    "tírame la ruta del {topic} por favor",
    "necesito saber dónde está el {topic}",
    "encuentra el {topic}",
    "checa si existe un archivo de {topic}",
    "dime la dirección exacta del {topic}"
]

ANAPHORA_OPENINGS = [
    "ábrelo", "ábrelo por favor", "abrelo", "ábrela", "abre el archivo", "abrirlo",
    "ponlo en pantalla para revisarlo", "muéstramelo", "dale, ábrelo ya", "sí, ábrelo",
    "lánzalo", "abre ese documento", "despliégalo", "échalo a andar"
]

ANAPHORA_METAS = [
    ("coméntame cuándo fueron creados", "fecha de creación"),
    ("¿cuándo fue creado?", "fecha de creación"),
    ("¿cuándo se creó?", "fecha de creación"),
    ("¿a qué hora lo hicieron?", "hora de creación"),
    ("¿cuándo lo guardaron en el disco?", "fecha de guardado"),
    ("dime la fecha de creación de ese archivo", "fecha exacta"),
    ("checa el timestamp de creación", "timestamp"),
    ("¿cuánto pesa?", "tamaño"),
    ("¿qué tamaño tiene ese archivo?", "peso"),
    ("dime el peso en KB del documento", "peso en KB"),
    ("saca la metadata completa de ese informe", "metadatos completos")
]

for kw, fname, topic_desc in TOPICS:
    full_path = f"C:\\Users\\User\\Documents\\{fname}"
    size_kb = random.randint(18, 520)
    day = random.randint(10, 21)
    hour = random.randint(9, 23)
    minute = random.randint(10, 59)
    created_date = f"{day:02d}/09/2026 {hour:02d}:{minute:02d}:15"

    for f_tmpl in FIND_PATTERNS[:8]:
        user_find = f_tmpl.format(topic=kw)
        ms = random.randint(4, 25)
        
        # Turno 1: Búsqueda (<100ms) -> Anáfora "ábrelo"
        open_anaph = random.choice(ANAPHORA_OPENINGS)
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_find},
                {"role": "assistant", "content": f"<thought>Buscando el archivo referente a '{kw}' de forma nativa en Documentos.</thought>\n<tool_call>{{\"name\": \"os_find_files\", \"arguments\": {{\"pattern\": \"*{kw}*\", \"root\": \"C:\\\\Users\\\\User\\\\Documents\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== ARCHIVOS ENCONTRADOS (1 coincidencias para \"*{kw}*\" · ⏱️ {ms} ms) ===\n1. 📄 {fname}\n   Ubicación: {full_path}"},
                {"role": "assistant", "content": f"Encontré el archivo solicitado en **{ms} ms**:\n- **Nombre:** `{fname}`\n- **Ubicación:** `{full_path}`\n\n¿Deseas que abra el archivo en pantalla para revisarlo?"},
                {"role": "user", "content": open_anaph},
                {"role": "assistant", "content": f"<thought>El usuario solicita abrir el archivo en foco ('{full_path}'). Invoco os_launch_app.</thought>\n<tool_call>{{\"name\": \"os_launch_app\", \"arguments\": {{\"target\": \"{full_path}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Documento {fname} abierto con éxito en el visor predeterminado."},
                {"role": "assistant", "content": f"He abierto `{fname}` en pantalla para que puedas revisarlo."}
            ]
        })

        # Turno 2: Búsqueda -> Anáfora de metadatos ("cuándo se creó?")
        meta_q, meta_aspect = random.choice(ANAPHORA_METAS)
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_find},
                {"role": "assistant", "content": f"<thought>Buscando el archivo referente a '{kw}' de forma nativa en Documentos.</thought>\n<tool_call>{{\"name\": \"os_find_files\", \"arguments\": {{\"pattern\": \"*{kw}*\", \"root\": \"C:\\\\Users\\\\User\\\\Documents\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== ARCHIVOS ENCONTRADOS (1 coincidencias para \"*{kw}*\" · ⏱️ {ms} ms) ===\n1. 📄 {fname}\n   Ubicación: {full_path}"},
                {"role": "assistant", "content": f"El archivo está ubicado en:\n`{full_path}`\n\n¿Deseas revisarlo o consultar sus propiedades?"},
                {"role": "user", "content": meta_q},
                {"role": "assistant", "content": f"<thought>El usuario pregunta por la {meta_aspect} del archivo en foco ({fname}). Invocaré os_file_info.</thought>\n<tool_call>{{\"name\": \"os_file_info\", \"arguments\": {{\"path\": \"{full_path}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== INFORMACIÓN DE METADATOS: {fname} ===\n• Tipo: Archivo\n• Ruta completa: {full_path}\n• Tamaño: {size_kb}.2 KB ({size_kb*1024} bytes)\n• Fecha de creación: {created_date}\n• Última modificación: {created_date}"},
                {"role": "assistant", "content": f"`{fname}` fue creado el **{created_date}** con un peso de **{size_kb}.2 KB** en `{full_path}`."}
            ]
        })

# ==============================================================================
# 4. HARDWARE FÍSICO, SENSORES, USB Y REDES (~150 ejemplos)
# ==============================================================================
HARDWARE_CASES = [
    ("¿Qué tanta RAM se está tragando el equipo?", "os_hardware_inspector", {"action": "telemetry"}, "Consultando uso de RAM y CPU.", "Memoria RAM al 48% (7.6 GB usados de 16 GB), CPU estable al 15%."),
    ("Checa la temperatura de la gráfica", "os_hardware_inspector", {"action": "telemetry"}, "Consultando sensores de GPU.", "La GPU está a 49°C con 3.1 GB de VRAM libre."),
    ("Revisa si hay algún pendrive o memoria USB puesta", "os_hardware_inspector", {"action": "usb"}, "Inspeccionando puertos USB físicos.", "Dispositivos USB conectados:\n- Concentrador raíz USB 3.0 (OK)\n- Unidad USB Kingston 32GB en D:\\"),
    ("¿Está encendida la cámara o el micro?", "os_hardware_inspector", {"action": "usb"}, "Auditoría de periféricos sensibles.", "La cámara HP HD Camera está en reposo y ningún proceso la está ocupando."),
    ("Tírame un diagnóstico de salud del hardware", "os_hardware_inspector", {"action": "health"}, "Evaluando SMART y temperaturas.", "Hardware saludable: Discos NVMe en óptimo estado (0 errores SMART) y temperaturas por debajo de 55°C."),
    ("¿A qué WiFi estamos pegados?", "os_wifi_manager", {"action": "status"}, "Consultando estado de red WiFi.", "Conectado a la red **ANGEL GS** (1.2 Gbps enlace, IP 192.168.1.6)."),
    ("Checa qué onda con la red inalámbrica", "os_wifi_manager", {"action": "status"}, "Verificando tarjeta de red inalámbrica.", "WiFi activo y con señal excelente (100%), puerta de enlace 192.168.1.1."),
    ("Escanea las redes WiFi del vecindario", "os_wifi_manager", {"action": "scan"}, "Escaneando SSIDs disponibles.", "Redes WiFi encontradas:\n1. ANGEL GS (-45 dBm)\n2. Fibra_5G (-62 dBm)\n3. Red_Vecino (-80 dBm)")
]

for q, tool_name, args, thought, resp in HARDWARE_CASES:
    for prefix in ["Ozy, ", "Oye ", "Por favor ", "", "Dime: "]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"{prefix}{q}"},
                {"role": "assistant", "content": f"<thought>{thought}</thought>\n<tool_call>{{\"name\": \"{tool_name}\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": f"Resultado obtenido para {tool_name}."},
                {"role": "assistant", "content": resp}
            ]
        })

# ==============================================================================
# 5. RESOLUCIÓN DE PROBLEMAS REALES (MULTI-TURNO)
# ==============================================================================
# Problema 1: Disco C casi lleno -> Diagnóstico -> Limpieza
dataset.append({
    "messages": [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Oye, siento que el disco C está casi lleno y la máquina anda algo pesada. ¿Qué podemos hacer?"},
        {"role": "assistant", "content": "<thought>El usuario reporta problemas de espacio en disco. Primero evaluaré la salud y saturación del almacenamiento con os_hardware_inspector.</thought>\n<tool_call>{\"name\": \"os_hardware_inspector\", \"arguments\": {\"action\": \"health\"}}</tool_call>"},
        {"role": "tool", "content": "=== AUDITORÍA PREVENTIVA DE SALUD DEL HARDWARE ===\n• Almacenamiento:\n  - C: 94.2% usado (solo 12.8 GB libres de 220 GB) [ALERTA ESPACIO BAJO]\n  - Temporales detectados: ~3.4 GB purga segura\n• Temperaturas: 52°C (Normal)\n• SMART: OK (0 sectores defectuosos)"},
        {"role": "assistant", "content": "Efectivamente, el volumen **C:** está al **94.2% de capacidad** (solo quedan 12.8 GB libres). Sin embargo, he detectado aproximadamente **3.4 GB de archivos temporales huérfanos** que podemos purgar de forma completamente segura.\n\n¿Deseas que ejecute la limpieza del disco ahora mismo?"},
        {"role": "user", "content": "Sí, límpialo de una vez por favor"},
        {"role": "assistant", "content": "<thought>El usuario autoriza la limpieza de temporales huérfanos. Invocaré os_disk_cleaner.</thought>\n<tool_call>{\"name\": \"os_disk_cleaner\", \"arguments\": {\"action\": \"clean\"}}</tool_call>"},
        {"role": "tool", "content": "=== PURGA Y LIMPIEZA DE DISCO COMPLETADA ===\n• Espacio liberado: 3,420 MB (3.34 GB)\n• Archivos temporales eliminados: 1,842\n• Papelera de reciclaje: Vaciada\n• Estado C: 16.2 GB libres"},
        {"role": "assistant", "content": "¡Listo! He liberado **3.34 GB** de espacio en el disco C eliminando temporales y vaciando la papelera. Tu almacenamiento ahora tiene 16.2 GB libres."}
    ]
})

# Problema 2: WiFi Lento -> Diagnóstico -> Ping
dataset.append({
    "messages": [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "¿Por qué siento la conexión a internet tan lenta? Revisa el WiFi a ver qué pasa."},
        {"role": "assistant", "content": "<thought>El usuario reporta lentitud de red. Consultaré el estado de la conexión WiFi y la señal.</thought>\n<tool_call>{\"name\": \"os_wifi_manager\", \"arguments\": {\"action\": \"status\"}}</tool_call>"},
        {"role": "tool", "content": "=== TELEMETRÍA DE RED INALÁMBRICA (WI-FI) ===\n• Interfaz: Wi-Fi 6 AX201\n• SSID Activo: ANGEL GS\n• Señal: 98% (-46 dBm, Excelente)\n• Velocidad de enlace: 1200 Mbps (Tx) / 1200 Mbps (Rx)\n• Canal: 36 (Banda 5 GHz)"},
        {"role": "assistant", "content": "La tarjeta Wi-Fi y el enlace con el router están en perfectas condiciones:\n- **SSID:** `ANGEL GS` (Banda 5 GHz)\n- **Intensidad de señal:** 98% (Excelente)\n- **Velocidad de enlace local:** 1200 Mbps\n\nEl enlace inalámbrico no presenta degradación. ¿Deseas que haga una prueba de ping y latencia externa con 'os_network_diagnostics' para verificar si el problema es del proveedor de internet?"},
        {"role": "user", "content": "Dale, haz la prueba de ping"},
        {"role": "assistant", "content": "<thought>Ejecutaré el diagnóstico de red y ping externo para verificar latencia y pérdida de paquetes.</thought>\n<tool_call>{\"name\": \"os_network_diagnostics\", \"arguments\": {\"target\": \"8.8.8.8\"}}</tool_call>"},
        {"role": "tool", "content": "=== DIAGNÓSTICO DE RED E INTERNET ===\n• Destino: 8.8.8.8 (Google DNS)\n• Paquetes: 4 enviados, 4 recibidos (0% pérdida)\n• Latencia mínima: 14 ms | Máxima: 18 ms | Media: 15 ms\n• Puerta de enlace local: 192.168.1.1 (1 ms)"},
        {"role": "assistant", "content": "La conexión a internet responde con **15 ms de latencia media** y **0% de pérdida de paquetes**. La red física y el internet están funcionando a máxima velocidad."}
    ]
})

# ==============================================================================
# 6. AUDIO Y VENTANAS (~150 ejemplos)
# ==============================================================================
AUDIO_CASES = [
    ("Sube el volumen al 75%", {"action": "set_volume", "volume": 75}, "Volumen maestro ajustado al 75%."),
    ("Baja el volumen al 30%", {"action": "set_volume", "volume": 30}, "Volumen bajado al 30%."),
    ("Mutea el audio", {"action": "mute", "mute": True}, "Audio silenciado."),
    ("Desmutea el sonido", {"action": "mute", "mute": False}, "Sonido activado."),
    ("¿A cuánto está el volumen maestro?", {"action": "get_volume"}, "El volumen maestro actual está al 65%.")
]

for q, args, resp in AUDIO_CASES:
    for prefix in ["", "Ozy ", "Porfa ", "Oye "]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"{prefix}{q}".strip()},
                {"role": "assistant", "content": f"<thought>Ajustando dispositivo de audio del sistema.</thought>\n<tool_call>{{\"name\": \"os_audio_device\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": "Comando de audio ejecutado con éxito."},
                {"role": "assistant", "content": resp}
            ]
        })

WINDOW_CASES = [
    ("Mátame la ventana del bloc de notas", "os_close_window", {"title": "bloc de notas"}, "Bloc de notas cerrado."),
    ("Cierra esa porquería de spotify", "os_close_window", {"title": "spotify"}, "He cerrado Spotify."),
    ("Quita la calculadora de pantalla", "os_close_window", {"title": "calculadora"}, "Calculadora cerrada."),
    ("Acomódame esta ventana a la izquierda", "os_tile_windows", {"action": "left"}, "Ventana fijada a la mitad izquierda."),
    ("Pega la ventana a la derecha", "os_tile_windows", {"action": "right"}, "Ventana acoplada a la derecha."),
    ("Maximízame todo", "os_tile_windows", {"action": "maximize"}, "Ventana maximizada a pantalla completa."),
    ("Limpia la pantalla y muestra el escritorio", "os_tile_windows", {"action": "show_desktop"}, "Todas las ventanas minimizadas. Mostrando el escritorio.")
]

for q, tool_name, args, resp in WINDOW_CASES:
    for prefix in ["Ozy ", "Porfa ", "", "Oye hazme el paro: "]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"{prefix}{q}".strip()},
                {"role": "assistant", "content": f"<thought>Gestionando ventana del sistema.</thought>\n<tool_call>{{\"name\": \"{tool_name}\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": "Acción sobre ventana completada."},
                {"role": "assistant", "content": resp}
            ]
        })

# ------------------------------------------------------------------------------
# GUARDAR EL DATASET ENRIQUECIDO v6
# ------------------------------------------------------------------------------
random.shuffle(dataset)
out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "dataset")
os.makedirs(out_dir, exist_ok=True)
out_file = os.path.join(out_dir, "ozy_distilled_v6.jsonl")

with open(out_file, "w", encoding="utf-8") as f:
    for entry in dataset:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")

print(f"============================================================")
print(f"[OK] DATASET MASIVO v6 GENERADO CON ÉXITO")
print(f"     Total de ejemplos enriquecidos: {len(dataset)}")
print(f"     Destino: {out_file}")
print(f"============================================================")
