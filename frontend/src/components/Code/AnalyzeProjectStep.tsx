import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import { api } from "../../services/api";

const permissionLevels = [
  { value: "read_only", label: "Lectura", icon: "visibility" },
  { value: "sandboxed", label: "Sandbox", icon: "shield" },
  { value: "trusted", label: "Total", icon: "verified_user" },
];

export default function AnalyzeProjectStep({ onDone }: { onDone?: (projectId: string) => void }) {
  const createProject = useProjectsStore((s) => s.createProject);
  const [path, setPath] = useState("");
  const [permission, setPermission] = useState("sandboxed");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [progress, setProgress] = useState("");

  const folderName = path.replace(/[\\/]+$/, "").split(/[\\/]/).pop() || "";

  const handleAnalyze = async () => {
    if (!path.trim()) return;
    setLoading(true);
    setError("");
    setProgress("Creando proyecto...");

    try {
      const projectId = await createProject(
        folderName || "Proyecto",
        path.trim(),
        undefined,
        permission,
      );
      if (!projectId) {
        setError("Error al crear el proyecto. Verificá que el servidor esté corriendo.");
        return;
      }

      setProgress("Indexando archivos...");
      try {
        const result = await api.projects.index(projectId);
        setProgress(`Indexación completa: ${result.files} archivos, ${result.imports} importaciones`);
      } catch {
        setProgress("Proyecto creado (indexación opcional no disponible)");
      }

      await new Promise((r) => setTimeout(r, 500));
      onDone?.(projectId);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Error al analizar");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center bg-[#1a1a1a] p-6">
      <div className="w-full max-w-[340px]">
        <div className="text-center mb-8">
          <div className="w-14 h-14 rounded-2xl bg-[#2a2a2a] flex items-center justify-center mx-auto mb-4">
            <img src="/ozybaselogo.png" alt="Ozy" className="w-8 h-8 object-contain" />
          </div>
          <h2 className="text-[18px] font-semibold text-white mb-2">Analizar proyecto</h2>
          <p className="text-[13px] text-white/40">Ingresá la ruta completa de tu proyecto para analizarlo.</p>
        </div>

        <div className="bg-[#1e1e1e] rounded-xl border border-white/10 p-4 flex flex-col gap-4">
          <div>
            <label className="text-[11px] font-medium text-white/40 uppercase tracking-wider mb-2 block">
              Ruta del proyecto
            </label>

            {path && folderName && (
              <div className="flex items-center gap-2 bg-[#c8e64a]/10 rounded-lg px-3 py-2.5 mb-2">
                <span className="material-symbols-outlined text-[#c8e64a] text-[18px]">folder</span>
                <span className="flex-1 text-[13px] text-white font-medium truncate">{folderName}</span>
                <button
                  className="w-6 h-6 rounded flex items-center justify-center text-white/40 hover:text-red-400 hover:bg-red-400/10 transition-colors shrink-0"
                  onClick={() => { setPath(""); setError(""); setProgress(""); }}
                  type="button"
                >
                  <span className="material-symbols-outlined text-[14px]">close</span>
                </button>
              </div>
            )}

            <input
              className="w-full bg-[#2a2a2a] border border-white/10 rounded-lg px-3 py-2.5 text-white text-[13px] outline-none focus:border-[#c8e64a]/50 transition-all placeholder:text-white/25 font-mono"
              value={path}
              onChange={(e) => { setPath(e.target.value); setError(""); }}
              placeholder="C:\Users\Lenovo\Documents\mi-proyecto"
              spellCheck={false}
            />
            <p className="text-[11px] text-white/25 mt-1.5">
              Ruta absoluta en tu disco local
            </p>
          </div>

          <div>
            <label className="text-[11px] font-medium text-white/40 uppercase tracking-wider mb-2 block">
              Permisos
            </label>
            <div className="grid grid-cols-3 gap-2">
              {permissionLevels.map((lvl) => (
                <button
                  key={lvl.value}
                  className={`flex flex-col items-center gap-1 py-2.5 px-2 rounded-lg border transition-all text-[11px] font-medium ${
                    permission === lvl.value
                      ? "border-[#c8e64a] bg-[#c8e64a]/10 text-white"
                      : "border-white/10 bg-[#2a2a2a] text-white/50 hover:bg-[#333]"
                  }`}
                  onClick={() => setPermission(lvl.value)}
                >
                  <span className={`material-symbols-outlined text-[18px] ${
                    permission === lvl.value ? "text-[#c8e64a]" : "text-white/40"
                  }`}>
                    {lvl.icon}
                  </span>
                  {lvl.label}
                </button>
              ))}
            </div>
          </div>

          {error && (
            <div className="flex items-center gap-2 text-[12px] text-red-400 bg-red-400/10 rounded-lg p-2.5">
              <span className="material-symbols-outlined text-[16px]">error</span>
              {error}
            </div>
          )}

          {progress && !error && (
            <div className="flex items-center gap-2 text-[12px] text-[#c8e64a] bg-[#c8e64a]/10 rounded-lg p-2.5">
              <span className="material-symbols-outlined text-[16px] animate-spin">progress_activity</span>
              {progress}
            </div>
          )}

          <button
            className="w-full flex items-center justify-center gap-2 py-2.5 bg-[#c8e64a] text-[#1a1a1a] rounded-lg hover:bg-[#b8d63a] transition-all text-[13px] font-semibold disabled:opacity-30 disabled:cursor-not-allowed"
            disabled={!path.trim() || loading}
            onClick={handleAnalyze}
          >
            {loading ? (
              <>
                <span className="material-symbols-outlined text-[16px] animate-spin">progress_activity</span>
                Analizando...
              </>
            ) : (
              <>
                <span className="material-symbols-outlined text-[16px]">search</span>
                Analizar
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}