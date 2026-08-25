import { useState, useRef, useEffect } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import { useToastStore } from "../../store/toastStore";
import { api } from "../../services/api";

interface LogEntry {
  type: "cmd" | "stdout" | "stderr" | "system";
  text: string;
  exitCode?: number;
  duration?: string;
}

interface BottomTerminalDrawerProps {
  isOpen: boolean;
  onToggle: () => void;
}

export default function BottomTerminalDrawer({
  isOpen,
  onToggle,
}: BottomTerminalDrawerProps) {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === activeProjectId);
  const toast = useToastStore((s) => s.show);

  const [activeTab, setActiveTab] = useState<"terminal" | "output" | "problems">("terminal");
  const [command, setCommand] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [historyIdx, setHistoryIdx] = useState<number>(-1);
  const [isRunning, setIsRunning] = useState(false);
  const [height, setHeight] = useState(240);
  const [isDragging, setIsDragging] = useState(false);

  const [logs, setLogs] = useState<LogEntry[]>([
    { type: "system", text: "OzyAssist Workspace Shell — conectado al proyecto." },
    { type: "system", text: `Directorio raíz: ${activeProject?.rootPath || "c:/Users/User/Documents/ozyAsis"}` },
  ]);

  const outputRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [logs]);

  // Handle resizing
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging) return;
      const newHeight = window.innerHeight - e.clientY;
      if (newHeight >= 120 && newHeight <= 600) {
        setHeight(newHeight);
      }
    };
    const handleMouseUp = () => setIsDragging(false);

    if (isDragging) {
      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    }
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isDragging]);

  const handleRunCommand = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!command.trim() || !activeProjectId || isRunning) return;

    const cmd = command.trim();
    setHistory((prev) => [...prev, cmd]);
    setHistoryIdx(-1);
    setCommand("");

    // Add command to log
    setLogs((prev) => [...prev, { type: "cmd", text: cmd }]);
    setIsRunning(true);

    try {
      const res = await api.terminal.exec(activeProjectId, cmd);
      if (res.stdout) {
        setLogs((prev) => [
          ...prev,
          { type: "stdout", text: res.stdout, exitCode: res.exitCode, duration: res.duration },
        ]);
      }
      if (res.stderr) {
        setLogs((prev) => [
          ...prev,
          { type: "stderr", text: res.stderr, exitCode: res.exitCode, duration: res.duration },
        ]);
      }
      if (!res.stdout && !res.stderr) {
        setLogs((prev) => [
          ...prev,
          {
            type: "system",
            text: `Comando completado sin salida (código ${res.exitCode}, duración ${res.duration})`,
            exitCode: res.exitCode,
          },
        ]);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error ejecutando comando";
      setLogs((prev) => [...prev, { type: "stderr", text: msg, exitCode: 1 }]);
      toast(msg, "error");
    } finally {
      setIsRunning(false);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowUp") {
      e.preventDefault();
      if (history.length === 0) return;
      const nextIdx = historyIdx === -1 ? history.length - 1 : Math.max(0, historyIdx - 1);
      setHistoryIdx(nextIdx);
      setCommand(history[nextIdx] || "");
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      if (historyIdx === -1) return;
      const nextIdx = historyIdx + 1;
      if (nextIdx >= history.length) {
        setHistoryIdx(-1);
        setCommand("");
      } else {
        setHistoryIdx(nextIdx);
        setCommand(history[nextIdx] || "");
      }
    }
  };

  if (!isOpen) return null;

  return (
    <div
      style={{ height: `${height}px` }}
      className="border-t border-white/10 bg-[#1a1a1a] flex flex-col font-mono text-xs z-40 relative shrink-0 select-none shadow-2xl"
    >
      {/* Drag handle for resizing */}
      <div
        onMouseDown={() => setIsDragging(true)}
        className="w-full h-1.5 cursor-row-resize bg-transparent hover:bg-[#d1f107]/40 transition-colors absolute -top-1 left-0 z-50"
      />

      {/* Terminal Drawer Header Bar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-white/10 bg-[#202020] select-none">
        <div className="flex items-center gap-1">
          <button
            onClick={() => setActiveTab("terminal")}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-sans font-medium transition-colors ${
              activeTab === "terminal"
                ? "bg-white/10 text-white font-semibold"
                : "text-zinc-400 hover:text-white"
            }`}
          >
            <span className="material-symbols-outlined text-[#d1f107] text-[15px]">terminal</span>
            <span>Terminal</span>
            {isRunning && <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />}
          </button>

          <button
            onClick={() => setActiveTab("output")}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-sans font-medium transition-colors ${
              activeTab === "output"
                ? "bg-white/10 text-white font-semibold"
                : "text-zinc-400 hover:text-white"
            }`}
          >
            <span className="material-symbols-outlined text-[15px]">output</span>
            <span>Salida</span>
          </button>

          <button
            onClick={() => setActiveTab("problems")}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-sans font-medium transition-colors ${
              activeTab === "problems"
                ? "bg-white/10 text-white font-semibold"
                : "text-zinc-400 hover:text-white"
            }`}
          >
            <span className="material-symbols-outlined text-[15px]">error_outline</span>
            <span>Problemas</span>
            <span className="text-[10px] bg-zinc-800 text-zinc-400 px-1 rounded-full">0</span>
          </button>
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => setLogs([])}
            className="p-1 rounded text-zinc-400 hover:text-white hover:bg-white/5 transition-colors"
            title="Limpiar consola"
          >
            <span className="material-symbols-outlined text-[16px]">block</span>
          </button>

          <button
            onClick={onToggle}
            className="p-1 rounded text-zinc-400 hover:text-white hover:bg-white/5 transition-colors"
            title="Ocultar panel inferior"
          >
            <span className="material-symbols-outlined text-[16px]">expand_more</span>
          </button>
        </div>
      </div>

      {/* Terminal Body */}
      {activeTab === "terminal" && (
        <div className="flex-1 flex flex-col min-h-0 bg-[#161616]">
          <div
            ref={outputRef}
            className="flex-1 overflow-y-auto p-3 space-y-1.5 text-zinc-300 text-[12px] leading-relaxed select-text scrollbar-thin"
          >
            {logs.map((entry, idx) => (
              <div key={idx} className="flex flex-col">
                {entry.type === "cmd" && (
                  <div className="flex items-center gap-2 text-white font-bold">
                    <span className="text-[#d1f107]">❯</span>
                    <span>{entry.text}</span>
                  </div>
                )}
                {entry.type === "stdout" && (
                  <pre className="text-zinc-300 whitespace-pre-wrap pl-4 font-mono text-[11px]">
                    {entry.text}
                  </pre>
                )}
                {entry.type === "stderr" && (
                  <pre className="text-rose-400 whitespace-pre-wrap pl-4 font-mono text-[11px]">
                    {entry.text}
                  </pre>
                )}
                {entry.type === "system" && (
                  <div className="text-zinc-500 italic text-[11px] pl-2">{entry.text}</div>
                )}
              </div>
            ))}
            {isRunning && (
              <div className="flex items-center gap-2 text-zinc-500 pl-4">
                <span className="w-2 h-2 rounded-full bg-lime-400 animate-ping" />
                <span>Ejecutando comando...</span>
              </div>
            )}
          </div>

          {/* Prompt input */}
          <form
            onSubmit={handleRunCommand}
            className="p-2 border-t border-white/10 bg-[#202020] flex items-center gap-2"
          >
            <span className="text-[#d1f107] font-bold pl-2 select-none">❯</span>
            <input
              ref={inputRef}
              type="text"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={isRunning}
              placeholder="Escribe un comando (ej: go test ./..., npm run build, git log -n 5)..."
              className="flex-1 bg-transparent text-white text-[12px] outline-none placeholder-zinc-600 font-mono disabled:opacity-50"
            />
            {isRunning ? (
              <span className="text-[11px] text-zinc-500 px-2 font-mono">ejecutando...</span>
            ) : (
              <button
                type="submit"
                className="px-2.5 py-1 rounded bg-[#d1f107]/15 hover:bg-[#d1f107]/25 text-[#d1f107] font-semibold text-[11px] border border-[#d1f107]/30 transition-colors"
              >
                Ejecutar
              </button>
            )}
          </form>
        </div>
      )}

      {/* Output Tab */}
      {activeTab === "output" && (
        <div className="flex-1 overflow-y-auto p-3 text-zinc-400 text-[11px] bg-[#161616] select-text">
          <div className="text-zinc-500">[OzyAssist Core Runtime] Transmisión de registros de agente y compilación activa.</div>
          <div className="mt-2 text-zinc-400">Todo el feedback de herramientas ejecutadas se canaliza a través de esta salida.</div>
        </div>
      )}

      {/* Problems Tab */}
      {activeTab === "problems" && (
        <div className="flex-1 flex items-center justify-center text-zinc-500 bg-[#161616]">
          <div className="flex items-center gap-2 text-[12px]">
            <span className="material-symbols-outlined text-emerald-400 text-[18px]">check_circle</span>
            <span>No se han detectado errores ni problemas de sintaxis.</span>
          </div>
        </div>
      )}
    </div>
  );
}
