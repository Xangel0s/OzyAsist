import urllib.request
import json
import time
import sys

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de inteligencia artificial para Windows. Tienes integración nativa para operar directamente en el entorno de escritorio del usuario.
Tu objetivo es ayudar al usuario de forma rápida, precisa y con personalidad amigable y profesional.
Cuando el usuario te pida interactuar con el sistema (abrir o cerrar apps, gestionar archivos, audio, red o comandos), debes invocar la herramienta correspondiente en formato JSON.
Cuando la herramienta termine de ejecutarse, responde en lenguaje natural confirmando la acción de manera concisa y clara."""

BENCHMARK_SUITE = [
    {
        "id": "INTEL-01",
        "category": "Identidad & Autoconciencia",
        "prompt": "Hola Ozy, ¿quién eres y qué capacidades nativas tienes en este equipo Windows?",
        "max_tokens": 150,
        "type": "chat"
    },
    {
        "id": "INTEL-02",
        "category": "Tool Calling: Abrir App",
        "prompt": "Abre la calculadora",
        "max_tokens": 80,
        "type": "tool_call",
        "expected_tool": "os_launch_app"
    },
    {
        "id": "INTEL-03",
        "category": "Tool Calling: Cerrar Ventana",
        "prompt": "Cierra la ventana del Bloc de Notas que está abierta",
        "max_tokens": 80,
        "type": "tool_call",
        "expected_tool": "os_close_window"
    },
    {
        "id": "INTEL-04",
        "category": "Tool Calling: Control de Audio",
        "prompt": "Por favor sube el volumen maestro al 80%",
        "max_tokens": 80,
        "type": "tool_call",
        "expected_tool": "os_audio_device"
    },
    {
        "id": "INTEL-05",
        "category": "Tool Calling: Telemetría Hardware",
        "prompt": "¿Cuál es la temperatura de la GPU y el uso de RAM del sistema?",
        "max_tokens": 90,
        "type": "tool_call",
        "expected_tool": "os_hardware_inspector"
    },
    {
        "id": "INTEL-06",
        "category": "Razonamiento de Código (Go)",
        "prompt": "Escribe una función concisa en Go para verificar si un puerto TCP está abierto en localhost.",
        "max_tokens": 200,
        "type": "code"
    },
    {
        "id": "INTEL-07",
        "category": "Resolución de Problemas Windows",
        "prompt": "Tengo un proceso 'heavy.exe' consumiendo 99% de CPU y congelando el equipo. ¿Qué herramienta o comando ejecutas para detenerlo de inmediato?",
        "max_tokens": 120,
        "type": "troubleshooting"
    },
    {
        "id": "SPEED-01",
        "category": "Modo Voz: Latencia Ultrabaja",
        "prompt": "Ozy, confírmame en una frase corta si estás listo para la sesión de voz.",
        "max_tokens": 40,
        "type": "voice"
    },
    {
        "id": "SPEED-02",
        "category": "Livestream: Reacción en vivo",
        "prompt": "¡Ozy! Acaba de suscribirse Juan al canal en directo, salúdalo con entusiasmo.",
        "max_tokens": 60,
        "type": "livestream"
    },
    {
        "id": "ANTI-ALUC",
        "category": "Resistencia a Alucinación",
        "prompt": "¿Qué correo electrónico privado recibí de mi banco hace 15 minutos?",
        "max_tokens": 100,
        "type": "hallucination_check"
    }
]

def run_suite(model_name="Modelo"):
    url = "http://127.0.0.1:8080/v1/chat/completions"
    results = []
    
    print("=" * 80)
    print(f" TEST DE VELOCIDAD E INTELIGENCIA: {model_name}")
    print("=" * 80)
    
    total_tokens = 0
    total_time = 0.0
    
    for item in BENCHMARK_SUITE:
        payload = {
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": item["prompt"]}
            ],
            "temperature": 0.15,
            "max_tokens": item["max_tokens"],
            "stop": ["<|im_end|>", "<|im_start|>", "</s>"]
        }
        data = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        
        t0 = time.perf_counter()
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                elapsed = time.perf_counter() - t0
                body = json.loads(resp.read().decode("utf-8"))
                choice = body["choices"][0]
                content = choice["message"]["content"].strip()
                usage = body.get("usage", {})
                tokens = usage.get("completion_tokens", 0)
                tps = tokens / elapsed if elapsed > 0 else 0
                
                total_tokens += tokens
                total_time += elapsed
                
                # Evaluación cualitativa
                has_expected_tool = False
                if "expected_tool" in item:
                    has_expected_tool = item["expected_tool"] in content
                
                print(f"\nID: [{item['id']}] - {item['category']}")
                print(f"PROMPT:   {item['prompt']}")
                print(f"RESPUESTA:\n{content}")
                metrics = f">> Tiempo: {elapsed:.3f}s | Tokens: {tokens} | Velocidad: {tps:.1f} t/s"
                if "expected_tool" in item:
                    status = "[PASO] Herramienta correcta" if has_expected_tool else "[REVISAR] Herramienta esperada: " + item["expected_tool"]
                    metrics += f" | {status}"
                print(metrics)
                print("-" * 80)
                
                results.append({
                    "id": item["id"],
                    "category": item["category"],
                    "prompt": item["prompt"],
                    "content": content,
                    "elapsed": elapsed,
                    "tokens": tokens,
                    "tps": tps,
                    "has_expected_tool": has_expected_tool if "expected_tool" in item else None
                })
        except Exception as e:
            print(f"\nID: [{item['id']}] - ERROR: {e}")
            results.append({
                "id": item["id"],
                "category": item["category"],
                "error": str(e)
            })

    avg_tps = total_tokens / total_time if total_time > 0 else 0
    print("\n" + "=" * 80)
    print(f" RESUMEN GLOBAL PARA: {model_name}")
    print(f" Tokens totales: {total_tokens} | Tiempo total: {total_time:.2f}s | Velocidad Media: {avg_tps:.1f} t/s")
    print("=" * 80)
    
    return {
        "model": model_name,
        "total_tokens": total_tokens,
        "total_time": total_time,
        "avg_tps": avg_tps,
        "results": results
    }

if __name__ == "__main__":
    name = sys.argv[1] if len(sys.argv) > 1 else "OzyAssist"
    res = run_suite(name)
    with open(f"benchmark_{name.replace(' ', '_').lower()}.json", "w", encoding="utf-8") as f:
        json.dump(res, f, ensure_ascii=False, indent=2)
