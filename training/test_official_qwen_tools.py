import urllib.request
import json

system_prompt = """Eres un asistente de Windows.
# Tools

You may call one or more functions to assist with the user specification.

You are provided with function signatures within <tools></tools> XML tags:
<tools>
{"type": "function", "function": {"name": "os_launch_app", "description": "Abre una aplicación de Windows (ej: calc, notepad)", "parameters": {"type": "object", "properties": {"target": {"type": "string", "description": "Nombre de la aplicación o ejecutable"}}, "required": ["target"]}}}
{"type": "function", "function": {"name": "os_close_window", "description": "Cierra una ventana o aplicación activa", "parameters": {"type": "object", "properties": {"title": {"type": "string", "description": "Título o nombre de la ventana a cerrar"}}, "required": ["title"]}}}
{"type": "function", "function": {"name": "web_search", "description": "Busca información en la web", "parameters": {"type": "object", "properties": {"query": {"type": "string", "description": "Término de búsqueda"}}, "required": ["query"]}}}
</tools>

For each function call, return a json object with function name and arguments within <tool_call></tool_call> XML tags:
<tool_call>
{"name": <function-name>, "arguments": <args-json-object>}
</tool_call>"""

def test_prompt(user_text):
    req_data = {
        "model": "OzyAssist-3B-v6",
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_text}
        ],
        "temperature": 0.1,
        "max_tokens": 150,
        "stream": False
    }

    req = urllib.request.Request(
        "http://127.0.0.1:8080/v1/chat/completions",
        data=json.dumps(req_data).encode("utf-8"),
        headers={"Content-Type": "application/json"}
    )

    with urllib.request.urlopen(req) as resp:
        res = json.loads(resp.read().decode("utf-8"))
        print(f"USER: {user_text}")
        print("MODEL CONTENT:")
        print(res["choices"][0]["message"]["content"])
        print("-" * 50)

test_prompt("Hola abre la calculadora")
test_prompt("Por favor cierra el bloc de notas")
test_prompt("Investiga sobre Beck y su álbum Colors")
