import { useState } from "react";

export type ToolCallStatus = "calling" | "success" | "error" | "awaiting_approval";

export interface ToolCallBlockData {
  toolId: string;
  toolName: string;
  toolInput: string;
  status: ToolCallStatus;
  output?: string;
  durationMs?: number;
}

interface ToolCallBlockProps {
  block: ToolCallBlockData;
  onApprove?: (toolId: string) => void;
  onDeny?: (toolId: string) => void;
  onViewDiff?: (path: string) => void;
}

const TOOL_ICONS: Record<string, string> = {
  read_file: "description",
  write_file: "edit_document",
  run_command: "terminal",
  list_files: "folder_open",
  search_text: "search",
  apply_diff: "difference",
};

const TOOL_LABELS: Record<string, string> = {
  read_file: "read_file",
  write_file: "write_file",
  run_command: "run_command",
  list_files: "list_files",
  search_text: "search_text",
  apply_diff: "apply_diff",
};

function StatusBadge({ status }: { status: ToolCallStatus }) {
  if (status === "calling") {
    return (
      <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-mono bg-lime-500/10 text-lime-400 border border-lime-500/20">
        <span className="w-1.5 h-1.5 rounded-full bg-lime-400 animate-pulse" />
        Running...
      </span>
    );
  }
  if (status === "awaiting_approval") {
    return (
      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono bg-amber-500/15 text-amber-300 border border-amber-500/30">
        <span className="material-symbols-outlined text-[12px]">pending</span>
        Requiere Aprobación
      </span>
    );
  }
  if (status === "success") {
    return (
      <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
        <span className="material-symbols-outlined text-[12px]">check</span>
        Completado
      </span>
    );
  }
  return (
    <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono bg-rose-500/10 text-rose-400 border border-rose-500/20">
      <span className="material-symbols-outlined text-[12px]">error</span>
      Error
    </span>
  );
}

function formatInput(toolName: string, input: string): string {
  try {
    const parsed = JSON.parse(input);
    if (toolName === "read_file" || toolName === "write_file" || toolName === "apply_diff") {
      return parsed.path ?? input;
    }
    if (toolName === "run_command") {
      return parsed.command ?? input;
    }
    if (toolName === "list_files") {
      return parsed.pattern ?? input;
    }
    if (toolName === "search_text") {
      return parsed.query ?? input;
    }
    return JSON.stringify(parsed, null, 2);
  } catch {
    return input;
  }
}

export default function ToolCallBlock({ block, onApprove, onDeny, onViewDiff }: ToolCallBlockProps) {
  const [outputExpanded, setOutputExpanded] = useState(false);
  const icon = TOOL_ICONS[block.toolName] ?? "build";
  const label = TOOL_LABELS[block.toolName] ?? block.toolName;
  const displayInput = formatInput(block.toolName, block.toolInput);
  const hasOutput = block.output && block.output.length > 0;
  const isDiffTool = block.toolName === "apply_diff" || block.toolName === "write_file";

  const handleOpenDiff = () => {
    if (onViewDiff) {
      onViewDiff(displayInput);
    } else {
      window.dispatchEvent(
        new CustomEvent("switch-project-tab", { detail: { tab: "diffs", path: displayInput } })
      );
    }
  };

  return (
    <div
      className={`my-1.5 rounded-xl border text-xs font-mono transition-all duration-200 overflow-hidden shadow-sm ${
        block.status === "awaiting_approval"
          ? "border-amber-500/30 bg-amber-950/20"
          : block.status === "success"
          ? "border-white/10 bg-zinc-900/90 hover:border-white/20"
          : block.status === "error"
          ? "border-rose-900/40 bg-rose-950/20"
          : "border-lime-500/25 bg-zinc-900/95 shadow-lime-500/5 shadow-md"
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 bg-zinc-900/60">
        <div className="w-6 h-6 rounded-lg bg-white/5 flex items-center justify-center text-lime-400 shrink-0">
          <span className="material-symbols-outlined text-[15px]">{icon}</span>
        </div>

        <span className="text-zinc-400 font-medium font-mono text-[11px]">{label}</span>

        <code
          className="flex-1 text-zinc-200 text-[11px] truncate font-mono bg-black/20 px-2 py-0.5 rounded border border-white/5"
          title={displayInput}
        >
          {displayInput}
        </code>

        {block.durationMs !== undefined && block.durationMs > 0 && (
          <span className="text-zinc-500 text-[10px] tabular-nums shrink-0">
            {block.durationMs < 1000 ? `${block.durationMs}ms` : `${(block.durationMs / 1000).toFixed(1)}s`}
          </span>
        )}

        {isDiffTool && block.status === "success" && (
          <button
            onClick={handleOpenDiff}
            className="flex items-center gap-1 px-2 py-0.5 rounded-md bg-lime-500/10 hover:bg-lime-500/20 text-lime-400 border border-lime-500/30 text-[10px] font-mono transition-colors"
            title="Abrir inspector de diferencias"
          >
            <span className="material-symbols-outlined text-[12px]">difference</span>
            <span>Ver Diff</span>
          </button>
        )}

        <StatusBadge status={block.status} />
      </div>

      {/* Botones de aprobación interactiva */}
      {block.status === "awaiting_approval" && (
        <div className="flex items-center justify-between gap-2 px-3 py-2 bg-amber-950/30 border-t border-amber-500/20">
          <div className="text-[11px] text-amber-200/80 font-sans">
            Esta acción requiere tu confirmación para modificar el espacio de trabajo.
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <button
              onClick={() => onApprove?.(block.toolId)}
              className="flex items-center gap-1 px-3 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-[11px] font-sans font-medium transition-colors shadow-sm"
            >
              <span className="material-symbols-outlined text-[14px]">check</span>
              Permitir
            </button>
            <button
              onClick={() => onDeny?.(block.toolId)}
              className="flex items-center gap-1 px-3 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] font-sans font-medium transition-colors border border-white/10"
            >
              <span className="material-symbols-outlined text-[14px]">close</span>
              Denegar
            </button>
          </div>
        </div>
      )}

      {/* Output colapsable */}
      {hasOutput && block.status !== "awaiting_approval" && (
        <div className="border-t border-white/5 bg-black/20">
          <button
            onClick={() => setOutputExpanded((v) => !v)}
            className="flex items-center gap-1 w-full px-3 py-1.5 text-zinc-400 hover:text-zinc-200 transition-colors font-mono text-[10px]"
          >
            <span className="material-symbols-outlined text-[14px]">
              {outputExpanded ? "expand_less" : "expand_more"}
            </span>
            <span>{outputExpanded ? "Ocultar salida" : "Ver salida de la herramienta"}</span>
          </button>
          {outputExpanded && (
            <pre className="px-3 pb-3 text-[11px] text-zinc-300 whitespace-pre-wrap overflow-x-auto max-h-56 scrollbar-thin font-mono leading-relaxed bg-black/30 p-2.5 m-2 rounded-lg border border-white/5">
              {block.output}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
