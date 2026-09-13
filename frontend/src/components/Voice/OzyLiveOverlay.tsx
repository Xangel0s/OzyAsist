import React, { useEffect, useRef, useState, useCallback } from "react";
import { Mic, MicOff, Volume2, X, Sparkles, Radio, AlertTriangle } from "lucide-react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";

import { wsService } from "../../services/ws";
import { api } from "../../services/api";

type LiveState = "idle" | "listening" | "thinking" | "speaking";

export const OzyLiveOverlay: React.FC = () => {
  const isOpen = useUIStore((s) => s.voiceLiveOpen);
  const setIsOpen = useUIStore((s) => s.setVoiceLiveOpen);
  const openSettings = useUIStore((s) => s.openSettings);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const createChat = useChatStore((s) => s.createChat);
  const sendMessage = useChatStore((s) => s.sendMessage);

  const [state, setState] = useState<LiveState>("idle");
  const stateRef = useRef<LiveState>("idle"); // Mirror de state para closures sin re-render
  const [transcript, setTranscript] = useState<string>("");
  const [lastResponse, setLastResponse] = useState<string>("");
  const [isMuted, setIsMuted] = useState<boolean>(false);
  const [audioLevel, setAudioLevel] = useState<number>(0);
  const [hasConnectedModel, setHasConnectedModel] = useState<boolean>(true);
  const [hasCheckedModel, setHasCheckedModel] = useState<boolean>(false);

  // Sincronizar stateRef con state para callbacks que no pueden depender de state directamente
  useEffect(() => { stateRef.current = state; }, [state]);

  const recognitionRef = useRef<any>(null);
  const silenceTimerRef = useRef<any>(null);
  const watchdogTimerRef = useRef<any>(null);
  const synthRef = useRef<SpeechSynthesis | null>(null);
  const currentUtteranceRef = useRef<SpeechSynthesisUtterance | null>(null);
  const audioIntervalRef = useRef<any>(null);
  const hasGreetedRef = useRef<boolean>(false);
  const voicesRef = useRef<SpeechSynthesisVoice[]>([]);
  // Guard para evitar doble-disparo de handleAssistantResponse
  const isSpeakingResponseRef = useRef<boolean>(false);

  const speakTextRef = useRef<(text: string, onFinish?: () => void) => void>(() => {});
  const startListeningRef = useRef<() => void>(() => {});
  const stopSpeakingRef = useRef<() => void>(() => {});

  // Inicializar SpeechSynthesis y cargar voces de forma asíncrona
  useEffect(() => {
    if (typeof window !== "undefined" && "speechSynthesis" in window) {
      synthRef.current = window.speechSynthesis;
      // Las voces se cargan asíncronamente en Chrome/Edge
      const loadVoices = () => {
        voicesRef.current = window.speechSynthesis.getVoices();
      };
      loadVoices();
      window.speechSynthesis.onvoiceschanged = loadVoices;
    }
  }, []);

  // Simulación de oscilador visual de audio para el orbe
  useEffect(() => {
    if (state === "listening" || state === "speaking") {
      audioIntervalRef.current = setInterval(() => {
        const base = state === "speaking" ? 0.6 : 0.3;
        const randomFluctuation = Math.random() * 0.5;
        setAudioLevel(base + randomFluctuation);
      }, 100);
    } else {
      setAudioLevel(0.1);
      if (audioIntervalRef.current) {
        clearInterval(audioIntervalRef.current);
      }
    }
    return () => {
      if (audioIntervalRef.current) clearInterval(audioIntervalRef.current);
    };
  }, [state]);

  // Cancelar TTS
  const stopSpeaking = useCallback(() => {
    if (synthRef.current) {
      synthRef.current.cancel();
    }
    currentUtteranceRef.current = null;
  }, []);
  stopSpeakingRef.current = stopSpeaking;

  // Función para reproducir la voz de Ozy con TTS
  const speakText = useCallback(
    (text: string, onFinish?: () => void) => {
      if (!synthRef.current || isMuted) {
        setState("listening");
        onFinish?.();
        return;
      }

      stopSpeaking();

      // Limpiar texto de markdown y tags para que la lectura sea fluida
      const cleanText = text
        .replace(/<[^>]*>[\s\S]*?(?:<\/[^>]*>|$)/g, "") // Limpiar tags como <Sistema> o <thought> y su contenido
        .replace(/```[\s\S]*?```/g, "Código ejecutado.")
        .replace(/`([^`]+)`/g, "$1")
        .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
        .replace(/[#*_~>]/g, "")
        .replace(/\n+/g, " ")
        .trim();

      if (!cleanText) {
        setState("listening");
        onFinish?.();
        return;
      }

      const utterance = new SpeechSynthesisUtterance(cleanText);
      utterance.lang = "es-ES";
      utterance.rate = 1.05;
      utterance.pitch = 1.0;

      // Seleccionar la mejor voz española disponible (cargadas asíncronamente)
      const voices = voicesRef.current.length > 0
        ? voicesRef.current
        : window.speechSynthesis.getVoices();

      const spanishVoice =
        voices.find(
          (v) =>
            v.lang.startsWith("es") &&
            (v.name.includes("Google") ||
              v.name.includes("Natural") ||
              v.name.includes("Sabina") ||
              v.name.includes("Helena") ||
              v.name.includes("Pablo"))
        ) || voices.find((v) => v.lang.startsWith("es"));

      if (spanishVoice) {
        utterance.voice = spanishVoice;
      }

      utterance.onstart = () => {
        setState("speaking");
      };

      utterance.onend = () => {
        isSpeakingResponseRef.current = false;
        setState("listening");
        onFinish?.();
      };

      utterance.onerror = (e) => {
        // "interrupted" es normal cuando stopSpeaking() cancela el habla en curso
        if (e.error !== "interrupted") {
          console.warn("SpeechSynthesis error:", e.error);
        }
        isSpeakingResponseRef.current = false;
        setState("listening");
        onFinish?.();
      };

      currentUtteranceRef.current = utterance;

      // Chrome suspende speechSynthesis si el tab estuvo inactivo.
      // resume() antes de speak() garantiza que funcione.
      synthRef.current.resume();
      synthRef.current.speak(utterance);
    },
    [isMuted, stopSpeaking]
  );
  speakTextRef.current = speakText;

  // Iniciar reconocimiento de voz continuo
  const startListening = useCallback(() => {
    stopSpeakingRef.current();

    const SpeechRec =
      (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRec) return;

    if (recognitionRef.current) {
      try {
        recognitionRef.current.stop();
      } catch {}
      recognitionRef.current = null;
    }

    try {
      const rec = new SpeechRec();
      rec.lang = "es-ES";
      rec.continuous = true;
      rec.interimResults = true;

      rec.onstart = () => {
        if (!useChatStore.getState().isResponding && stateRef.current !== "speaking") {
          setState("listening");
        }
      };

      rec.onresult = (event: any) => {
        // No procesar si Ozy está hablando (evitar eco) o pensando (ya hay una solicitud en curso)
        // Usamos un ref auxiliar que refleja el estado actual sin causar re-renders en este callback
        const currentState = stateRef.current;
        if (currentState === "speaking" || currentState === "thinking") return;

        let fullTranscript = "";
        for (let i = event.resultIndex; i < event.results.length; ++i) {
          fullTranscript += event.results[i][0].transcript;
        }

        const trimmed = fullTranscript.trim();
        if (trimmed) {
          setTranscript(trimmed);

          // Voice Activity Detection: Debounce de 1 segundo tras silencio
          if (silenceTimerRef.current) {
            clearTimeout(silenceTimerRef.current);
          }

          silenceTimerRef.current = setTimeout(() => {
            if (trimmed.length > 1) {
              try {
                rec.stop();
              } catch {}
              handleDispatchMessage(trimmed);
            }
          }, 1100);
        }
      };

      rec.onerror = (e: any) => {
        if (e.error !== "no-speech") {
          console.debug("Voice recognition error:", e.error);
        }
      };

      rec.onend = () => {
        // Auto-reanudar si seguimos en el modal y no estamos hablando
        if (useUIStore.getState().voiceLiveOpen && recognitionRef.current) {
          setTimeout(() => {
            if (useUIStore.getState().voiceLiveOpen && recognitionRef.current) {
              try {
                rec.start();
              } catch {}
            }
          }, 250);
        }
      };

      recognitionRef.current = rec;
      rec.start();
    } catch (e) {
      console.debug("Error starting speech recognition:", e);
    }
  }, []);
  startListeningRef.current = startListening;

  // Enviar mensaje al backend
  const handleDispatchMessage = useCallback(
    async (text: string) => {
      // Limpiar watchdog anterior antes de todo
      if (watchdogTimerRef.current) {
        clearTimeout(watchdogTimerRef.current);
        watchdogTimerRef.current = null;
      }

      setState("thinking");
      setTranscript(text);

      // Watchdog de 90 segundos — los modelos locales (gemma, mistral) pueden tardar
      // hasta 60s en responder. En VoiceMode el backend usa un prompt ultra-corto
      // (~500 tokens) que acorta el tiempo de prefill de 14s a ~2s.
      // IMPORTANTE: el watchdog limpia isResponding vía cancelResponse antes de reiniciar.
      watchdogTimerRef.current = setTimeout(() => {
        console.warn("Watchdog timeout triggered in OzyLiveOverlay");
        // Cancelar la respuesta en curso en el store para desbloquear isResponding
        useChatStore.getState().cancelResponse?.();
        setState("listening");
        setTranscript("");
        watchdogTimerRef.current = null;
        speakTextRef.current("El modelo tardó demasiado en cargar. Te sigo escuchando.", () => {
          startListeningRef.current();
        });
      }, 180000);

      try {
        let chatId = activeChatId;
        if (!chatId) {
          chatId = await createChat("chat");
        }
        if (chatId) {
          // voiceMode=true activa el path de baja latencia en el backend:
          // system prompt de ~50 tokens y solo 8 tools OS (vs 18 tools + 2000 tokens)
          await sendMessage(chatId, text, true);
        }
      } catch (err: any) {
        console.error("Error sending voice message:", err);
        // Limpiar watchdog inmediatamente
        if (watchdogTimerRef.current) {
          clearTimeout(watchdogTimerRef.current);
          watchdogTimerRef.current = null;
        }
        setState("listening");
        setTranscript("");

        const isBusy = err?.message === "busy";
        if (isBusy) {
          // El modelo aún está procesando — no hablar, solo volver a escuchar
          console.info("OzyLive: modelo ocupado, esperando...");
          setTimeout(() => startListeningRef.current(), 500);
        } else {
          // Error real: WS caído, proveedor no disponible, etc.
          speakTextRef.current("Ocurrió un error de conexión. Verifica que el servidor esté activo.", () => {
            startListeningRef.current();
          });
        }
      }
    },
    [activeChatId, createChat, sendMessage]
  );


  // Escuchar respuestas de Ozy por Chat Store y WebSocket
  useEffect(() => {
    if (!isOpen) return;

    const handleAssistantResponse = (content?: string) => {
      // Guard de disparo único: evita que store + WS events llamen speakText
      // dos veces casi simultáneamente (el segundo interrumpe al primero).
      if (isSpeakingResponseRef.current) return;
      isSpeakingResponseRef.current = true;

      if (watchdogTimerRef.current) {
        clearTimeout(watchdogTimerRef.current);
        watchdogTimerRef.current = null;
      }

      const chats = useChatStore.getState().chats;
      const currentChatId = useChatStore.getState().activeChatId;
      const activeC = chats.find((c) => c.id === currentChatId);
      const lastMsg = activeC?.messages?.[activeC.messages.length - 1];
      const finalContent = content || lastMsg?.content || "Listo, tarea completada.";

      setLastResponse(finalContent);
      speakTextRef.current(finalContent, () => {
        setTranscript("");
        startListeningRef.current();
      });
    };

    // 1. Suscripción a Zustand store
    const unsubStore = useChatStore.subscribe((curr, prev) => {
      const isVoiceLiveOpen = useUIStore.getState().voiceLiveOpen;
      
      // Cuando termina de responder, reproducir la voz
      if (prev.isResponding && !curr.isResponding && isVoiceLiveOpen) {
        handleAssistantResponse();
      }
      
      // Si el modelo EMPIEZA a responder (por ejemplo vía chat de texto o WS),
      // asegurarnos de cambiar visualmente al estado "thinking"
      if (!prev.isResponding && curr.isResponding && isVoiceLiveOpen) {
        setState("thinking");
        stopSpeakingRef.current();
      }
    });

    // 2. Streaming en vivo de deltas
    const unsubDelta = wsService.subscribe("message:delta", (ev: any) => {
      if (ev?.content) {
        setLastResponse((prev) => prev + ev.content);
      }
    });

    // 3. Suscripciones a eventos de finalización
    const unsubComplete = wsService.subscribe("agent:completed", () => {
      handleAssistantResponse();
    });

    const unsubDone = wsService.subscribe("done", () => {
      handleAssistantResponse();
    });

    const unsubError = wsService.subscribe("error", (ev: any) => {
      const errText = ev?.content || ev?.error || "Error al procesar la orden";
      handleAssistantResponse(`Nota: ${errText}`);
    });

    return () => {
      unsubStore();
      unsubDelta();
      unsubComplete();
      unsubDone();
      unsubError();
      if (watchdogTimerRef.current) {
        clearTimeout(watchdogTimerRef.current);
        watchdogTimerRef.current = null;
      }
    };
  }, [isOpen]);

  // Efecto de ciclo de vida del modal: se ejecuta EXACTAMENTE una vez al abrir
  useEffect(() => {
    if (isOpen) {
      setTranscript("");
      setLastResponse("");
      setState("listening");
      hasGreetedRef.current = false;

      startListeningRef.current();

      // Verificar si hay modelos de IA disponibles (LM Studio, Ollama, etc.)
      api.models
        .available()
        .then((available) => {
          const hasModels = Array.isArray(available) && available.length > 0;
          setHasConnectedModel(hasModels);
          setHasCheckedModel(true);

          if (!hasModels && !hasGreetedRef.current) {
            hasGreetedRef.current = true;
            speakTextRef.current(
              "Hola, te escucho, pero no tienes ningún modelo de inteligencia artificial conectado. Por favor, abre Ajustes o inicia LM Studio u Ollama para que pueda ayudarte."
            );
          }
        })
        .catch(() => {
          setHasConnectedModel(false);
          setHasCheckedModel(true);
          if (!hasGreetedRef.current) {
            hasGreetedRef.current = true;
            speakTextRef.current(
              "Hola, te escucho, pero no tienes ningún modelo de inteligencia artificial conectado. Por favor, abre Ajustes o inicia LM Studio u Ollama para que pueda ayudarte."
            );
          }
        });
    } else {
      stopSpeakingRef.current();
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
        } catch {}
        recognitionRef.current = null;
      }
      if (silenceTimerRef.current) clearTimeout(silenceTimerRef.current);
      setState("idle");
      setHasCheckedModel(false);
      hasGreetedRef.current = false;
    }

    return () => {
      stopSpeakingRef.current();
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
        } catch {}
      }
    };
  }, [isOpen]);

  // Cerrar con Escape o Toggle con Alt+V
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        setIsOpen(false);
      }
      if (e.altKey && (e.key === "v" || e.key === "V")) {
        useUIStore.getState().toggleVoiceLive();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, setIsOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[100] flex flex-col items-center justify-between bg-black/85 backdrop-blur-2xl p-6 sm:p-12 text-white select-none transition-all duration-300">
      {/* Header Superior */}
      <div className="w-full max-w-3xl flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-[#d1f107]/20 border border-[#d1f107]/40 flex items-center justify-center text-[#d1f107] shadow-[0_0_15px_rgba(209,241,7,0.3)]">
            <Sparkles className="w-5 h-5 animate-pulse" />
          </div>
          <div>
            <h2 className="text-lg font-bold tracking-wide text-white flex items-center gap-2">
              Ozy Live
              <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-[#d1f107]/10 text-[#d1f107] border border-[#d1f107]/30">
                <Radio className="w-3 h-3 animate-ping" /> Realtime Voice
              </span>
            </h2>
            <p className="text-xs text-neutral-400">Conversación continua bidireccional y control del sistema</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setIsMuted(!isMuted)}
            className={`p-2.5 rounded-xl border transition-all ${
              isMuted
                ? "bg-red-500/20 border-red-500/40 text-red-400"
                : "bg-neutral-800/80 border-neutral-700 hover:border-neutral-600 text-neutral-300"
            }`}
            title={isMuted ? "Activar audio" : "Silenciar voz de Ozy"}
          >
            {isMuted ? <Volume2 className="w-5 h-5 line-through" /> : <Volume2 className="w-5 h-5" />}
          </button>
          <button
            onClick={() => setIsOpen(false)}
            className="p-2.5 rounded-xl bg-neutral-800/80 border border-neutral-700 hover:border-neutral-500 text-neutral-300 hover:text-white transition-all shadow-lg"
            title="Cerrar Ozy Live (Esc)"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Banner de Aviso si no hay Modelo Conectado */}
      {hasCheckedModel && !hasConnectedModel && (
        <div className="w-full max-w-2xl mt-4 bg-amber-500/15 border border-amber-500/40 rounded-2xl p-4 flex flex-col sm:flex-row items-center justify-between gap-3 text-amber-200 text-xs shadow-[0_0_25px_rgba(245,158,11,0.2)] animate-fadeIn">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-amber-500/20 flex items-center justify-center text-amber-400 flex-shrink-0">
              <AlertTriangle className="w-4 h-4" />
            </div>
            <div>
              <strong className="text-amber-300 text-sm font-semibold block">Sin modelo de IA conectado</strong>
              <span className="text-neutral-300">Inicia LM Studio u Ollama localmente o añade una API key en Ajustes.</span>
            </div>
          </div>
          <button
            onClick={() => {
              setIsOpen(false);
              openSettings("proveedores");
            }}
            className="px-4 py-2 bg-[#d1f107] text-[#181e00] font-bold text-xs rounded-xl hover:bg-[#b8d406] transition-all whitespace-nowrap shadow-md cursor-pointer"
          >
            Configurar Modelo
          </button>
        </div>
      )}

      {/* Centro: Orbe de Voz Pulsante Holográfico */}
      <div className="flex flex-col items-center justify-center my-auto relative">
        {/* Anillos de difusión animada */}
        <div
          className={`absolute rounded-full transition-all duration-300 blur-2xl ${
            state === "listening"
              ? "bg-[#d1f107]/25 w-72 h-72 animate-pulse"
              : state === "speaking"
              ? "bg-cyan-400/30 w-80 h-80 animate-ping"
              : state === "thinking"
              ? "bg-purple-500/20 w-64 h-64 animate-spin"
              : "bg-neutral-700/20 w-56 h-56"
          }`}
          style={{ transform: `scale(${1 + audioLevel * 0.4})` }}
        />

        {/* Orbe Central Interactivo */}
        <button
          onClick={() => {
            if (state === "speaking") {
              stopSpeaking();
              startListening();
            } else if (state === "listening") {
              if (transcript) handleDispatchMessage(transcript);
            }
          }}
          className={`relative z-10 w-44 h-44 rounded-full flex flex-col items-center justify-center transition-all duration-500 shadow-2xl cursor-pointer ${
            state === "listening"
              ? "bg-gradient-to-tr from-[#181e00] via-[#2a3500] to-[#d1f107] border-4 border-[#d1f107] shadow-[0_0_50px_rgba(209,241,7,0.5)]"
              : state === "speaking"
              ? "bg-gradient-to-tr from-sky-950 via-cyan-900 to-sky-400 border-4 border-cyan-400 shadow-[0_0_50px_rgba(56,189,248,0.6)]"
              : state === "thinking"
              ? "bg-gradient-to-tr from-neutral-900 via-purple-950 to-purple-600 border-4 border-purple-500 shadow-[0_0_40px_rgba(168,85,247,0.4)]"
              : "bg-neutral-900 border-4 border-neutral-700"
          }`}
          style={{ transform: `scale(${1 + audioLevel * 0.15})` }}
        >
          {state === "listening" ? (
            <Mic className="w-16 h-16 text-[#d1f107] animate-bounce" />
          ) : state === "speaking" ? (
            <Volume2 className="w-16 h-16 text-cyan-300 animate-pulse" />
          ) : state === "thinking" ? (
            <Sparkles className="w-16 h-16 text-purple-300 animate-spin" />
          ) : (
            <MicOff className="w-14 h-14 text-neutral-500" />
          )}
        </button>

        {/* Estado en Texto */}
        <div className="mt-8 text-center">
          <span
            className={`text-sm font-semibold tracking-wider uppercase px-4 py-1.5 rounded-full border ${
              state === "listening"
                ? "bg-[#d1f107]/10 text-[#d1f107] border-[#d1f107]/40 shadow-[0_0_15px_rgba(209,241,7,0.2)]"
                : state === "speaking"
                ? "bg-cyan-500/10 text-cyan-300 border-cyan-500/40 shadow-[0_0_15px_rgba(56,189,248,0.2)]"
                : state === "thinking"
                ? "bg-purple-500/10 text-purple-300 border-purple-500/40"
                : "bg-neutral-800 text-neutral-400 border-neutral-700"
            }`}
          >
            {state === "listening" && "Te escucho..."}
            {state === "speaking" && "Ozy respondiendo..."}
            {state === "thinking" && "Razonando y ejecutando..."}
            {state === "idle" && "En pausa"}
          </span>
        </div>
      </div>

      {/* Footer: Transcripción en Vivo y Última Respuesta */}
      <div className="w-full max-w-2xl bg-neutral-900/90 border border-neutral-800/80 rounded-2xl p-6 shadow-2xl backdrop-blur-md min-h-[120px] flex flex-col justify-center text-center">
        {transcript ? (
          <p className="text-lg font-medium text-white animate-fade-in">
            &ldquo;{transcript}&rdquo;
          </p>
        ) : lastResponse ? (
          <p className="text-sm text-neutral-300 line-clamp-3 text-left">
            <strong className="text-[#d1f107]">Ozy:</strong> {lastResponse.replace(/<(think|thought)>[\s\S]*?<\/\1>/gi, "").replace(/(?:<(think|thought)>)+\s*$/gi, "").replace(/<(think|thought)>\s*/gi, "").replace(/<\/(think|thought)>\s*/gi, "").replace(/<tool\s+name=[^>]+>[\s\S]*?<\/tool>/gi, "").replace(/<tool_call>[\s\S]*?<\/tool_call>/gi, "").replace(/<HOLDER>[\s\S]*?<\/HOLDER>/gi, "").replace(/<context>[\s\S]*?(<\/context>|$)/gi, "").replace(/\s*\{"name":\s*"[^"]+",\s*"arguments":\s*\{[^}]*\}\}\s*/gi, "").replace(/^\\n\s*/, "").replace(/\\n/g, "\n")}
          </p>
        ) : (
          <p className="text-sm text-neutral-500 italic">
            Habla naturalmente. Ejemplo: &ldquo;Crea una carpeta en Documentos llamada Proyectos y dime qué ventanas tengo abiertas&rdquo;.
          </p>
        )}
      </div>
    </div>
  );
};
