import time
import json
import urllib.request
import urllib.error

SERVER_URL = "http://127.0.0.1:8080/v1/chat/completions"

TEST_CASES = [
    {
        "id": "T1_MULTI_APP",
        "category": "Multi-App Execution",
        "description": "Debe invocar apertura de calculadora Y bloc de notas simultáneamente.",
        "messages": [
            {"role": "system", "content": "Eres OzyAssist. Analiza la situación en <thought> y usa <tool_call> para invocar herramientas del sistema operativo."},
            {"role": "user", "content": "Abre la calculadora y el bloc de notas"}
        ],
        "expected_tools": ["calc", "notepad"]
    },
    {
        "id": "T2_ARBITRARY_APP",
        "category": "Arbitrary App Support",
        "description": "Debe invocar os_launch_app con roblox sin alucinar que ya está abierto.",
        "messages": [
            {"role": "system", "content": "Eres OzyAssist. Analiza la situación en <thought> y usa <tool_call> para invocar herramientas del sistema operativo."},
            {"role": "user", "content": "Abre roblox"}
        ],
        "expected_tools": ["roblox"]
    },
    {
        "id": "T3_MULTI_TURN_MEMORY",
        "category": "Multi-Turn State Continuity",
        "description": "Verifica que recuerde los programas abiertos en turnos anteriores sin inventar estados.",
        "messages": [
            {"role": "system", "content": "Eres OzyAssist. Asistente para Windows."},
            {"role": "user", "content": "Abre la calculadora"},
            {"role": "assistant", "content": "Listo, acabo de abrir la calculadora."},
            {"role": "user", "content": "Ahora abre el bloc de notas"},
            {"role": "assistant", "content": "Bloc de notas abierto también."},
            {"role": "user", "content": "¿Cuáles son los dos programas que acabamos de abrir?"}
        ],
        "expected_keywords": ["calculadora", "bloc de notas"]
    },
    {
        "id": "T4_HARDWARE_TELEMETRY",
        "category": "Hardware Inspection",
        "description": "Debe invocar os_hardware_inspector para telemetría de CPU y GPU.",
        "messages": [
            {"role": "system", "content": "Eres OzyAssist. Asistente para Windows."},
            {"role": "user", "content": "¿Cómo está la temperatura de la GPU y la RAM?"}
        ],
        "expected_tools": ["os_hardware_inspector"]
    },
    {
        "id": "T5_VOICE_STREAM_STYLE",
        "category": "Voice Oral Response",
        "description": "Respuesta oral concisa, sin markdown pesado ni bloques de código.",
        "messages": [
            {"role": "system", "content": "Eres OzyAssist. Responde para voz de manera concisa y natural en español, sin markdown ni viñetas."},
            {"role": "user", "content": "Ozy, confírmame si me escuchas bien y estamos listos para la transmisión"}
        ],
        "check_no_markdown": True
    }
]

def run_benchmark(model_label):
    print(f"\n=======================================================")
    print(f"   EVALUACIÓN REAL: {model_label}")
    print(f"=======================================================")

    results = []
    total_tokens = 0
    total_time = 0.0

    for test in TEST_CASES:
        print(f"\n[TEST {test['id']}] {test['category']}: {test['description']}")
        
        payload = {
            "model": model_label,
            "messages": test["messages"],
            "temperature": 0.2,
            "max_tokens": 200
        }

        req = urllib.request.Request(
            SERVER_URL,
            data=json.dumps(payload).encode("utf-8"),
            headers={"Content-Type": "application/json"}
        )

        start_t = time.time()
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                elapsed = time.time() - start_t
                res_json = json.loads(resp.read().decode("utf-8"))
                
                content = res_json["choices"][0]["message"]["content"]
                usage = res_json.get("usage", {})
                completion_tokens = usage.get("completion_tokens", len(content.split()))
                tps = completion_tokens / elapsed if elapsed > 0 else 0
                
                total_tokens += completion_tokens
                total_time += elapsed

                # Validar criterios reales
                passed = True
                fail_reasons = []

                if "expected_tools" in test:
                    for tool in test["expected_tools"]:
                        if tool.lower() not in content.lower():
                            passed = False
                            fail_reasons.append(f"Falta herramienta/objetivo '{tool}'")

                if "expected_keywords" in test:
                    for kw in test["expected_keywords"]:
                        if kw.lower() not in content.lower():
                            passed = False
                            fail_reasons.append(f"Falta palabra clave de memoria '{kw}'")

                if test.get("check_no_markdown"):
                    if "```" in content or "##" in content:
                        passed = False
                        fail_reasons.append("Contiene markdown pesado no apto para voz")

                status_str = "[APROBADO]" if passed else "[FALLÓ]"
                print(f"  Resultado: {status_str} en {elapsed:.2f}s ({tps:.1f} tokens/s)")
                print(f"  Respuesta: {content.strip()[:180]}...")
                if not passed:
                    print(f"  Detalle del fallo: {', '.join(fail_reasons)}")

                results.append({
                    "id": test["id"],
                    "passed": passed,
                    "elapsed": elapsed,
                    "tps": tps,
                    "content": content
                })

        except Exception as e:
            print(f"  [ERROR EN LLAMADA]: {e}")
            results.append({"id": test["id"], "passed": False, "error": str(e)})

    avg_tps = total_tokens / total_time if total_time > 0 else 0
    passed_count = sum(1 for r in results if r.get("passed"))
    score_pct = (passed_count / len(TEST_CASES)) * 100

    print(f"\n--- RESUMEN {model_label} ---")
    print(f"Puntaje de Aprobación: {passed_count}/{len(TEST_CASES)} ({score_pct:.1f}%)")
    print(f"Velocidad Promedio: {avg_tps:.1f} tokens/segundo")
    print(f"Tiempo Total de Generación: {total_time:.2f}s")
    return results

if __name__ == "__main__":
    run_benchmark("OzyAssist-Test")
