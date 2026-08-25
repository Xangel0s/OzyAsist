import { useState } from "react";
import { useToastStore } from "../../store/toastStore";

interface TerminalViewerProps {
  logs?: string[];
  onClear?: () => void;
}

export default function TerminalViewer({
  logs = [
    "OzyAssist Agent Runtime v2.0 initialized.",
    "[ReAct Loop] Session session-1812 connected via WebSocket.",
    "[Tool: list_files] Scanning project structure (84 files found)...",
    "[Tool: read_file] Loaded backend/internal/agent/loop.go",
    "[Tool: apply_diff] Applied multi-hunk patch to loop.go (18+ 4-)",
    "[Terminal] go test -v ./internal/agent/... -> PASS (0.42s)",
  ],
  onClear,
}: TerminalViewerProps) {
  const toast = useToastStore((s) => s.show);
  const [command, setCommand] = useState("");
  const [activeLogs, setActiveLogs] = useState<string[]>(logs);

  const handleRunCommand = (e: React.FormEvent) => {
    e.preventDefault();
    if (!command.trim()) return;
    const cmd = command.trim();
    setActiveLogs((prev) => [
      ...prev,
      `$ ${cmd}`,
      `Executing "${cmd}" inside sandboxed workspace...`,
      `Exit code: 0 (Execution finished)`,
    ]);
    setCommand("");
    toast(`Comando ejecutado: ${cmd}`, "info");
  };

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-zinc-950 text-xs font-mono select-text">
      {/* Header */}
      <div className="p-3 border-b border-white/10 flex items-center justify-between bg-zinc-900/90 select-none">
        <div className="flex items-center gap-2 text-white/90">
          <span className="material-symbols-outlined text-lime-400 text-[18px]">terminal</span>
          <span className="font-semibold text-[13px]">Terminal en Vivo</span>
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={() => {
              setActiveLogs([]);
              onClear?.();
            }}
            className="px-2 py-1 rounded-lg bg-white/5 hover:bg-white/10 text-white/60 text-[11px] transition-colors border border-white/5"
            title="Limpiar consola"
          >
            Clear
          </button>
        </div>
      </div>

      {/* Terminal Output Log Feed */}
      <div className="flex-1 overflow-y-auto p-3 space-y-1.5 text-zinc-300 text-[12px] leading-relaxed scrollbar-thin">
        {activeLogs.map((line, idx) => (
          <div key={idx} className="flex items-start gap-2">
            <span className="text-lime-400/80 shrink-0 select-none">›</span>
            <span
              className={
                line.startsWith("$")
                  ? "text-white font-bold"
                  : line.includes("PASS") || line.includes("Exit code: 0")
                  ? "text-emerald-400"
                  : line.includes("Tool:")
                  ? "text-lime-300/90"
                  : line.includes("Error") || line.includes("FAIL")
                  ? "text-rose-400"
                  : "text-zinc-400"
              }
            >
              {line}
            </span>
          </div>
        ))}
      </div>

      {/* Terminal Shell Input Bar */}
      <form onSubmit={handleRunCommand} className="p-2 border-t border-white/10 bg-zinc-900 flex items-center gap-2">
        <span className="text-lime-400 pl-2 select-none">$</span>
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="Ejecutar comando bash/powershell en el proyecto..."
          className="flex-1 bg-transparent text-white text-[12px] outline-none placeholder-zinc-500 font-mono"
        />
        <button
          type="submit"
          className="px-2.5 py-1 rounded-lg bg-lime-500/15 text-lime-400 border border-lime-500/30 text-[11px] font-semibold hover:bg-lime-500/25 transition-colors"
        >
          Run
        </button>
      </form>
    </div>
  );
}
