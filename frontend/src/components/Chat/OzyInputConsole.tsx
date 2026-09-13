import React, { useState, useRef, useEffect } from 'react';
import { 
  Plus, 
  Folder, 
  Mic, 
  ArrowUp, 
  SlidersHorizontal, 
  X, 
  Check, 
  ChevronDown,
  Sparkles,
  ShieldCheck,
  Search,
  Code2
} from 'lucide-react';

import { useUIStore } from '../../store/uiStore';

interface OzyInputConsoleProps {
  onSendMessage: (text: string, folder?: string) => void;
  onStartVoiceRecording?: () => void;
  isProcessing?: boolean;
  placeholder?: string;
}

export const OzyInputConsole: React.FC<OzyInputConsoleProps> = ({
  onSendMessage,
  onStartVoiceRecording: _onStartVoiceRecording,
  isProcessing = false,
  placeholder = "Asigna una tarea a Ozy o escribe / para habilidades...",
}) => {
  const [inputText, setInputText] = useState('');
  const [isRecording, setIsRecording] = useState(false);
  const [recordSeconds, setRecordSeconds] = useState(0);
  const [selectedDirectory, setSelectedDirectory] = useState('Workspace Principal (/workspace)');
  const [showDirectoryMenu, setShowDirectoryMenu] = useState(false);
  const [showSkillsPalette, setShowSkillsPalette] = useState(false);
  const [waveLevels, setWaveLevels] = useState<number[]>(new Array(28).fill(4));

  const audioCtxRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animFrameRef = useRef<number | null>(null);
  const mediaStreamRef = useRef<MediaStream | null>(null);

  // Activación del analizador de frecuencia con Web Audio API real
  useEffect(() => {
    if (isRecording) {
      const initAudio = async () => {
        try {
          const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
          mediaStreamRef.current = stream;
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const AudioContextClass = window.AudioContext || (window as any).webkitAudioContext;
          const ctx = new AudioContextClass();
          const analyser = ctx.createAnalyser();
          analyser.fftSize = 64;
          const source = ctx.createMediaStreamSource(stream);
          source.connect(analyser);

          audioCtxRef.current = ctx;
          analyserRef.current = analyser;

          const dataArray = new Uint8Array(analyser.frequencyBinCount);
          const updateWaves = () => {
            analyser.getByteFrequencyData(dataArray);
            const normalized = Array.from(dataArray.slice(0, 28)).map((val) =>
              Math.max(4, Math.floor((val / 255) * 26))
            );
            setWaveLevels(normalized);
            animFrameRef.current = requestAnimationFrame(updateWaves);
          };
          updateWaves();
        } catch {
          // Fallback a simulación estocástica si no hay micrófono disponible
          setWaveLevels(Array.from({ length: 28 }, () => Math.floor(Math.random() * 20) + 4));
        }
      };

      initAudio();
      const timer = setInterval(() => setRecordSeconds((prev) => prev + 1), 1000);

      return () => {
        clearInterval(timer);
        if (animFrameRef.current) cancelAnimationFrame(animFrameRef.current);
        if (audioCtxRef.current) audioCtxRef.current.close();
        if (mediaStreamRef.current) {
          mediaStreamRef.current.getTracks().forEach((t) => t.stop());
        }
      };
    } else {
      setRecordSeconds(0);
      setWaveLevels(new Array(28).fill(4));
    }
  }, [isRecording]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (inputText.trim()) {
        onSendMessage(inputText, selectedDirectory);
        setInputText('');
      }
    }
    if (e.key === '/') {
      setShowSkillsPalette(true);
    }
  };

  const formatTimer = (totalSecs: number) => {
    const mins = Math.floor(totalSecs / 60);
    const secs = totalSecs % 60;
    return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
  };

  const ozySkills = [
    { id: 'deep-search', name: 'deep-search', desc: 'Investigación web con citas verificadas [1][2]', icon: Search },
    { id: 'charc-audit', name: 'charc-audit', desc: 'Auditoría heurística y control de seguridad local', icon: ShieldCheck },
    { id: 'code-refactor', name: 'code-refactor', desc: 'Refactorización y síntesis con OpenCode', icon: Code2 },
    { id: 'triad-solve', name: 'triad-solve', desc: 'Razonamiento estratégico profundo con Nine', icon: Sparkles },
  ];

  return (
    <div className="relative w-full max-w-3xl mx-auto">
      {/* Popover de Habilidades Ozy */}
      {showSkillsPalette && (
        <div className="absolute bottom-full mb-3 left-0 w-80 rounded-2xl border border-white/10 bg-[#171717]/95 p-2 shadow-2xl backdrop-blur-xl z-30">
          <p className="px-3 py-1.5 text-[10px] font-bold tracking-wider text-neutral-400 uppercase">
            Habilidades de OzyAssist
          </p>
          <div className="space-y-1">
            {ozySkills.map((s) => {
              const IconComp = s.icon;
              return (
                <button
                  key={s.id}
                  onClick={() => {
                    setInputText(`/${s.name} `);
                    setShowSkillsPalette(false);
                  }}
                  className="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-xs text-neutral-200 hover:bg-white/5 transition-all"
                >
                  <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-[#d1f107]/10 text-[#d1f107]">
                    <IconComp className="h-4 w-4" />
                  </span>
                  <div>
                    <span className="font-mono font-bold text-[#d1f107]">/{s.name}</span>
                    <p className="text-[10px] text-neutral-400">{s.desc}</p>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Contenedor Principal de la Consola */}
      <div className="relative rounded-3xl border border-white/10 bg-[#161616]/90 p-4 shadow-2xl backdrop-blur-2xl transition-all duration-300 focus-within:border-[#d1f107]/40 focus-within:ring-1 focus-within:ring-[#d1f107]/20">
        {!isRecording ? (
          <>
            <textarea
              rows={2}
              value={inputText}
              onChange={(e) => {
                setInputText(e.target.value);
                if (!e.target.value.includes('/')) setShowSkillsPalette(false);
              }}
              onKeyDown={handleKeyDown}
              placeholder={placeholder}
              className="w-full resize-none bg-transparent font-sans text-sm text-neutral-100 placeholder-neutral-500 focus:outline-none"
            />

            {/* Barra Inferior de Herramientas y Contexto */}
            <div className="mt-3 flex items-center justify-between border-t border-white/5 pt-3">
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  title="Adjuntar archivo o imagen"
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-neutral-400 hover:bg-white/10 hover:text-white transition-all"
                >
                  <Plus className="h-4 w-4" />
                </button>

                <button
                  type="button"
                  title="Conectores MCP y Herramientas"
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-neutral-400 hover:bg-white/10 hover:text-white transition-all"
                >
                  <SlidersHorizontal className="h-3.5 w-3.5" />
                </button>

                {/* Selector de Directorio / Contexto de Trabajo */}
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setShowDirectoryMenu(!showDirectoryMenu)}
                    className="flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs text-neutral-300 hover:bg-white/10 hover:text-white transition-all"
                  >
                    <Folder className="h-3.5 w-3.5 text-[#d1f107]" />
                    <span className="max-w-[150px] truncate">{selectedDirectory}</span>
                    <ChevronDown className="h-3 w-3 text-neutral-500" />
                  </button>

                  {showDirectoryMenu && (
                    <div className="absolute left-0 top-full mt-2 w-64 rounded-2xl border border-white/10 bg-[#181818] p-1.5 shadow-2xl z-30">
                      <button
                        type="button"
                        onClick={() => { setSelectedDirectory('Workspace Principal (/workspace)'); setShowDirectoryMenu(false); }}
                        className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5"
                      >
                        <Folder className="h-3.5 w-3.5 text-[#d1f107]" />
                        <span>Workspace Principal (/workspace)</span>
                      </button>
                      <button
                        type="button"
                        onClick={() => { setSelectedDirectory('ZBook System Root'); setShowDirectoryMenu(false); }}
                        className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5"
                      >
                        <Folder className="h-3.5 w-3.5 text-neutral-400" />
                        <span>ZBook System Root</span>
                      </button>
                    </div>
                  )}
                </div>
              </div>

              {/* Controles de Entrada */}
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => useUIStore.getState().setVoiceLiveOpen(true)}
                  title="Ozy Live — Voz en Tiempo Real (Hey Ozy)"
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-neutral-400 hover:bg-[#d1f107]/20 hover:text-[#d1f107] transition-all"
                >
                  <Mic className="h-4 w-4" />
                </button>

                <button
                  type="button"
                  disabled={!inputText.trim() || isProcessing}
                  onClick={() => {
                    onSendMessage(inputText, selectedDirectory);
                    setInputText('');
                  }}
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-[#d1f107] text-black font-bold hover:opacity-90 disabled:opacity-30 disabled:hover:opacity-30 transition-all shadow-[0_0_15px_rgba(209,241,7,0.3)]"
                >
                  <ArrowUp className="h-4 w-4 stroke-[2.5]" />
                </button>
              </div>
            </div>
          </>
        ) : (
          /* Estado de Grabación de Audio con Ecualizador Reactivo */
          <div className="flex h-16 items-center justify-between px-2">
            <div className="flex items-center gap-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-full bg-[#d1f107]/20 text-[#d1f107]">
                <Mic className="h-4 w-4 animate-pulse" />
              </span>

              <div className="flex items-center gap-[3px] h-7">
                {waveLevels.map((lvl, idx) => (
                  <span
                    key={idx}
                    style={{ height: `${lvl}px` }}
                    className="w-[3px] rounded-full bg-[#d1f107] transition-all duration-75"
                  />
                ))}
              </div>

              <span className="font-mono text-xs font-semibold text-neutral-400 pl-2">
                {formatTimer(recordSeconds)}
              </span>
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsRecording(false)}
                title="Cancelar grabación"
                className="flex h-8 w-8 items-center justify-center rounded-full bg-white/5 text-neutral-400 hover:bg-white/10 hover:text-rose-400 transition-all"
              >
                <X className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => {
                  setIsRecording(false);
                  onSendMessage('Audio recibido por voz', selectedDirectory);
                }}
                title="Enviar audio"
                className="flex h-8 w-8 items-center justify-center rounded-full bg-[#d1f107] text-black font-bold hover:opacity-90 transition-all shadow-[0_0_15px_rgba(209,241,7,0.3)]"
              >
                <Check className="h-4 w-4 stroke-[2.5]" />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Píldoras de Sugerencia Inferiores */}
      <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
        {['Desarrollar app Go', 'Auditar seguridad local', 'Analizar datos del sistema', 'DeepSearch web'].map((item) => (
          <button
            key={item}
            type="button"
            onClick={() => setInputText(item)}
            className="rounded-full border border-white/5 bg-white/[0.03] px-4 py-1.5 text-xs text-neutral-400 hover:border-[#d1f107]/30 hover:bg-white/[0.07] hover:text-neutral-200 transition-all"
          >
            {item}
          </button>
        ))}
      </div>
    </div>
  );
};
export default OzyInputConsole;
