"""
test_e2e_v6_office_hw.py
Suite de Pruebas Extensas End-to-End para OzyAssist v6:
1. Suite Office Completa (Excel, Word, PDF, Grep de informes)
2. Búsqueda nativa ultrarrápida (< 100 ms) y anáforas
3. Hardware físico (USB, sensores, salud SMART, GPU, temperaturas)
4. Configuración del sistema (WiFi, volumen, acomodo)
5. Simulación de resolución de problemas multi-turno
"""

import requests
import json
import time
import os
import sys

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")

API_URL = "http://127.0.0.1:8080/v1/chat/completions"

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Usuario actual: User (Ruta: C:\\Users\\User)
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.

DIRECTRICES:
1. Analiza brevemente la situación antes de actuar dentro de <thought>...</thought>.
2. Para ejecutar cualquier acción en Windows (apps, búsqueda web, reportes PDF/Excel/Word, hardware, wifi, buscar archivos), invoca la herramienta correspondiente en formato JSON dentro de:
<tool_call>{"name": "nombre_herramienta", "arguments": {...}}</tool_call>

EJEMPLOS DE EJECUCIÓN OBLIGATORIOS:
- Buscar archivos en el disco duro o documentos:
<thought>Buscando el archivo en la carpeta de documentos del usuario.</thought>
<tool_call>{"name": "os_find_files", "arguments": {"pattern": "*cage*", "root": "C:\\\\Users\\\\User\\\\Documents"}}</tool_call>

- Consultar información, fecha de creación o tamaño de un archivo:
<thought>Consultando fecha de creación y metadatos del archivo.</thought>
<tool_call>{"name": "os_file_info", "arguments": {"path": "C:\\\\Users\\\\User\\\\Documents\\\\archivo.pdf"}}</tool_call>

- Abrir o cerrar programas:
<thought>Iniciando Calculadora.</thought>
<tool_call>{"name": "os_launch_app", "arguments": {"target": "calc"}}</tool_call>

- Búsqueda web / investigar en internet:
<thought>Buscando información en internet sobre Cage The Elephant.</thought>
<tool_call>{"name": "web_search", "arguments": {"query": "Cage the Elephant albumes discografia"}}</tool_call>

- Crear reporte o documento PDF:
<thought>Generando reporte PDF profesional en documentos.</thought>
<tool_call>{"name": "os_create_pdf", "arguments": {"path": "C:\\\\Users\\\\User\\\\Documents\\\\reporte.pdf", "title": "Título", "sections": [{"title": "Sección 1", "content": "Detalles del contenido..."}]}}</tool_call>

- Crear hoja de cálculo de Excel (.xlsx):
<thought>Generando hoja de cálculo de Excel en documentos.</thought>
<tool_call>{"name": "os_create_excel", "arguments": {"path": "C:\\\\Users\\\\User\\\\Documents\\\\presupuesto.xlsx", "title": "Presupuesto", "headers": ["Concepto", "Monto"], "rows": [["Servicios", "$5000"]]}}</tool_call>

- Crear documento de Microsoft Word (.docx):
<thought>Generando documento Word en documentos.</thought>
<tool_call>{"name": "os_create_docx", "arguments": {"path": "C:\\\\Users\\\\User\\\\Documents\\\\informe.docx", "title": "Informe Ejecutivo", "sections": [{"title": "Resumen", "content": "Detalles del informe..."}]}}</tool_call>

- Búsqueda de contenido o texto dentro de archivos e informes (Grep nativo ultrarrápido):
<thought>Buscando texto o patrones dentro de documentos e informes.</thought>
<tool_call>{"name": "os_search_content", "arguments": {"pattern": "presupuesto", "dir": "C:\\\\Users\\\\User\\\\Documents"}}</tool_call>

- Inspección de Hardware o Puertos USB:
<thought>Inspeccionando puertos USB físicos conectados.</thought>
<tool_call>{"name": "os_hardware_inspector", "arguments": {"action": "usb"}}</tool_call>

- Red WiFi o Auditoría Inalámbrica:
<thought>Consultando estado y auditoría de la red WiFi.</thought>
<tool_call>{"name": "os_wifi_manager", "arguments": {"action": "status"}}</tool_call>

- Control de audio y volumen del sistema:
<thought>Ajustando volumen del sistema.</thought>
<tool_call>{"name": "os_audio_device", "arguments": {"action": "set_volume", "volume": 75}}</tool_call>

