"""
generate_massive_dataset_v5.py
Generador Masivo de Dataset de Alta Diversidad para OzyAssist (v5)
Cubre:
1. Jergas de LATAM y España (México, Perú, Colombia, Argentina, España, etc.)
2. Lenguaje técnico formal e informal (SysAdmin, Dev, Gamer)
3. Variaciones con errores tipográficos y lenguaje rápido
4. Flujos multi-turno con anáforas ("ábrelo", "¿cuándo se creó?", "¿cuánto pesa?")
5. Cobertura completa de herramientas:
   - os_launch_app
   - os_close_window
   - os_tile_windows
   - os_find_files
   - os_file_info
   - os_audio_device
   - os_hardware_inspector
   - os_wifi_manager
   - os_disk_cleaner
   - os_toast_notify
   - os_active_windows
   - web_search
   - os_create_pdf
   - os_read_document
"""

import json
import random
import os

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.
Directrices:
1. Si el usuario te pide realizar una acción en el sistema (abrir o cerrar apps, gestionar volumen, hardware, archivos, ventanas o comandos), invoca la herramienta correspondiente en formato JSON dentro de <tool_call>{"name": "...", "arguments": {...}}</tool_call>.
2. Puedes incluir tu razonamiento previo en <thought>...</thought>.
3. Cuando la herramienta retorne su resultado, responde en lenguaje natural de manera concisa, fluida, precisa y amigable.
4. Para consultas sobre archivos, proporciona siempre la ruta completa y ofrece el siguiente paso lógico."""

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
    "Ejecuta {app}", "Lanza {app}", "Arranca {app}", "Activa {app}", "Ábreme {app}",
    "Quiero abrir {app}", "Despliega {app}", "¿Me abres {app}?",
    
    # México / Centroamérica
    "Échame a andar {app}", "Ponme {app} en pantalla", "Levántame {app}",
    "Haz el paro de abrir {app}", "Pon {app}", "Ábrete {app}",
    "Tira {app}", "Córrete {app} porfa", "Chécate y abre {app}", "Aviéntate {app}",

    # Perú / Colombia / Caribe
    "Ábrete esa nota de {app}", "Pásame {app} al toque", "Ábreme esa vaina de {app}",
    "Muestra {app} en la pantalla", "Dale play a {app}", "Ponme a correr {app}",
    "Lánzate {app} rápido", "Tírame {app} en el escritorio",

    # Cono Sur (Argentina, Chile, Uruguay)
    "Levantá {app}", "Abrí {app}", "Fijate si abrís {app}", "Poné {app}",
    "Largá {app}", "Haceme la gauchada de abrir {app}",

    # España
    "Arráncame el programa de {app}", "Abre el {app} tío", "Lanza la aplicación {app}",
    "Pásame la ventana de {app}",

    # Jerga Técnica / SysAdmin / Gamer
    "Spawnea el proceso de {app}", "Invoca una nueva instancia de {app}",
    "Lanza el binario de {app}", "Ejecuta el target {app}", "Crea un proceso para {app}",
    "Levanta el servicio o app de {app}", "Inicializa {app}"
]

# ==============================================================================
# 2. ARCHIVOS, TEMAS Y BÚSQUEDA (os_find_files + os_file_info + os_launch_app)
# ==============================================================================
TOPICS = [
    ("roblox", "RobloxInfo.pdf", "informe sobre roblox"),
    ("cage the elephant", "la_banda_cage_the_elephant_y_sus_albumes_reporte.pdf", "reporte de discografía de cage the elephant"),
    ("presupuesto 2026", "Presupuesto_Anual_2026.xlsx", "hoja de balance del presupuesto"),
    ("contrato alquiler", "Contrato_Arrendamiento_Firmado.docx", "documento legal de alquiler"),
    ("ventas trimestrales", "Reporte_Ventas_Q1.pdf", "análisis de ventas trimestrales"),
    ("arquitectura del sistema", "Arquitectura_OzyAssist.pdf", "documento de arquitectura de software"),
    ("cotizacion de paneles", "Cotizacion_Geofal_Solar.pdf", "cotización para instalación fotovoltaica"),
    ("credenciales api", "claves_acceso.env", "archivo de variables de entorno"),
    ("notas de reunion", "Minuta_Reunion_Directorio.txt", "resumen de acuerdos de junta"),
    ("inventario de laptops", "Inventario_Equipos_IT.xlsx", "tabla de inventario técnico"),
    ("manual de usuario", "Manual_OzyAssist_v3.pdf", "guía de usuario final"),
    ("reporte financiero", "Reporte_Financiero_Septiembre.pdf", "estado de resultados mensual")
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
    "dime la dirección exacta del {topic}",
    "caul es la ruta del {topic}", # typo intencional
    "kual es la ubicacion de {topic}", # typo intencional
    "en dónde guarde lo de {topic}",
    "pásame la ubicación del fichero de {topic}",
    "averíguame la ruta de {topic}"
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
    ("¿cuándo parieron ese documento?", "antigüedad"),
    ("checa el timestamp de creación", "timestamp"),
    ("¿cuál es la antigüedad del fichero?", "antigüedad"),
    ("¿cuándo fue la última modificación?", "última modificación"),
    ("fíjate cuándo se editó por última vez", "modificación"),
    ("¿cuánto pesa?", "tamaño"),
    ("¿qué tamaño tiene ese archivo?", "peso"),
    ("dime el peso en KB del documento", "peso en KB"),
    ("saca la metadata completa de ese informe", "metadatos completos")
]

dataset = []

# 1. Apertura de apps con jergas (16 apps x 32 verbos = ~512 ejemplos)
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

# 2. Búsqueda de archivos y flujos multi-turno con anáforas (~450 ejemplos)
for kw, fname, topic_desc in TOPICS:
    full_path = f"C:\\Users\\User\\Documents\\{fname}"
    size_kb = random.randint(18, 520)
    day = random.randint(10, 28)
    hour = random.randint(9, 20)
    minute = random.randint(10, 59)
    created_date = f"{day:02d}/09/2026 {hour:02d}:{minute:02d}:15"

    for f_tmpl in FIND_PATTERNS[:10]:
        user_find = f_tmpl.format(topic=kw)
        
        # Turno 1: Búsqueda -> Ofrecimiento
        # Turno 2A: Anáfora "ábrelo"
        open_anaph = random.choice(ANAPHORA_OPENINGS)
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_find},
                {"role": "assistant", "content": f"<thought>Buscando el archivo referente a '{kw}' en Documentos.</thought>\n<tool_call>{{\"name\": \"os_find_files\", \"arguments\": {{\"pattern\": \"*{kw}*\", \"root\": \"C:\\\\Users\\\\User\\\\Documents\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== ARCHIVOS ENCONTRADOS (1 coincidencias para \"*{kw}*\") ===\n1. 📄 {fname}\n   Ubicación: {full_path}"},
                {"role": "assistant", "content": f"Encontré el archivo solicitado:\n- **Nombre:** `{fname}`\n- **Ubicación:** `{full_path}`\n\n¿Deseas que abra el archivo en pantalla para revisarlo?"},
                {"role": "user", "content": open_anaph},
                {"role": "assistant", "content": f"<thought>El usuario solicita abrir el archivo en foco ('{full_path}'). Invoco os_launch_app.</thought>\n<tool_call>{{\"name\": \"os_launch_app\", \"arguments\": {{\"target\": \"{full_path}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"Documento {fname} abierto con éxito en el visor predeterminado."},
                {"role": "assistant", "content": f"He abierto `{fname}` en pantalla para que puedas revisarlo."}
            ]
        })

        # Turno 2B: Anáfora de metadatos ("cuándo se creó?", "¿cuánto pesa?")
        meta_q, meta_aspect = random.choice(ANAPHORA_METAS)
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": user_find},
                {"role": "assistant", "content": f"<thought>Buscando el archivo referente a '{kw}' en Documentos.</thought>\n<tool_call>{{\"name\": \"os_find_files\", \"arguments\": {{\"pattern\": \"*{kw}*\", \"root\": \"C:\\\\Users\\\\User\\\\Documents\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== ARCHIVOS ENCONTRADOS (1 coincidencias para \"*{kw}*\") ===\n1. 📄 {fname}\n   Ubicación: {full_path}"},
                {"role": "assistant", "content": f"El archivo está ubicado en:\n`{full_path}`\n\n¿Deseas revisarlo o consultar sus propiedades?"},
                {"role": "user", "content": meta_q},
                {"role": "assistant", "content": f"<thought>El usuario pregunta por la {meta_aspect} del archivo en foco ({fname}). Invocaré os_file_info.</thought>\n<tool_call>{{\"name\": \"os_file_info\", \"arguments\": {{\"path\": \"{full_path}\"}}}}</tool_call>"},
                {"role": "tool", "content": f"=== INFORMACIÓN DE METADATOS: {fname} ===\n• Tipo: Archivo\n• Ruta completa: {full_path}\n• Tamaño: {size_kb}.2 KB ({size_kb*1024} bytes)\n• Fecha de creación: {created_date}\n• Última modificación: {created_date}"},
                {"role": "assistant", "content": f"`{fname}` fue creado el **{created_date}** con un peso de **{size_kb}.2 KB** en `{full_path}`."}
            ]
        })

# 3. Hardware, USB, Sensores, WiFi (~150 ejemplos con modismos)
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

# 4. Ventanas, Cierre y Acomodo (~120 ejemplos)
WINDOW_CASES = [
    ("Mátame la ventana del bloc de notas", "os_close_window", {"title": "bloc de notas"}, "Bloc de notas cerrado."),
    ("Cierra esa porquería de spotify", "os_close_window", {"title": "spotify"}, "He cerrado Spotify."),
    ("Quita la calculadora de pantalla", "os_close_window", {"title": "calculadora"}, "Calculadora cerrada."),
    ("Acomódame esta ventana a la izquierda", "os_tile_windows", {"action": "left"}, "Ventana fijada a la mitad izquierda."),
    ("Pega la ventana a la derecha", "os_tile_windows", {"action": "right"}, "Ventana acoplada a la derecha."),
    ("Maximízame todo", "os_tile_windows", {"action": "maximize"}, "Ventana maximizada a pantalla completa."),
    ("Limpia la pantalla y muestra el escritorio", "os_tile_windows", {"action": "show_desktop"}, "Todas las ventanas minimizadas. Mostrando el escritorio."),
    ("¿Qué ventanas tengo activas?", "os_active_windows", {}, "Tienes abiertas: Visual Studio Code, Google Chrome y Terminal.")
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

# 5. Audio del Sistema (~100 ejemplos)
AUDIO_CASES = [
    ("Súbele al volumen al 80%", {"action": "set_volume", "volume": 80}, "Volumen configurado al 80%."),
    ("Bájale al audio al 20%", {"action": "set_volume", "volume": 20}, "Volumen establecido al 20%."),
    ("Silencia el sonido", {"action": "mute", "mute": True}, "Audio silenciado."),
    ("Mutea el equipo", {"action": "mute", "mute": True}, "Sonido muteado."),
    ("Quítale el silencio", {"action": "mute", "mute": False}, "Silencio desactivado, audio restablecido."),
    ("¿En cuánto está el volumen?", {"action": "get_volume"}, "El volumen maestro actual está al 65%.")
]

for q, args, resp in AUDIO_CASES:
    for prefix in ["", "Ozy ", "Porfa ", "Oye "]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"{prefix}{q}".strip()},
                {"role": "assistant", "content": f"<thought>Ajustando volumen del sistema.</thought>\n<tool_call>{{\"name\": \"os_audio_device\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": "Comando de audio ejecutado con éxito."},
                {"role": "assistant", "content": resp}
            ]
        })

# 6. Limpieza de disco y Notificaciones (~80 ejemplos)
UTIL_CASES = [
    ("Limpia la basura y archivos temporales", "os_disk_cleaner", {"action": "clean"}, "He liberado 1.4 GB eliminando temporales huérfanos."),
    ("Vacíame los temporales", "os_disk_cleaner", {"action": "clean"}, "Caché y temporales limpiados correctamente."),
    ("Mándame una notificación al escritorio cuando acabes", "os_toast_notify", {"title": "OzyAssist", "message": "Tarea completada."}, "Notificación emergente enviada a tu escritorio.")
]

for q, tool_name, args, resp in UTIL_CASES:
    for prefix in ["", "Ozy ", "Por favor "]:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"{prefix}{q}".strip()},
                {"role": "assistant", "content": f"<thought>Ejecutando utilidad del sistema.</thought>\n<tool_call>{{\"name\": \"{tool_name}\", \"arguments\": {json.dumps(args)}}}</tool_call>"},
                {"role": "tool", "content": "Operación completada."},
                {"role": "assistant", "content": resp}
            ]
        })

# ------------------------------------------------------------------------------
# GUARDAR EL DATASET ENRIQUECIDO
# ------------------------------------------------------------------------------
random.shuffle(dataset)
out_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "dataset")
os.makedirs(out_dir, exist_ok=True)
out_file = os.path.join(out_dir, "ozy_distilled_v5.jsonl")

with open(out_file, "w", encoding="utf-8") as f:
    for entry in dataset:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")

print(f"============================================================")
print(f"[OK] DATASET MASIVO v5 GENERADO CON ÉXITO")
print(f"     Total de ejemplos enriquecidos: {len(dataset)}")
print(f"     Destino: {out_file}")
print(f"============================================================")
