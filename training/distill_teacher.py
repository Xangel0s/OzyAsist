import os
import json
import random
import urllib.request
import urllib.error

SYSTEM_PROMPT = """Eres OzyAssist, un asistente autónomo de escritorio y cowork para Windows de alto rendimiento.
Cuentas con herramientas nativas para interactuar directamente con el sistema operativo del usuario.
Directrices:
1. Analiza siempre la situación antes de actuar dentro de etiquetas <thought>...</thought>, revisando el historial y verificando qué aplicaciones realmente necesitan ser modificadas.
2. Si el usuario pide realizar una o varias acciones en el sistema (abrir o cerrar apps, audio, ventanas, archivos, telemetría), invoca las herramientas en formato JSON dentro de <tool_call>{"name": "...", "arguments": {...}}</tool_call>. Puedes invocar múltiples herramientas consecutivas si la orden lo amerita.
3. Nunca alucines que una aplicación ya está abierta si el usuario te está pidiendo abrirla ahora.
4. Para interacciones de voz y chat, responde de manera concisa, fluida, enérgica y sin símbolos pesados de markdown."""

TEACHER_URL = "http://127.0.0.1:8080/v1/chat/completions"

def query_teacher(prompt, max_tokens=250):
    try:
        data = {
            "model": "OzyAssist-7B-v3",
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": prompt}
            ],
            "max_tokens": max_tokens,
            "temperature": 0.3
        }
        req = urllib.request.Request(
            TEACHER_URL,
            data=json.dumps(data).encode("utf-8"),
            headers={"Content-Type": "application/json"}
        )
        with urllib.request.urlopen(req, timeout=30) as resp:
            res = json.loads(resp.read().decode("utf-8"))
            return res["choices"][0]["message"]["content"]
    except Exception as e:
        print(f"Error consultando Teacher: {e}")
        return None

