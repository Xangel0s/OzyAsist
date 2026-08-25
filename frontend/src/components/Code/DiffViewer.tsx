import { useState } from "react";

export interface DiffFile {
  path: string;
  oldContent: string;
  newContent: string;
  addedLines: number;
  deletedLines: number;
}

interface DiffViewerProps {
  files?: DiffFile[];
  activeFilePath?: string | null;
  onAcceptAll?: () => void;
  onRevertAll?: () => void;
  onClose?: () => void;
}

const DEFAULT_DEMO_DIFFS: DiffFile[] = [
  {
    path: "backend/internal/api/handlers/git.go",
    oldContent: `// No native git integration`,
    newContent: `// Native Git operations handler
func GetGitStatus(c *gin.Context) {
    // Real git status parsing
}
func InitGit(c *gin.Context) {
    // Real git init
}`,
    addedLines: 8,
    deletedLines: 1,
  },
  {
    path: "frontend/src/components/Code/BottomTerminalDrawer.tsx",
    oldContent: `// Previous sidebar terminal`,
    newContent: `export default function BottomTerminalDrawer() {
    // Real interactive shell execution with output streaming
}`,
    addedLines: 12,
    deletedLines: 1,
  },
];

export default function DiffViewer({
  files = DEFAULT_DEMO_DIFFS,
  activeFilePath,
  onAcceptAll,
  onRevertAll,
  onClose,
}: DiffViewerProps) {
  const [selectedPath, setSelectedPath] = useState<string>(
    activeFilePath || (files.length > 0 ? files[0].path : "")
  );
  const [viewMode, setViewMode] = useState<"unified" | "split">("unified");

  const currentFile = files.find((f) => f.path === selectedPath) || files[0];

  const totalAdded = files.reduce((acc, f) => acc + f.addedLines, 0);
  const totalDeleted = files.reduce((acc, f) => acc + f.deletedLines, 0);

  const renderUnified = () => {
    if (!currentFile) return null;
    const oldLines = currentFile.oldContent.split("\n");
    const newLines = currentFile.newContent.split("\n");

    return (
      <div className="font-mono text-[12px] leading-relaxed select-text overflow-x-auto divide-y divide-white/[0.03]">
        {oldLines.map((line, i) => (
          <div
            key={`del-${i}`}
            className="flex items-start bg-rose-500/[0.08] text-rose-300/90 hover:bg-rose-500/[0.12] transition-colors"
          >
            <span className="w-10 select-none text-rose-500/40 text-[10px] text-right pr-2 shrink-0 py-0.5 border-r border-rose-500/20 font-mono">
              {i + 1}
            </span>
            <span className="w-6 select-none text-rose-400 font-bold text-center shrink-0 py-0.5">
              -
            </span>
            <span className="flex-1 whitespace-pre py-0.5 pl-2 font-mono text-[11px] leading-snug">
              {line}
            </span>
          </div>
        ))}
        {newLines.map((line, i) => (
          <div
            key={`add-${i}`}
            className="flex items-start bg-emerald-500/[0.08] text-emerald-300/90 hover:bg-emerald-500/[0.12] transition-colors"
          >
            <span className="w-10 select-none text-emerald-500/40 text-[10px] text-right pr-2 shrink-0 py-0.5 border-r border-emerald-500/20 font-mono">
              {i + 1}
            </span>
            <span className="w-6 select-none text-emerald-400 font-bold text-center shrink-0 py-0.5">
              +
            </span>
            <span className="flex-1 whitespace-pre py-0.5 pl-2 font-mono text-[11px] leading-snug">
              {line}
            </span>
          </div>
        ))}
      </div>
    );
  };

  const renderSplit = () => {
    if (!currentFile) return null;
    const oldLines = currentFile.oldContent.split("\n");
    const newLines = currentFile.newContent.split("\n");
    const maxLines = Math.max(oldLines.length, newLines.length);

    return (
      <div className="grid grid-cols-2 divide-x divide-white/10 font-mono text-[11px] leading-relaxed select-text overflow-x-auto">
        {/* Left: Original */}
        <div className="flex flex-col divide-y divide-white/[0.03]">
          <div className="bg-[#181818] text-zinc-400 px-3 py-1 text-[10px] uppercase font-bold sticky top-0 border-b border-white/5">
            Original
          </div>
          {Array.from({ length: maxLines }).map((_, i) => {
            const line = oldLines[i];
            return (
              <div
                key={`left-${i}`}
                className={`flex items-start ${
                  line !== undefined
                    ? "bg-rose-500/[0.08] text-rose-300"
                    : "bg-transparent text-transparent"
                }`}
              >
                <span className="w-8 select-none text-zinc-600 text-[10px] text-right pr-2 shrink-0 py-0.5 border-r border-white/5">
                  {line !== undefined ? i + 1 : ""}
                </span>
                <span className="flex-1 whitespace-pre py-0.5 pl-2 truncate font-mono text-[11px]">
                  {line || " "}
                </span>
              </div>
            );
          })}
        </div>

        {/* Right: Modified */}
        <div className="flex flex-col divide-y divide-white/[0.03]">
          <div className="bg-[#181818] text-[#d1f107] px-3 py-1 text-[10px] uppercase font-bold sticky top-0 border-b border-white/5">
            Propuesto (Nuevo)
          </div>
          {Array.from({ length: maxLines }).map((_, i) => {
            const line = newLines[i];
            return (
              <div
                key={`right-${i}`}
                className={`flex items-start ${
                  line !== undefined
                    ? "bg-emerald-500/[0.08] text-emerald-300"
                    : "bg-transparent text-transparent"
                }`}
              >
                <span className="w-8 select-none text-zinc-600 text-[10px] text-right pr-2 shrink-0 py-0.5 border-r border-white/5">
                  {line !== undefined ? i + 1 : ""}
                </span>
                <span className="flex-1 whitespace-pre py-0.5 pl-2 truncate font-mono text-[11px]">
                  {line || " "}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-[#181818] text-xs font-sans select-none">
      {/* Header bar */}
      <div className="p-3 border-b border-white/10 flex items-center justify-between bg-[#202020]">
        <div className="flex items-center gap-2 font-mono">
          <span className="material-symbols-outlined text-[#d1f107] text-[18px]">difference</span>
          <span className="font-semibold text-white/90 text-[13px]">
            Cambios propuestos ({files.length})
          </span>
          <span className="text-[11px] text-emerald-400 font-semibold bg-emerald-500/10 px-1.5 py-0.5 rounded border border-emerald-500/20">
            +{totalAdded}
          </span>
          <span className="text-[11px] text-rose-400 font-semibold bg-rose-500/10 px-1.5 py-0.5 rounded border border-rose-500/20">
            -{totalDeleted}
          </span>
        </div>

        <div className="flex items-center gap-1.5">
          <div className="flex rounded-lg bg-black/40 border border-white/10 p-0.5">
            <button
              onClick={() => setViewMode("unified")}
              className={`px-2 py-0.5 rounded text-[11px] font-mono transition-colors ${
                viewMode === "unified"
                  ? "bg-white/15 text-white font-semibold"
                  : "text-zinc-400 hover:text-white"
              }`}
            >
              Unified
            </button>
            <button
              onClick={() => setViewMode("split")}
              className={`px-2 py-0.5 rounded text-[11px] font-mono transition-colors ${
                viewMode === "split"
                  ? "bg-white/15 text-white font-semibold"
                  : "text-zinc-400 hover:text-white"
              }`}
            >
              Split
            </button>
          </div>

          {onClose && (
            <button
              onClick={onClose}
              className="w-7 h-7 rounded-lg flex items-center justify-center text-zinc-400 hover:text-white hover:bg-white/5 transition-colors"
            >
              <span className="material-symbols-outlined text-[16px]">close</span>
            </button>
          )}
        </div>
      </div>

      {/* File Pills Bar */}
      <div className="flex items-center gap-1.5 px-3 py-2 border-b border-white/5 overflow-x-auto bg-[#1c1c1c] scrollbar-none font-mono text-[11px]">
        {files.map((f) => (
          <button
            key={f.path}
            onClick={() => setSelectedPath(f.path)}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg transition-all shrink-0 ${
              f.path === selectedPath
                ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 font-semibold shadow-sm"
                : "bg-white/5 hover:bg-white/10 text-white/60 border border-white/5"
            }`}
          >
            <span className="truncate max-w-[150px]">{f.path.split("/").pop()}</span>
            <span className="text-emerald-400 font-bold">+{f.addedLines}</span>
            <span className="text-rose-400 font-bold">-{f.deletedLines}</span>
          </button>
        ))}
      </div>

      {/* Diff Code Container */}
      <div className="flex-1 overflow-y-auto p-3 min-h-0 bg-[#161616]">
        <div className="rounded-xl border border-white/10 bg-[#1e1e1e] overflow-hidden shadow-xl">
          <div className="px-3 py-2 border-b border-white/5 bg-[#242424] flex items-center justify-between text-white/50 font-mono text-[11px]">
            <span className="text-white/90 font-medium truncate">{currentFile?.path}</span>
            <span className="text-white/40 text-[10px]">{viewMode === "unified" ? "Vista unificada" : "Vista dividida"}</span>
          </div>
          {viewMode === "unified" ? renderUnified() : renderSplit()}
        </div>
      </div>

      {/* Footer Controls */}
      <div className="p-3 border-t border-white/10 bg-[#202020] flex items-center justify-between">
        <button
          onClick={onRevertAll}
          className="px-3 py-1.5 rounded-xl bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 font-medium text-[11px] transition-colors border border-rose-500/20 flex items-center gap-1 font-mono"
        >
          <span className="material-symbols-outlined text-[15px]">undo</span>
          <span>Revertir cambios</span>
        </button>

        <button
          onClick={onAcceptAll}
          className="px-4 py-1.5 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[11px] transition-all flex items-center gap-1.5 shadow-lg shadow-[#d1f107]/10 font-mono"
        >
          <span className="material-symbols-outlined text-[15px]">check_circle</span>
          <span>Aceptar y aplicar</span>
        </button>
      </div>
    </div>
  );
}
