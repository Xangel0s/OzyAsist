import urllib.request
import json
import time

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de inteligencia artificial para Windows. Tienes integración nativa para operar directamente en el entorno de escritorio del usuario.
Tu objetivo es ayudar al usuario de forma rápida, precisa y con personalidad amigable y profesional.
Cuando el usuario te pida interactuar con el sistema (abrir o cerrar apps, gestionar archivos, audio, red o comandos), debes invocar la herramienta correspondiente en formato JSON.
Cuando la herramienta termine de ejecutarse, responde en lenguaje natural confirmando la acción de manera concisa y clara."""

TEST_PROMPTS = [
    ("Conversación & Identidad", "Hola Ozy, ¿quién eres y qué puedes hacer?"),
    ("Livestream & Audiencia", "Estamos en vivo en el stream, saluda a todos"),
    ("Herramienta: Abrir App", "Abre la calculadora"),
    ("Herramienta: Cerrar Ventana", "Cierra el bloc de notas"),
    ("Herramienta: Control Audio", "Sube el volumen al 80%"),
    ("Herramienta: Hardware/GPU", "¿Cómo está el hardware y la temperatura de la GPU?"),
    ("Modo Voz: Brevedad Oral", "Ozy, confírmame si me escuchas bien")
]

def run_benchmark(model_label):
    url = "http://127.0.0.1:8080/v1/chat/completions"
    results = []
    print(f"\n{'='*70}")
    print(f"   EJECUTANDO BENCHMARK: {model_label}")
    print(f"{'='*70}\n")
    
    for category, prompt in TEST_PROMPTS:
        payload = {
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.2,
            "max_tokens": 120,
            "stop": ["<|im_end|>", "<|im_start|>", "</s>"]
        }
        data = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        
        t0 = time.perf_counter()
        try:
            with urllib.request.urlopen(req) as resp:
                elapsed = time.perf_counter() - t0
                res = json.loads(resp.read().decode("utf-8"))
                content = res["choices"][0]["message"]["content"].strip()
                usage = res.get("usage", {})
                completion_tokens = usage.get("completion_tokens", 0)
                tps = completion_tokens / elapsed if elapsed > 0 else 0
                
                print(f"[{category}]")
                print(f"User: {prompt}")
                print(f"Ozy:  {content}")
                print(f">> Tiempo: {elapsed:.2f}s | Tokens: {completion_tokens} | Velocidad: {tps:.1f} t/s")
                print("-" * 70)
                
                results.append({
                    "category": category,
                    "prompt": prompt,
                    "content": content,
                    "elapsed": elapsed,
                    "tokens": completion_tokens,
                    "tps": tps
                })
        except Exception as e:
            print(f"[{category}] Error: {e}")
            
    return results

if __name__ == "__main__":
    import sys
    label = sys.argv[1] if len(sys.argv) > 1 else "Modelo en Puerto 8080"
    run_benchmark(label)