def build_distilled_dataset():
    dataset = []
    print("=== FASE 1: GENERANDO DATASET DESTILADO CON CADENAS DE PENSAMIENTO (CoT) ===")

    # 1. Multi-App Execution (Abre X e Y / Cierra X e Y)
    multi_app_pairs = [
        (("calculadora", "calc"), ("bloc de notas", "notepad")),
        (("google chrome", "chrome"), ("visual studio code", "code")),
        (("spotify", "spotify"), ("administrador de tareas", "taskmgr")),
        (("terminal", "wt"), ("explorador de archivos", "explorer")),
        (("bloc de notas", "notepad"), ("calculadora", "calc")),
        (("discord", "discord"), ("roblox", "roblox")),
    ]

    for (name1, exe1), (name2, exe2) in multi_app_pairs:
        # Abrir ambas
        thought = f"<thought>El usuario solicita abrir dos aplicaciones al mismo tiempo: {name1} y {name2}. Invoco secuencialmente os_launch_app para ambas herramientas.</thought>"
        tools = f'<tool_call>{{"name": "os_launch_app", "arguments": {{"target": "{exe1}"}}}}</tool_call>\n<tool_call>{{"name": "os_launch_app", "arguments": {{"target": "{exe2}"}}}}</tool_call>'
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"Abre la {name1} y el {name2}"},
                {"role": "assistant", "content": f"{thought}\n{tools}"},
                {"role": "tool", "content": f"Aplicaciones {exe1} y {exe2} iniciadas exitosamente."},
                {"role": "assistant", "content": f"Listo, acabo de abrir {name1} y {name2} en tu pantalla."}
            ]
        })
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"Por favor inicia {name1} y también {name2}"},
                {"role": "assistant", "content": f"<thought>Ejecutaré apertura simultánea de {name1} ({exe1}) y {name2} ({exe2}).</thought>\n{tools}"},
                {"role": "tool", "content": f"Ambos programas iniciados."},
                {"role": "assistant", "content": f"He iniciado {name1} y {name2}."}
            ]
        })

        # Cerrar ambas
        win1 = name1.capitalize()
        win2 = name2.capitalize()
        thought_close = f"<thought>El usuario solicita cerrar dos aplicaciones: {name1} y {name2}. Usaré os_close_window para ambas ventanas específicas.</thought>"
        tools_close = f'<tool_call>{{"name": "os_close_window", "arguments": {{"target": "{win1}"}}}}</tool_call>\n<tool_call>{{"name": "os_close_window", "arguments": {{"target": "{win2}"}}}}</tool_call>'
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"Cierra la {name1} y el {name2}"},
                {"role": "assistant", "content": f"{thought_close}\n{tools_close}"},
                {"role": "tool", "content": f"Ventanas {win1} y {win2} cerradas exitosamente."},
                {"role": "assistant", "content": f"He cerrado tanto {name1} como {name2}."}
            ]
        })

    # 2. Conversaciones Multi-Turno con Estado Continuo (3 a 5 turnos cada una)
    # Escenario A: Calc -> Notepad -> Memoria -> Cierre parcial -> App arbitraria (Roblox)
    dataset.append({
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": "Hola Ozy, abre la calculadora"},
            {"role": "assistant", "content": '<thought>El usuario saluda y pide abrir la calculadora. Llamaré a os_launch_app con calc.</thought>\n<tool_call>{"name": "os_launch_app", "arguments": {"target": "calc"}}</tool_call>'},
            {"role": "tool", "content": "Aplicación calc iniciada exitosamente."},
            {"role": "assistant", "content": "¡Hola! He abierto la calculadora."},
            {"role": "user", "content": "Ahora abre el bloc de notas"},
            {"role": "assistant", "content": '<thought>En el turno previo abrí la calculadora. Ahora el usuario quiere abrir el bloc de notas. Procedo con os_launch_app para notepad.</thought>\n<tool_call>{"name": "os_launch_app", "arguments": {"target": "notepad"}}</tool_call>'},
            {"role": "tool", "content": "Aplicación notepad iniciada exitosamente."},
            {"role": "assistant", "content": "Listo, bloc de notas abierto también."},
            {"role": "user", "content": "¿Qué aplicaciones hemos abierto hasta ahora?"},
            {"role": "assistant", "content": "<thought>El usuario pregunta por los programas abiertos en esta sesión. Revisando el historial: abrimos la calculadora y luego el bloc de notas.</thought>\nHemos abierto la Calculadora y el Bloc de notas."},
            {"role": "user", "content": "Cierra la calculadora pero deja abierto el bloc de notas"},
            {"role": "assistant", "content": '<thought>El usuario solicita cerrar exclusivamente la calculadora. El bloc de notas debe mantenerse intacto. Usaré os_close_window con Calculadora.</thought>\n<tool_call>{"name": "os_close_window", "arguments": {"target": "Calculadora"}}</tool_call>'},
            {"role": "tool", "content": "Ventana Calculadora cerrada."},
            {"role": "assistant", "content": "He cerrado la Calculadora. El Bloc de notas sigue abierto."},
            {"role": "user", "content": "Abre roblox"},
            {"role": "assistant", "content": '<thought>El usuario solicita abrir una aplicación externa: roblox. No está abierta actualmente. Ejecuto os_launch_app con roblox.</thought>\n<tool_call>{"name": "os_launch_app", "arguments": {"target": "roblox"}}</tool_call>'},
            {"role": "tool", "content": "Aplicación roblox iniciada exitosamente."},
            {"role": "assistant", "content": "Iniciando Roblox en tu equipo."}
        ]
    })

    # Escenario B: Chrome -> Audio -> Cierre Chrome -> YouTube en Edge
    dataset.append({
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": "Abre Google Chrome"},
            {"role": "assistant", "content": '<thought>El usuario solicita iniciar el navegador Google Chrome. Usaré os_launch_app con chrome.</thought>\n<tool_call>{"name": "os_launch_app", "arguments": {"target": "chrome"}}</tool_call>'},
            {"role": "tool", "content": "Aplicación chrome iniciada exitosamente."},
            {"role": "assistant", "content": "Google Chrome está abierto."},
            {"role": "user", "content": "Sube el volumen al 70%"},
            {"role": "assistant", "content": '<thought>El usuario quiere ajustar el volumen maestro al 70%. Usaré os_audio_device.</thought>\n<tool_call>{"name": "os_audio_device", "arguments": {"action": "set_volume", "volume": 70}}</tool_call>'},
            {"role": "tool", "content": "Volumen establecido en 70%."},
            {"role": "assistant", "content": "Volumen ajustado al 70%."},
            {"role": "user", "content": "Cierra Chrome"},
            {"role": "assistant", "content": '<thought>El usuario pide cerrar Google Chrome. Usaré os_close_window con Chrome.</thought>\n<tool_call>{"name": "os_close_window", "arguments": {"target": "Google Chrome"}}</tool_call>'},
            {"role": "tool", "content": "Ventana Google Chrome cerrada."},
            {"role": "assistant", "content": "Google Chrome ha sido cerrado."},
            {"role": "user", "content": "Abre spotify"},
            {"role": "assistant", "content": '<thought>El usuario solicita abrir Spotify. No está en ejecución. Uso os_launch_app con spotify.</thought>\n<tool_call>{"name": "os_launch_app", "arguments": {"target": "spotify"}}</tool_call>'},
            {"role": "tool", "content": "Aplicación spotify iniciada exitosamente."},
            {"role": "assistant", "content": "Listo, abriendo Spotify."}
        ]
    })

    # Escenario C: Telemetría de hardware -> Limpieza de temporales -> Notificación
    dataset.append({
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": "¿Cómo están los recursos del equipo?"},
            {"role": "assistant", "content": '<thought>Consultando telemetría de hardware (CPU, RAM, GPU).</thought>\n<tool_call>{"name": "os_hardware_inspector", "arguments": {"action": "telemetry"}}</tool_call>'},
            {"role": "tool", "content": "{\"cpu_load\": 12, \"ram_used_gb\": 14.2, \"gpu_temp\": 47, \"gpu_vram_free_mb\": 3200}"},
            {"role": "assistant", "content": "Tu equipo está en excelentes condiciones: CPU al 12%, RAM con 14.2 GB en uso y GPU a 47°C con 3.2 GB de VRAM libre."},
            {"role": "user", "content": "Limpia los temporales"},
            {"role": "assistant", "content": '<thought>El usuario solicita purga de archivos temporales. Ejecutaré os_disk_cleaner.</thought>\n<tool_call>{"name": "os_disk_cleaner", "arguments": {"action": "clean"}}</tool_call>'},
            {"role": "tool", "content": "{\"cleaned_mb\": 850, \"files_removed\": 230}"},
            {"role": "assistant", "content": "He limpiado 230 archivos temporales, liberando 850 MB de almacenamiento."},
            {"role": "user", "content": "Mándame una notificación al escritorio"},
            {"role": "assistant", "content": '<thought>El usuario solicita una notificación Toast en Windows. Uso os_toast_notify.</thought>\n<tool_call>{"name": "os_toast_notify", "arguments": {"title": "OzyAssist", "message": "Sistema optimizado y recursos estables."}}</tool_call>'},
            {"role": "tool", "content": "Notificación emitida."},
            {"role": "assistant", "content": "Notificación enviada al centro de notificaciones de Windows."}
        ]
    })

    # 3. Soporte para Apps Arbitrarias y Juegos
    arbitrary_apps = [
        ("roblox", "roblox", "Roblox"),
        ("discord", "discord", "Discord"),
        ("steam", "steam", "Steam"),
        ("blender", "blender", "Blender"),
        ("visual studio code", "code", "Visual Studio Code"),
        ("spotify", "spotify", "Spotify"),
        ("obs studio", "obs64", "OBS Studio"),
        ("telegram", "telegram", "Telegram"),
        ("photoshop", "photoshop", "Adobe Photoshop"),
        ("sublime text", "sublime_text", "Sublime Text")
    ]

    for name, exe, title in arbitrary_apps:
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"Abre {name}"},
                {"role": "assistant", "content": f'<thought>El usuario solicita iniciar {name}. Invoco os_launch_app con el ejecutable o comando correspondiente ({exe}).</thought>\n<tool_call>{{"name": "os_launch_app", "arguments": {{"target": "{exe}"}}}}</tool_call>'},
                {"role": "tool", "content": f"Aplicación {exe} iniciada exitosamente."},
                {"role": "assistant", "content": f"Listo, abriendo {name.capitalize()}."}
            ]
        })
        dataset.append({
            "messages": [
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": f"Cierra {name}"},
                {"role": "assistant", "content": f'<thought>El usuario solicita cerrar {name}. Invoco os_close_window buscando la ventana {title}.</thought>\n<tool_call>{{"name": "os_close_window", "arguments": {{"target": "{title}"}}}}</tool_call>'},
                {"role": "tool", "content": f"Ventana {title} cerrada exitosamente."},
                {"role": "assistant", "content": f"He cerrado la ventana de {name.capitalize()}."}
            ]
        })

    # 4. Modo Voz Oral y Livestream
    voice_dialogues = [
        ("Hola Ozy, ¿me escuchas?", "Te escucho perfecto. ¿Qué necesitas que hagamos?"),
        ("Estamos en vivo en el stream", "¡Saludos a todos los espectadores del stream! OzyAssist listo para asistir en tiempo real."),
        ("Ozy saluda al chat", "¡Hola a toda la gente del chat! Bienvenidos a la transmisión."),
        ("Dime un dato curioso sobre computación", "El primer error informático real fue una polilla atrapada en un relé de la computadora Mark Two en 1947. De ahí nació el término bug."),
        ("¿Quién te creó?", "Fui desarrollado como OzyAssist, un asistente autónomo y cowork de alto rendimiento para Windows."),
        ("Ozy, gracias por la ayuda", "¡Con mucho gusto! Aquí sigo pendiente si requieres otra cosa."),
        ("Pon la música en pausa", '<thought>Pausando música vía multimedia.</thought>\n<tool_call>{"name": "os_media_control", "arguments": {"action": "play_pause"}}</tool_call>'),
        ("Pasa a la siguiente canción", '<thought>Siguiente pista vía multimedia.</thought>\n<tool_call>{"name": "os_media_control", "arguments": {"action": "next"}}</tool_call>')
    ]

    for user_q, bot_a in voice_dialogues:
        if "<tool_call>" in bot_a:
            dataset.append({
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": user_q},
                    {"role": "assistant", "content": bot_a},
                    {"role": "tool", "content": "Comando ejecutado exitosamente."},
                    {"role": "assistant", "content": "Listo, acción completada."}
                ]
            })
        else:
            dataset.append({
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": user_q},
                    {"role": "assistant", "content": f"<thought>El usuario se comunica por voz de manera directa. Respondo con naturalidad, calidez y concisión.</thought>\n{bot_a}"}
                ]
            })

    # 5. Cargar y enriquecer con los casos existentes de dataset_v3 (audio, tiles, find_files)
    script_dir = os.path.dirname(os.path.abspath(__file__))
    v3_path = os.path.join(script_dir, "dataset", "ozy_dataset_v3.jsonl")
    if os.path.exists(v3_path):
        with open(v3_path, "r", encoding="utf-8") as f:
            for line in f:
                item = json.loads(line.strip())
                # Asegurar que el assistant tenga <thought>
                messages = item.get("messages", [])
                for m in messages:
                    if m["role"] == "assistant" and not m["content"].startswith("<thought>") and not m["content"].startswith("¡"):
                        if "<tool_call>" in m["content"]:
                            m["content"] = "<thought>Analizando instrucción del sistema para ejecutar la herramienta adecuada.</thought>\n" + m["content"]
                dataset.append(item)

    # 6. Consultar al Teacher 7B para 5 casos reales en vivo si está disponible
    print("\n--- Consultando al Teacher 7B en vivo para destilación de CoT complejos ---")
    teacher_prompts = [
        "Abre la calculadora y el bloc de notas al mismo tiempo",
        "El usuario tenía abierto chrome y ahora te pide abrir discord y cerrar chrome. ¿Qué haces?",
        "Abre roblox y dime cómo está la temperatura de la GPU",
        "Silencia el volumen y minimiza todas las ventanas",
        "Busca archivos PDF en la carpeta de Documentos y muéstramelos",
    ]

    for p in teacher_prompts:
        resp = query_teacher(p)
        if resp:
            print(f"[TEACHER OK] '{p[:30]}...' -> Generado CoT de {len(resp)} caracteres")
            dataset.append({
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": p},
                    {"role": "assistant", "content": resp}
                ]
            })

    # Mezclar y guardar
    random.shuffle(dataset)
    out_path = os.path.join(script_dir, "dataset", "ozy_distilled_v4.jsonl")
    with open(out_path, "w", encoding="utf-8") as f:
        for entry in dataset:
            f.write(json.dumps(entry, ensure_ascii=False) + "\n")

    print(f"\n[OK] ¡Dataset v4 destilado generado exitosamente con {len(dataset)} ejemplos en: {out_path}!")

if __name__ == "__main__":
    build_distilled_dataset()
