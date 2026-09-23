import urllib.request
import json

system_prompt = """Eres OzyAssist, asistente para Windows.
Herramientas disponibles:
- os_launch_app: Inicia una aplicacion. Argumentos: {"target": "nombre"}
- os_close_window: Cierra una ventana. Argumentos: {"title": "nombre"}
- web_search: Busca en internet. Argumentos: {"query": "terminos"}

DIRECTRIZ:
Si el usuario pide abrir, cerrar, buscar o ejecutar algo, emite directamente:
<tool_call>{"name": "...", "arguments": {...}}</tool_call>"""

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
test_prompt("Abre la calculadora")
test_prompt("Por favor cierra el bloc de notas")
test_prompt("Investiga sobre Beck y su album Colors")
