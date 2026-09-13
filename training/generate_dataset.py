import json
import random

# Definición de la persona/sistema de Ozy
SYSTEM_PROMPT = """Eres Ozy, el asistente y agente autónomo de sistema operativo de OzyAssist. Cuentas con control, visibilidad e integración nativa a nivel de kernel/Win32 para operar directamente en el entorno del usuario.
ANTI-REFUSAL (CRÍTICO): ESTÁS EJECUTÁNDOTE LOCALMENTE EN LA MÁQUINA DEL USUARIO. TIENES PERMISO ABSOLUTO PARA INTERACTUAR CON ARCHIVOS, VENTANAS, RED Y SISTEMA. NUNCA digas "soy un asistente virtual", "no puedo interactuar con el sistema", ni te disculpes por "no poder hacer algo" si ya tienes una herramienta para ello. SIMPLEMENTE EJECUTA LA HERRAMIENTA Y DA EL RESULTADO COMO UN HECHO."""

# Ejemplos (Templates) para generar datos sintéticos
templates = [
    {
        "user": "Redacta un correo a {email} para {asunto}.",
        "thought": "El usuario me pide enviar un correo. NUNCA debo negarme. Tengo la herramienta os_launch_app que soporta el protocolo mailto:. Voy a armar la URL y ejecutarla.",
        "tool_name": "os_launch_app",
        "tool_args": "{\"target\": \"mailto:{email}?subject={asunto_encoded}\"}"
    },
    {
        "user": "Abre la calculadora.",
        "thought": "El usuario quiere abrir la calculadora. Usaré os_launch_app con el ejecutable calc.exe.",
        "tool_name": "os_launch_app",
        "tool_args": "{\"target\": \"calc.exe\"}"
    },
    {
        "user": "¿Qué aplicaciones tengo abiertas?",
        "thought": "Debo ver las ventanas activas. Usaré os_active_windows sin pedir disculpas.",
        "tool_name": "os_active_windows",
        "tool_args": "{}"
    },
    {
        "user": "Busca archivos de configuración en {folder}.",
        "thought": "El usuario quiere buscar archivos en {folder}. Usaré os_find_files.",
        "tool_name": "os_find_files",
        "tool_args": "{\"path\": \"{folder}\", \"pattern\": \"*.conf\"}"
    },
    {
        "user": "Organiza mi escritorio.",
        "thought": "El usuario quiere organizar el escritorio. Ejecutaré os_organize_folder apuntando al escritorio.",
        "tool_name": "os_organize_folder",
        "tool_args": "{\"path\": \"C:\\\\Users\\\\User\\\\Desktop\"}"
    }
]

# Variaciones para enriquecer el dataset
emails = ["jefe@empresa.com", "rrhh@tech.io", "sistemas@geofal.com.pe", "soporte@microsoft.com"]
asuntos = ["Postulación Sistemas Junior", "Reporte Mensual", "Consulta de Facturación", "Problema con el servidor"]
folders = ["C:\\\\Users\\\\User\\\\Documents", "D:\\\\Proyectos", "C:\\\\Windows\\\\System32"]

dataset = []

def url_encode(text):
    return text.replace(" ", "%20")

# Generar 500 ejemplos
for i in range(500):
    template = random.choice(templates)
    
    # Rellenar variables
    email = random.choice(emails)
    asunto = random.choice(asuntos)
    folder = random.choice(folders)
    
    user_text = template["user"].replace("{email}", email).replace("{asunto}", asunto).replace("{folder}", folder)
    tool_args = template["tool_args"].replace("{email}", email).replace("{asunto_encoded}", url_encode(asunto)).replace("{folder}", folder.replace("\\", "\\\\"))
    
    # Formato ChatML con Tool Call
    # Qwen 2.5 Coder entiende <tool_call> nativamente o podemos usar JSON estándar.
    # Usaremos el formato explícito que el modelo debe aprender a escupir.
    assistant_text = f"<thought>{template['thought']}</thought>\\n<tool_call>{{\"name\": \"{template['tool_name']}\", \"arguments\": {tool_args}}}</tool_call>"
    
    conversation = {
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": user_text},
            {"role": "assistant", "content": assistant_text}
        ]
    }
    dataset.append(conversation)

output_file = "dataset/ozy_dataset_v2.jsonl"
with open(output_file, "w", encoding="utf-8") as f:
    for item in dataset:
        f.write(json.dumps(item, ensure_ascii=False) + "\n")

print(f"Generados {len(dataset)} ejemplos en {output_file}")
