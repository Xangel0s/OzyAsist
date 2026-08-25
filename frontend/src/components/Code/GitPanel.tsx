import { useState, useEffect, useCallback } from "react";
import { useToastStore } from "../../store/toastStore";
import { useProjectsStore } from "../../store/projectsStore";
import { api, type GitFileItem } from "../../services/api";

export default function GitPanel() {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === activeProjectId);
  const toast = useToastStore((s) => s.show);

  const [loading, setLoading] = useState(false);
  const [initialized, setInitialized] = useState(true);
  const [branch, setBranch] = useState("main");
  const [ahead, setAhead] = useState(0);
  const [behind, setBehind] = useState(0);
  const [files, setFiles] = useState<GitFileItem[]>([]);
  const [totalAdded, setTotalAdded] = useState(0);
  const [totalDeleted, setTotalDeleted] = useState(0);

  const [isSyncing, setIsSyncing] = useState(false);
  const [isInitializing, setIsInitializing] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isCommitting, setIsCommitting] = useState(false);
  const [commitMessage, setCommitMessage] = useState("");
  const [highlights, setHighlights] = useState<string[]>([]);

  const fetchGitStatus = useCallback(async () => {
    if (!activeProjectId) return;
    setLoading(true);
    try {
      const res = await api.git.status(activeProjectId);
      setInitialized(res.initialized);
      setBranch(res.branch || "main");
      setAhead(res.ahead || 0);
      setBehind(res.behind || 0);
      setFiles(res.files || []);
      setTotalAdded(res.totalAdded || 0);
      setTotalDeleted(res.totalDeleted || 0);
    } catch (err: unknown) {
      // If error or endpoint not reachable
      console.warn("Failed to fetch git status:", err);
    } finally {
      setLoading(false);
    }
  }, [activeProjectId]);

  useEffect(() => {
    fetchGitStatus();
  }, [fetchGitStatus]);

  const handleInitRepo = async () => {
    if (!activeProjectId) return;
    setIsInitializing(true);
    try {
      const res = await api.git.init(activeProjectId);
      toast(res.message || "Repositorio Git inicializado", "success");
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al inicializar git", "error");
    } finally {
      setIsInitializing(false);
    }
  };

  const handleSync = async () => {
    if (!activeProjectId) return;
    setIsSyncing(true);
    try {
      const res = await api.git.sync(activeProjectId);
      toast(res.message || "Repositorio sincronizado", "success");
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al sincronizar", "error");
    } finally {
      setIsSyncing(false);
    }
  };

  const handleToggleStage = async (file: GitFileItem) => {
    if (!activeProjectId) return;
    try {
      if (file.staged) {
        await api.git.unstage(activeProjectId, [file.path]);
      } else {
        await api.git.stage(activeProjectId, [file.path]);
      }
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al cambiar stage", "error");
    }
  };

  const handleStageAll = async () => {
    if (!activeProjectId) return;
    try {
      await api.git.stage(activeProjectId, [], true);
      toast("Todos los archivos preparados para commit", "info");
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al preparar archivos", "error");
    }
  };

  const handleUnstageAll = async () => {
    if (!activeProjectId) return;
    try {
      await api.git.unstage(activeProjectId, [], true);
      toast("Archivos retirados de stage", "info");
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al desmarcar archivos", "error");
    }
  };

  const handleGenerateCommit = async () => {
    if (!activeProjectId) return;
    setIsGenerating(true);
    try {
      const res = await api.git.generateMsg(activeProjectId);
      if (res.message) {
        setCommitMessage(res.message);
        setHighlights(res.highlights || []);
        toast("Mensaje de commit generado por IA", "success");
      }
    } catch {
      setCommitMessage("feat: update workspace files");
      toast("Mensaje sugerido creado", "info");
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCommit = async (andSync = false) => {
    if (!activeProjectId || !commitMessage.trim()) return;
    setIsCommitting(true);
    try {
      // If no files are staged, stage all first
      if (stagedFiles.length === 0 && files.length > 0) {
        await api.git.stage(activeProjectId, [], true);
      }
      const res = await api.git.commit(activeProjectId, commitMessage.trim(), andSync);
      toast(res.message || "Commit completado", "success");
      setCommitMessage("");
      setHighlights([]);
      await fetchGitStatus();
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Error al hacer commit", "error");
    } finally {
      setIsCommitting(false);
    }
  };

  const stagedFiles = files.filter((f) => f.staged);
  const unstagedFiles = files.filter((f) => !f.staged);

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "A":
        return <span className="w-4 h-4 rounded bg-emerald-500/20 text-emerald-400 font-bold flex items-center justify-center text-[10px]">A</span>;
      case "M":
        return <span className="w-4 h-4 rounded bg-amber-500/20 text-amber-400 font-bold flex items-center justify-center text-[10px]">M</span>;
      case "D":
        return <span className="w-4 h-4 rounded bg-rose-500/20 text-rose-400 font-bold flex items-center justify-center text-[10px]">D</span>;
      default:
        return <span className="w-4 h-4 rounded bg-sky-500/20 text-sky-400 font-bold flex items-center justify-center text-[10px]">U</span>;
    }
  };

  if (!initialized) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-6 text-center bg-[#181818] text-xs font-sans select-none">
        <div className="w-12 h-12 rounded-2xl bg-[#d1f107]/10 border border-[#d1f107]/20 flex items-center justify-center text-[#d1f107] mb-4 shadow-lg shadow-[#d1f107]/5">
          <span className="material-symbols-outlined text-[28px]">source_environment</span>
        </div>
        <h3 className="text-[14px] font-semibold text-white mb-1.5 font-mono">
          Repositorio Git no inicializado
        </h3>
        <p className="text-[12px] text-zinc-400 max-w-xs mb-5 leading-relaxed">
          El directorio de este proyecto no contiene control de versiones Git activo (<code className="text-[#d1f107]">.git</code>).
        </p>

        <button
          onClick={handleInitRepo}
          disabled={isInitializing}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[12px] font-mono transition-all shadow-lg shadow-[#d1f107]/10 disabled:opacity-50"
        >
          <span className={`material-symbols-outlined text-[16px] ${isInitializing ? "animate-spin" : ""}`}>
            {isInitializing ? "sync" : "add_circle"}
          </span>
          <span>{isInitializing ? "Inicializando..." : "Inicializar Git (git init)"}</span>
        </button>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-[#181818] text-xs font-sans select-none">
      {/* Branch & Sync Header */}
      <div className="p-3 border-b border-white/10 flex items-center justify-between bg-[#202020]">
        <div className="flex items-center gap-2 text-white/90 font-mono">
          <span className="material-symbols-outlined text-[#d1f107] text-[18px]">fork_right</span>
          <span className="font-semibold text-[13px]">{branch}</span>
          <span className="text-[11px] text-white/40">({activeProject?.name || "Proyecto"})</span>
          {(ahead > 0 || behind > 0) && (
            <span className="text-[10px] text-lime-400 font-mono bg-lime-500/10 px-1.5 py-0.5 rounded border border-lime-500/20">
              ↑{ahead} ↓{behind}
            </span>
          )}
          {(totalAdded > 0 || totalDeleted > 0) && (
            <span className="text-[10px] font-mono flex items-center gap-1 bg-white/5 px-1.5 py-0.5 rounded border border-white/5">
              {totalAdded > 0 && <span className="text-emerald-400 font-bold">+{totalAdded}</span>}
              {totalDeleted > 0 && <span className="text-rose-400 font-bold">-{totalDeleted}</span>}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={fetchGitStatus}
            disabled={loading}
            className="w-7 h-7 rounded-lg flex items-center justify-center bg-white/5 hover:bg-white/10 text-white/60 hover:text-white transition-colors border border-white/5"
            title="Refrescar estado Git"
          >
            <span className={`material-symbols-outlined text-[15px] ${loading ? "animate-spin" : ""}`}>
              refresh
            </span>
          </button>

          <button
            onClick={handleSync}
            disabled={isSyncing}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-white/5 hover:bg-white/10 text-white/80 text-[11px] font-mono transition-colors border border-white/10"
            title="Fetch, Pull y Push con el remoto"
          >
            <span className={`material-symbols-outlined text-[14px] text-[#d1f107] ${isSyncing ? "animate-spin" : ""}`}>
              sync
            </span>
            <span>sync</span>
          </button>
        </div>
      </div>

      {/* Files Diff List */}
      <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-3 min-h-0 scrollbar-thin">
        {files.length === 0 && !loading && (
          <div className="flex-1 flex flex-col items-center justify-center text-center py-12 text-zinc-500">
            <span className="material-symbols-outlined text-[28px] text-zinc-600 mb-2">check_circle</span>
            <span className="text-[12px] font-mono">El árbol de trabajo está limpio</span>
            <span className="text-[11px] text-zinc-600 mt-0.5">No hay cambios pendientes</span>
          </div>
        )}

        {/* Staged Section */}
        {stagedFiles.length > 0 && (
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between text-[11px] font-semibold text-white/50 uppercase tracking-wider px-1 pb-1 font-mono">
              <span>Staged ({stagedFiles.length})</span>
              <button
                onClick={handleUnstageAll}
                className="text-[10px] text-zinc-400 hover:text-white normal-case hover:underline"
              >
                unstage all
              </button>
            </div>

            <div className="flex flex-col gap-1">
              {stagedFiles.map((file) => (
                <div
                  key={file.path}
                  className="group flex items-center justify-between p-2 rounded-lg bg-[#1e1e1e] hover:bg-[#252525] border border-white/5 transition-all text-[11px] font-mono"
                >
                  <div
                    className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer"
                    onClick={() => {
                      window.dispatchEvent(
                        new CustomEvent("switch-project-tab", { detail: { tab: "diffs", path: file.path } })
                      );
                    }}
                  >
                    {getStatusBadge(file.status)}
                    <span className="text-white/90 truncate">{file.path}</span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {(file.added > 0 || file.deleted > 0) && (
                      <div className="flex items-center gap-1 text-[10px]">
                        {file.added > 0 && <span className="text-emerald-400 font-bold">+{file.added}</span>}
                        {file.deleted > 0 && <span className="text-rose-400 font-bold">-{file.deleted}</span>}
                      </div>
                    )}
                    <button
                      onClick={() => handleToggleStage(file)}
                      className="text-zinc-500 hover:text-rose-400 transition-colors p-1"
                      title="Quitar de stage"
                    >
                      <span className="material-symbols-outlined text-[15px]">remove</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Unstaged / Modified Section */}
        {unstagedFiles.length > 0 && (
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between text-[11px] font-semibold text-white/50 uppercase tracking-wider px-1 pb-1 font-mono">
              <span>Cambios ({unstagedFiles.length})</span>
              <button
                onClick={handleStageAll}
                className="text-[10px] text-lime-400 hover:underline normal-case"
              >
                stage all
              </button>
            </div>

            <div className="flex flex-col gap-1">
              {unstagedFiles.map((file) => (
                <div
                  key={file.path}
                  className="group flex items-center justify-between p-2 rounded-lg bg-[#1a1a1a] hover:bg-[#222222] border border-white/5 transition-all text-[11px] font-mono"
                >
                  <div
                    className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer"
                    onClick={() => {
                      window.dispatchEvent(
                        new CustomEvent("switch-project-tab", { detail: { tab: "diffs", path: file.path } })
                      );
                    }}
                  >
                    {getStatusBadge(file.status)}
                    <span className="text-white/80 truncate">{file.path}</span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {(file.added > 0 || file.deleted > 0) && (
                      <div className="flex items-center gap-1 text-[10px]">
                        {file.added > 0 && <span className="text-emerald-400 font-bold">+{file.added}</span>}
                        {file.deleted > 0 && <span className="text-rose-400 font-bold">-{file.deleted}</span>}
                      </div>
                    )}
                    <button
                      onClick={() => handleToggleStage(file)}
                      className="text-zinc-500 hover:text-lime-400 transition-colors p-1"
                      title="Preparar archivo (Stage)"
                    >
                      <span className="material-symbols-outlined text-[15px]">add</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Commit Box Footer */}
      <div className="p-3 border-t border-white/10 bg-[#202020] flex flex-col gap-2.5">
        {highlights.length > 0 && (
          <div className="p-2 rounded-lg bg-white/5 border border-white/5 text-[11px] text-white/70 font-sans">
            <div className="text-white/40 text-[10px] uppercase font-bold tracking-wider mb-1 font-mono">
              Puntos clave
            </div>
            <ul className="list-disc list-inside space-y-0.5">
              {highlights.map((h, i) => (
                <li key={i} className="truncate">
                  {h}
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="relative">
          <input
            type="text"
            value={commitMessage}
            onChange={(e) => setCommitMessage(e.target.value)}
            placeholder="Mensaje de commit..."
            className="w-full bg-[#181818] border border-white/10 rounded-xl px-3 py-2 text-white placeholder-white/30 text-[11px] font-mono focus:outline-none focus:border-[#d1f107]/50"
          />
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleGenerateCommit}
            disabled={isGenerating || files.length === 0}
            className="flex items-center gap-1 px-3 py-2 rounded-xl bg-white/5 hover:bg-white/10 text-white/80 text-[11px] font-mono border border-white/10 transition-colors disabled:opacity-40"
            title="Generar mensaje con IA a partir del diff"
          >
            <span className={`material-symbols-outlined text-[15px] text-[#d1f107] ${isGenerating ? "animate-spin" : ""}`}>
              {isGenerating ? "sync" : "auto_fix_high"}
            </span>
            <span>generate</span>
          </button>

          <button
            onClick={() => handleCommit(false)}
            disabled={isCommitting || !commitMessage.trim()}
            className="flex-1 py-2 rounded-xl bg-white/10 hover:bg-white/15 text-white font-semibold text-[11px] font-mono border border-white/10 transition-colors disabled:opacity-40"
          >
            commit
          </button>

          <button
            onClick={() => handleCommit(true)}
            disabled={isCommitting || !commitMessage.trim()}
            className="flex items-center justify-center gap-1 px-3 py-2 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[11px] font-mono transition-colors shadow-lg shadow-[#d1f107]/10 disabled:opacity-40"
            title="Hacer commit y empujar al repositorio remoto"
          >
            <span className="material-symbols-outlined text-[14px]">arrow_upward</span>
            <span>commit & sync</span>
          </button>
        </div>
      </div>
    </div>
  );
}
