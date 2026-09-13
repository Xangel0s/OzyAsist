import { useEffect, useRef, useState, useCallback } from "react";
import { useToastStore } from "../store/toastStore";

function playWakeChime() {
  try {
    const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
    if (!AudioContextClass) return;
    const ctx = new AudioContextClass();
    const now = ctx.currentTime;

    const osc1 = ctx.createOscillator();
    const gain1 = ctx.createGain();
    osc1.type = "sine";
    osc1.frequency.setValueAtTime(587.33, now);
    osc1.frequency.exponentialRampToValueAtTime(880, now + 0.15);

    gain1.gain.setValueAtTime(0.18, now);
    gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.35);

    osc1.connect(gain1);
    gain1.connect(ctx.destination);

    osc1.start(now);
    osc1.stop(now + 0.35);
  } catch {
    // ignore
  }
}

function isWakePhrase(text: string): boolean {
  if (!text) return false;
  const clean = text.toLowerCase().trim();

  // 1. Frases y palabras clave fonéticas directas comunes
  const directKeywords = [
    "hey ozy",
    "oye ozy",
    "hola ozy",
    "ok ozy",
    "okey ozy",
    "hey osi",
    "oye osi",
    "hey osy",
    "hey ozi",
    "oye ozi",
    "ey ozy",
    "ey ozi",
    "ey osi",
    "jei ozy",
    "jei ozi",
    "jei osi",
    "hay ozy",
    "hay ozi",
    "ay ozy",
    "ay ozi",
    "el ozy",
    "el ozi",
    "oe ozy",
    "oe ozi",
    "ozy assist",
    "ozy asist",
    "ozi assist",
    "ozi asist",
    "abrir ozy",
    "abre ozy",
    "iniciar ozy",
  ];

  if (directKeywords.some((kw) => clean.includes(kw))) {
    return true;
  }

  // 2. Expresión regular flexible para variaciones fonéticas en español e inglés
  const wakeRegex = /\b(hey|oye|hola|ey|jei|ay|hay|ok|okay|oe|e|el|halo|abre|abrir)?\s*(ozy|ozi|osi|osy|ozzy|osie|ossy|ocy|oz)\b/i;
  if (wakeRegex.test(clean)) {
    return true;
  }

  // 3. Palabra aislada "ozy" / "ozi" en enunciados cortos
  if (clean.length < 25 && (clean.includes("ozy") || clean.includes("ozi") || clean.includes("ozzy"))) {
    return true;
  }

  return false;
}

interface UseWakeWordOptions {
  enabled?: boolean;
  onWakeWordDetected?: () => void;
}

export function useWakeWord({
  enabled = true,
  onWakeWordDetected,
}: UseWakeWordOptions = {}) {
  const [isListening, setIsListening] = useState(false);
  const [isSupported, setIsSupported] = useState(false);
  const recognitionRef = useRef<any>(null);
  const onWakeRef = useRef(onWakeWordDetected);
  onWakeRef.current = onWakeWordDetected;
  const isEnabledRef = useRef(enabled);
  isEnabledRef.current = enabled;

  const triggerWake = useCallback(() => {
    playWakeChime();
    useToastStore.getState().show("✨ ¡Hey Ozy activado!", "success");
    if (onWakeRef.current) {
      onWakeRef.current();
    }
  }, []);

  // Solicitar permisos de micrófono al usuario si no han sido otorgados
  useEffect(() => {
    if (typeof navigator !== "undefined" && navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
      // Verificar estado del micrófono si está soportado
      navigator.mediaDevices
        .getUserMedia({ audio: true })
        .then((stream) => {
          // Detener las pistas de prueba de inmediato para no retener el audio
          stream.getTracks().forEach((track) => track.stop());
        })
        .catch((err) => {
          console.debug("Microphone access check:", err);
        });
    }
  }, []);

  useEffect(() => {
    const SpeechRecognition =
      (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;

    if (!SpeechRecognition) {
      setIsSupported(false);
      return;
    }

    setIsSupported(true);

    if (!enabled) {
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
        } catch {
          // ignore
        }
        recognitionRef.current = null;
      }
      setIsListening(false);
      return;
    }

    let isDestroyed = false;

    const startRecognizer = () => {
      if (isDestroyed || !isEnabledRef.current) return;

      try {
        if (recognitionRef.current) {
          try {
            recognitionRef.current.stop();
          } catch {}
          recognitionRef.current = null;
        }

        const recognition = new SpeechRecognition();
        recognition.continuous = true;
        recognition.interimResults = true;
        recognition.lang = "es-ES";

        recognition.onstart = () => {
          setIsListening(true);
        };

        recognition.onresult = (event: any) => {
          for (let i = event.resultIndex; i < event.results.length; ++i) {
            const transcript = event.results[i][0].transcript;
            if (isWakePhrase(transcript)) {
              triggerWake();
              break;
            }
          }
        };

        recognition.onerror = (e: any) => {
          if (e.error !== "no-speech") {
            console.debug("WakeWord recognition error:", e.error);
          }
        };

        recognition.onend = () => {
          setIsListening(false);
          // Reinicio continuo inteligente tras 500ms si sigue habilitado
          if (!isDestroyed && isEnabledRef.current) {
            setTimeout(() => {
              if (!isDestroyed && isEnabledRef.current) {
                startRecognizer();
              }
            }, 500);
          }
        };

        recognitionRef.current = recognition;
        recognition.start();
      } catch (err) {
        console.debug("Speech recognition start failed:", err);
      }
    };

    startRecognizer();

    return () => {
      isDestroyed = true;
      if (recognitionRef.current) {
        try {
          recognitionRef.current.stop();
          recognitionRef.current = null;
        } catch {
          // ignore
        }
      }
      setIsListening(false);
    };
  }, [enabled, triggerWake]);

  return { isListening, isSupported };
}
export default useWakeWord;
