import { useState } from "react";
import { open } from "@tauri-apps/plugin-dialog";
import { useProjectsStore } from "../../store/projectsStore";
import { api } from "../../services/api";

const permissionLevels = [
  {
    value: "read_only",
    label: "Solo lectura",
    desc: "El agente puede leer archivos pero no escribir ni ejecutar comandos.",
  },
  {
    value: "sandboxed",
    label: "Sandbox",
    desc: "El agente puede leer, escribir y ejecutar comandos dentro del proyecto. Comandos destructivos requieren confirmación.",
  },
  {
    value: "trusted",
    label: "Confianza total",
    desc: "El agente tiene acceso completo sin restricciones. Comandos destructivos se ejecutan sin confirmación.",
  },
];

export default function AnalyzeProjectStep({ onDone }: { onDone?: () => void }) {
  const createProject = useProjectsStore((s) => s.createProject);
  const [path, setPath] = useState("");
  const [name, setName] = useState("");
  const [permission, setPermission] = useState("sandboxed");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSelectFolder = async () => {
    try {
      const selected = await open({ directory: true });
      if (selected) {
        setPath(selected);
        setName(selected.split(/[/\\]/).pop() || "Proyecto");
        setError("");
      }
    } catch {
      // dialog cancelled or not in Tauri (web dev fallback)
    }
  };

  const handleAnalyze = async () => {
    if (!path) return;
    setLoading(true);
    setError("");
    try {
      const projectId = await createProject(name, path, undefined);
      if (!projectId) {
        setError("Error al crear el proyecto. ¿El servidor está corriendo?");
        setLoading(false);
        return;
      }
      // Index the project
      try {
        await api.projects.index(projectId);
      } catch {
        // non-fatal: tree/graph may be incomplete
      }
      onDone?.();
    } catch (e: any) {
      setError(e?.message || "Error al analizar el proyecto");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center bg-surface-deep p-8">
      <div className="max-w-md w-full">
        <div className="text-center mb-8">
          <div className="w-14 h-14 rounded-xl flex items-center justify-center overflow-hidden mx-auto mb-4">
            <img src="/ozybaselogo.png" alt="Ozy" className="w-full h-full object-contain" />
          </div>
          <h2 className="text-headline-md font-headline-md text-on-surface mb-2">
            Analizar proyecto
          </h2>
          <p className="text-body-md text-text-muted">
            Selecciona una carpeta de código para que Ozy entienda su estructura y dependencias.
          </p>
        </div>

        <div className="flex flex-col gap-4">
          <button
            className="w-full flex items-center gap-3 px-4 py-3 bg-surface-container rounded-xl border border-border-subtle text-on-surface hover:bg-surface-container-high transition-colors text-left"
            onClick={handleSelectFolder}
          >
            <span className="material-symbols-outlined text-[24px] text-primary-container">
              folder_open
            </span>
            <div className="flex-1 min-w-0">
              <div className="text-body-md truncate">
                {path || "Seleccionar carpeta del proyecto"}
              </div>
              {path && (
                <div className="text-label-caps text-text-muted mt-0.5 truncate">{path}</div>
              )}
            </div>
          </button>

          <div>
            <label className="text-label-caps text-text-muted mb-2 block">
              Nivel de permiso del agente
            </label>
            <div className="flex flex-col gap-2">
              {permissionLevels.map((lvl) => (
                <button
                  key={lvl.value}
                  className={`flex items-start gap-3 p-3 rounded-xl border text-left transition-colors ${
                    permission === lvl.value
                      ? "border-primary-container bg-primary-container/10"
                      : "border-border-subtle bg-surface-container hover:bg-surface-container-high"
                  }`}
                  onClick={() => setPermission(lvl.value)}
                >
                  <span
                    className={`material-symbols-outlined text-[18px] mt-0.5 ${
                      permission === lvl.value ? "text-primary-container fill" : "text-text-muted"
                    }`}
                  >
                    {permission === lvl.value ? "check_circle" : "radio_button_unchecked"}
                  </span>
                  <div>
                    <div className="text-body-md text-on-surface">{lvl.label}</div>
                    <div className="text-label-caps text-text-muted mt-0.5">{lvl.desc}</div>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {error && (
            <div className="text-body-md text-red-400 bg-red-400/10 rounded-lg p-3">{error}</div>
          )}

          <button
            className="w-full px-4 py-3 bg-primary-container text-on-primary rounded-xl hover:bg-primary-fixed-dim transition-colors text-body-md font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={!path || loading}
            onClick={handleAnalyze}
          >
            {loading ? (
              <span className="flex items-center justify-center gap-2">
                <span className="material-symbols-outlined text-[18px] animate-spin">
                  progress_activity
                </span>
                Analizando estructura del proyecto...
              </span>
            ) : (
              "Analizar proyecto"
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
