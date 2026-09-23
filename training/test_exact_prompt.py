import urllib.request
import json

system_prompt = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario actual: User (Ruta: C:\\Users\\User)
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.

DIRECTRICES:
1. Analiza brevemente la situación antes de actuar dentro de <thought>...</thought>.
2. Para ejecutar cualquier acción en Windows (apps, búsqueda web, reportes PDF, hardware, wifi, buscar archivos), invoca la herramienta correspondiente en formato JSON dentro de:
<tool_call>{"name": "nombre_herramienta", "arguments": {...}}</tool_call>"""

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

test_prompt("Abre la calculadora")
test_prompt("Hola abre la calculadora")
test_prompt("Cierra el bloc de notas")
test_prompt("Hola cierra el bloc de notas")
