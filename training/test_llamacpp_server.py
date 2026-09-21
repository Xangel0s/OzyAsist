import urllib.request
import json
import time

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de inteligencia artificial para Windows. Tienes integración nativa para operar directamente en el entorno de escritorio del usuario.
Tu objetivo es ayudar al usuario de forma rápida, precisa y con personalidad amigable y profesional.
Cuando el usuario te pida interactuar con el sistema (abrir o cerrar apps, gestionar archivos, audio, red o comandos), debes invocar la herramienta correspondiente en formato JSON.
Cuando la herramienta termine de ejecutarse, responde en lenguaje natural confirmando la acción de manera concisa y clara."""

def query_server(prompt, system=SYSTEM_PROMPT):
    url = "http://127.0.0.1:8080/v1/chat/completions"
    payload = {
        "model": "OzyAssist-7B-v3-q4_k_m.gguf",
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.2,
        "max_tokens": 128,
        "stop": ["<|im_end|>", "<|im_start|>", "</s>"]
    }
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req) as resp:
            elapsed = time.time() - t0
            res = json.loads(resp.read().decode("utf-8"))
            content = res["choices"][0]["message"]["content"]
            usage = res.get("usage", {})
            print(f"Prompt: {prompt}")
            print(f"Elapsed: {elapsed:.2f}s | Tokens: {usage}")
            print(f"Respuesta:\n{content}")
            print("-" * 50)
            return content
    except Exception as e:
        print(f"Error: {e}")
        return None

if __name__ == "__main__":
    print("=== TESTEANDO MOTOR LOCAL (OPCIÓN B - OZYASSIST-7B-V3) ===\n")
    query_server("Hola Ozy, ¿cómo estás?")
    query_server("Abre la calculadora")
    query_server("Estamos en vivo en el stream")
    query_server("Cierra la calculadora")
