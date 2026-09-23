import urllib.request
import json

# Esquema de herramientas de OzyAssist según estándares BFCL / Hermes / Qwen
tools_definition = [
    {
        "name": "os_launch_app",
        "description": "Abre una aplicación de Windows, ejecutable, archivo o protocolo (ej: 'calc', 'notepad', 'chrome', 'spotify', 'ms-settings:').",
        "parameters": {
            "type": "object",
            "properties": {
                "target": {"type": "string", "description": "Nombre de la aplicación, binario o ruta a abrir"}
            },
            "required": ["target"]
        }
    },
    {
        "name": "os_close_window",
        "description": "Cierra una ventana activa o proceso en ejecución por su título o nombre (ej: 'calculadora', 'bloc de notas', 'chrome').",
        "parameters": {
            "type": "object",
            "properties": {
                "title": {"type": "string", "description": "Título o nombre de la ventana a cerrar"}
            },
            "required": ["title"]
        }
    },
    {
        "name": "web_search",
        "description": "Busca información actualizada en la web a través de internet.",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Término o consulta de búsqueda"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "os_service_manager",
        "description": "Gestiona e inspecciona servicios en segundo plano de Windows (SCM) como spooler, wuauserv, etc.",
        "parameters": {
            "type": "object",
            "properties": {
                "action": {"type": "string", "enum": ["list", "status", "start", "stop", "restart"], "description": "Acción a realizar"},
                "name": {"type": "string", "description": "Nombre del servicio de Windows"}
            },
            "required": ["action"]
        }
    },
    {
        "name": "os_create_pdf",
        "description": "Genera un documento PDF formal y estructurado en el sistema.",
        "parameters": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Ruta destino del archivo PDF"},
                "title": {"type": "string", "description": "Título del documento"},
                "sections": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "title": {"type": "string"},
                            "content": {"type": "string"}
                        },
                        "required": ["title", "content"]
                    }
                }
            },
            "required": ["path", "title", "sections"]
        }
    }
]

system_prompt = f"""Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario: User (Ruta: C:\\Users\\User)

# HERRAMIENTAS DISPONIBLES:
{json.dumps(tools_definition, indent=2, ensure_ascii=False)}

# INSTRUCCIONES DE OPERACIÓN:
1. Conversación y Saludos: Si el usuario te saluda, conversa o hace preguntas conceptuales, responde en lenguaje natural amigable y conciso sin invocar herramientas.
2. Invocación de Herramientas: Si el usuario solicita una acción (abrir/cerrar apps, gestionar servicios, buscar archivos o investigar en la web), analiza la situación en <thought>...</thought> y emite inmediatamente:
<tool_call>
{{"name": "nombre_herramienta", "arguments": {{...}}}}
</tool_call>
3. Razonamiento ReAct: Para investigaciones o informes, ejecuta primero 'web_search' para recabar datos reales antes de crear cualquier documento.
4. Servicios vs Apps: En Windows 11, los servicios del sistema (como la cola de impresión 'spooler' o Windows Update 'wuauserv') usan 'os_service_manager'. Las aplicaciones de usuario (calculadora, bloc de notas, etc.) usan 'os_launch_app' o 'os_close_window'."""

def test_query(prompt_text):
    req_data = {
        "model": "OzyAssist-3B-v6",
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt_text}
        ],
        "temperature": 0.1,
        "max_tokens": 200,
        "stream": False
    }

    req = urllib.request.Request(
        "http://127.0.0.1:8080/v1/chat/completions",
        data=json.dumps(req_data).encode("utf-8"),
        headers={"Content-Type": "application/json"}
    )

    try:
        with urllib.request.urlopen(req) as resp:
            res = json.loads(resp.read().decode("utf-8"))
            print(f">>> USUARIO: {prompt_text}")
            print(">>> MODELO:")
            print(res["choices"][0]["message"]["content"])
            print("=" * 60)
    except Exception as e:
        print("Error:", e)

test_query("Hola, ¿cómo estás hoy?")
test_query("Hola abre la calculadora")
test_query("Hola cierra la calculadora")
test_query("Por favor reinicia el servicio de cola de impresión")
test_query("Investiga sobre Beck y su álbum Colors para hacer un informe")
