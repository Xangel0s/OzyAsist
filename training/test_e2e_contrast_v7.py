import urllib.request
import json
import time

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario actual: User (Ruta: C:\\Users\\User)
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.

DIRECTRICES:
1. Si el usuario te saluda ("hola", "buenos días") o hace preguntas conceptuales, responde de manera amigable, concisa y en lenguaje natural sin invocar herramientas.
2. Si el usuario solicita una acción (abrir/cerrar apps, gestionar servicios de Windows 11, buscar archivos o investigar en la web), analiza la situación en <thought>...</thought> y emite inmediatamente la herramienta correspondiente en formato JSON dentro de:
<tool_call>{"name": "nombre_herramienta", "arguments": {...}}</tool_call>
3. Para investigaciones temáticas ("investiga sobre Beck...", "haz un informe de..."), ejecuta PRIMERO 'web_search' para recopilar datos reales antes de invocar 'os_create_pdf' o 'os_create_docx'.
4. En Windows 11, los servicios del sistema en segundo plano usan 'os_service_manager'. Las aplicaciones de usuario usan 'os_launch_app' para abrir y 'os_close_window' para cerrar. Proporciona siempre rutas completas."""

def query_brain(messages, temp=0.1, max_tokens=150):
    payload = {
        "model": "OzyAssist-3B-v7",
        "messages": messages,
        "temperature": temp,
        "max_tokens": max_tokens,
        "stream": False
    }
    req = urllib.request.Request(
        "http://127.0.0.1:8080/v1/chat/completions",
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req) as resp:
        data = json.loads(resp.read().decode("utf-8"))
        return data["choices"][0]["message"]["content"]

def run_tests():
    print("=" * 70)
    print("      EVALUACIÓN E2E DE ALTO NIVEL: OzyAssist-3B-v7 NATIVO")
    print("=" * 70)

    # Test 1: Saludos + Apertura
    print("\n[TEST 1] Saludo + Apertura de Calculadora")
    res1 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Hola abre la calculadora"}
    ])
    print(f"Respuesta:\n{res1}")
    assert "os_launch_app" in res1 and "calc" in res1, "Falló apertura de calculadora"
    print("-> PASÓ [OK]")

    # Test 2: Saludos + Cierre
    print("\n[TEST 2] Saludo + Cierre de Calculadora")
    res2 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Hola cierra la calculadora"}
    ])
    print(f"Respuesta:\n{res2}")
    assert "os_close_window" in res2, "Falló cierre de calculadora"
    print("-> PASÓ [OK]")

    # Test 3: Windows 11 Servicios (SCM)
    print("\n[TEST 3] Gestión de Servicios de Windows 11 (spooler)")
    res3 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Por favor reinicia el servicio de cola de impresión"}
    ])
    print(f"Respuesta:\n{res3}")
    assert "os_service_manager" in res3 and "spooler" in res3, "Falló gestión de servicio"
    print("-> PASÓ [OK]")

    # Test 4: Ejemplo Negativo (Chitchat puro sin herramientas)
    print("\n[TEST 4] Ejemplo Negativo (Saludo puro sin invocar herramientas)")
    res4 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Hola Ozy, ¿cómo estás hoy?"}
    ])
    print(f"Respuesta:\n{res4}")
    assert "<tool_call>" not in res4 and "os_" not in res4, "Alucinó herramienta en chitchat"
    print("-> PASÓ [OK]")

    # Test 5: ReAct Multi-Turno (Investigación Web -> PDF)
    print("\n[TEST 5] Bucle ReAct: Turno 1 (web_search) -> Turno 2 (os_create_pdf)")
    turn1 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Investiga sobre Beck y su álbum Colors y genera un informe en mis documentos"}
    ])
    print(f"Turno 1:\n{turn1}")
    assert "web_search" in turn1 and "beck" in turn1.lower(), "Falló Turno 1 web_search"
    print("-> Turno 1 PASÓ [OK] (Emite web_search)")

    turn2 = query_brain([
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": "Investiga sobre Beck y su álbum Colors y genera un informe en mis documentos"},
        {"role": "assistant", "content": turn1},
        {"role": "tool", "content": "Colors es el decimotercer álbum de Beck lanzado el 13 de octubre de 2017 por Capitol Records. Ganó 2 premios Grammy."},
    ])
    print(f"Turno 2:\n{turn2}")
    assert "os_create_pdf" in turn2 or "os_create_docx" in turn2, "Falló Turno 2 generación documental"
    print("-> Turno 2 PASÓ [OK] (Emite generación documental con datos reales)")

    print("\n" + "=" * 70)
    print("      ¡TODOS LOS TESTS DE INTELIGENCIA NATIVA PASARON AL 100%!")
    print("=" * 70)

if __name__ == "__main__":
    run_tests()