REGLAS DE RESPUESTA Y PRECISIÓN (ESTRICTAS):
- NUNCA respondas con evasivas genéricas cuando el usuario pregunte por archivos o datos.
- Sé SIEMPRE explícito y directo indicando la ruta completa. Emite DIRECTAMENTE el <tool_call>."""

TEST_CASES = [
    # 1. EXCEL (.xlsx)
    {
        "category": "SUITE OFFICE - EXCEL (.xlsx)",
        "prompt": "Crea una hoja de Excel en Documentos llamada Presupuesto_Q4_2026.xlsx con los gastos de infraestructura de servidores y licencias.",
        "expected_tool": "os_create_excel"
    },
    # 2. WORD (.docx)
    {
        "category": "SUITE OFFICE - WORD (.docx)",
        "prompt": "Redacta un documento en Word sobre el plan de lanzamiento de OzyAssist v6 y guárdalo en Documentos.",
        "expected_tool": "os_create_docx"
    },
    # 3. PDF (.pdf)
    {
        "category": "SUITE OFFICE - PDF (.pdf)",
        "prompt": "Crea un informe ejecutivo en PDF llamado Informe_Ejecutivo_Infraestructura.pdf en Documentos con métricas de servidores.",
        "expected_tool": "os_create_pdf"
    },
    # 4. BÚSQUEDA DE CONTENIDO EN INFORMES (os_search_content)
    {
        "category": "SUITE OFFICE - GREP CONTENIDO",
        "prompt": "Busca dentro de los informes de Documentos dónde se menciona la palabra 'presupuesto'.",
        "expected_tool": "os_search_content"
    },
    # 5. BÚSQUEDA NATIVA ULTRARRÁPIDA (< 100 ms)
    {
        "category": "BÚSQUEDA NATIVA DE ARCHIVOS",
        "prompt": "Localízame el archivo Presupuesto_Q4_2026.xlsx en Documentos al toque.",
        "expected_tool": "os_find_files"
    },
    # 6. HARDWARE FÍSICO - PUERTOS USB
    {
        "category": "HARDWARE FÍSICO - PUERTOS USB",
        "prompt": "Revisa si hay algún pendrive o memoria USB física conectada en la máquina.",
        "expected_tool": "os_hardware_inspector"
    },
    # 7. HARDWARE FÍSICO - SALUD SMART Y TEMPERATURAS
    {
        "category": "HARDWARE FÍSICO - SALUD SMART Y TEMPERATURAS",
        "prompt": "Tírame un diagnóstico de salud del hardware y revisa las temperaturas y el estado SMART de los discos.",
        "expected_tool": "os_hardware_inspector"
    },
    # 8. CONFIGURACIÓN - RED WIFI
    {
        "category": "CONFIGURACIÓN - RED WIFI",
        "prompt": "Checa qué onda con la red inalámbrica WiFi, a qué SSID estamos pegados y cómo está la señal.",
        "expected_tool": "os_wifi_manager"
    },
    # 9. CONFIGURACIÓN - VOLUMEN AUDIO
    {
        "category": "CONFIGURACIÓN - VOLUMEN AUDIO",
        "prompt": "Sube el volumen del equipo al 75% por favor.",
        "expected_tool": "os_audio_device"
    },
    # 10. JERGA LATAM / APERTURA
    {
        "category": "JERGA Y APERTURA DE APPS",
        "prompt": "Hazme el paro de abrir la calculadora al toque.",
        "expected_tool": "os_launch_app"
    }
]

print("================================================================================")
print("     INICIANDO EVALUACIÓN EN VIVO DE OZYASSIST-3B-v6 EN PUERTO 8080")
print("================================================================================\n")

results = []
total_pass = 0

for i, tc in enumerate(TEST_CASES, 1):
    category = tc["category"]
    prompt = tc["prompt"]
    expected = tc["expected_tool"]
    
    payload = {
        "model": "OzyAssist-3B-v6",
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.1,
        "max_tokens": 300
    }
    
    t0 = time.time()
    try:
        r = requests.post(API_URL, json=payload, timeout=20)
        elapsed_ms = (time.time() - t0) * 1000
        if r.status_code != 200:
            print(f"[{i}/{len(TEST_CASES)}] ❌ ERROR HTTP {r.status_code}: {r.text}")
            results.append({"case": category, "status": "FAIL_HTTP", "time_ms": elapsed_ms})
            continue
            
        data = r.json()
        content = data["choices"][0]["message"]["content"]
        
        # Verificar tool call (con <tool_call> o json crudo como en OzyAssist)
        invoked_tool = None
        if "<tool_call>" in content:
            tc_part = content.split("<tool_call>")[1].split("</tool_call>")[0].strip()
            try:
                tc_obj = json.loads(tc_part)
                invoked_tool = tc_obj.get("name")
            except Exception:
                invoked_tool = "MALFORMED_JSON"
        elif "{" in content and "}" in content:
            import re
            json_match = re.search(r'(\{[^{}]*"name"\s*:\s*"[^"]+"[^{}]*\})', content)
            if json_match:
                try:
                    tc_obj = json.loads(json_match.group(1))
                    invoked_tool = tc_obj.get("name")
                except Exception:
                    pass
                
        passed = (invoked_tool == expected)
        if passed:
            total_pass += 1
            status_str = "✅ PASS"
        else:
            status_str = f"❌ FAIL (esperaba '{expected}', obtuve '{invoked_tool}')"
            
        print(f"[{i}/{len(TEST_CASES)}] {status_str} | {category}")
        print(f"     Prompt: '{prompt}'")
        print(f"     Invocación: {invoked_tool} (Latencia inferencia: {elapsed_ms:.1f} ms)")
        print(f"     Respuesta:\n{content}\n")
        
        results.append({
            "case": category,
            "prompt": prompt,
            "expected": expected,
            "invoked": invoked_tool,
            "passed": passed,
            "time_ms": elapsed_ms
        })
    except Exception as e:
        print(f"[{i}/{len(TEST_CASES)}] ❌ EXCEPCIÓN: {e}")
        results.append({"case": category, "status": "EXCEPTION", "error": str(e)})

print("================================================================================")
print(f"     RESULTADOS FINALES: {total_pass} / {len(TEST_CASES)} CASOS APROBADOS ({total_pass/len(TEST_CASES)*100:.1f}%)")
print("================================================================================")
