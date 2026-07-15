import { useState, useRef } from "react";
import { useProjectsStore, type Project } from "../../store/projectsStore";

interface ProjectEditorProps {
  project: Project | null;
  onClose: () => void;
}

function isTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI__" in window;
}

export default function ProjectEditor({ project, onClose }: ProjectEditorProps) {
  const createProject = useProjectsStore((s) => s.createProject);
  const updateProject = useProjectsStore((s) => s.updateProject);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const [name, setName] = useState(project?.name || "");
  const [rootPath, setRootPath] = useState(project?.rootPath || "");
  const [instructions, setInstructions] = useState(project?.instructions || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [dragging, setDragging] = useState(false);
  const pathInputRef = useRef<HTMLInputElement>(null);

  const handleSelectFolder = async () => {
    if (isTauri()) {
      try {
        const { open } = await import("@tauri-apps/plugin-dialog");
        const selected = await open({ directory: true });
        if (selected) {
          setRootPath(selected as string);
          setName((prev) => prev || (selected as string).split(/[/\\]/).pop() || "Proyecto");
        }
      } catch {
        // dialog cancelled
      }
    } else {
      pathInputRef.current?.focus();
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const items = e.dataTransfer.items;
    if (items?.length) {
      const entry = items[0].webkitGetAsEntry?.();
      if (entry?.isDirectory) {
        const path = entry.name;
        setRootPath(path);
        setName((prev) => prev || path);
      }
    }
  };

  const handlePathChange = (value: string) => {
    setRootPath(value);
    if (!name && value) {
      const folderName = value.split(/[/\\]/).pop();
      if (folderName) setName(folderName);
    }
  };

  const handleSave = async () => {
    if (!name.trim()) return;
    setLoading(true);
    setError("");
    try {
      if (project) {
        await updateProject(project.id, {
          name: name.trim(),
          rootPath: rootPath.trim() || undefined,
          instructionsMd: instructions.trim() || undefined,
        });
      } else {
        const id = await createProject(
          name.trim(),
          rootPath.trim() || undefined,
          instructions.trim() || undefined,
        );
        if (!id) {
          setError("Error al crear el proyecto. ¿El servidor está corriendo?");
          setLoading(false);
          return;
        }
        setActiveProject(id);
      }
      onClose();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Error al guardar el proyecto");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="bg-[#1e1e1e] border border-white/10 rounded-2xl w-full max-w-lg mx-4 overflow-hidden shadow-2xl shadow-black/50 animate-in">
        <div className="flex items-center justify-between px-6 pt-6 pb-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#2a2a2a] flex items-center justify-center">
              <span className="material-symbols-outlined text-[#c8e64a] text-[22px]">
                {project ? "edit" : "add_folder"}
              </span>
            </div>
            <div>
              <h2 className="text-[15px] font-semibold text-white">
                {project ? "Editar proyecto" : "Nuevo proyecto"}
              </h2>
              <p className="text-[12px] text-white/40">
                {project ? "Modificá los datos del proyecto" : "Configurá un nuevo proyecto para analizar"}
              </p>
            </div>
          </div>
          <button
            className="w-8 h-8 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined text-[20px]">close</span>
          </button>
        </div>

        <div className="px-6 pb-6 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium text-white/40 uppercase tracking-wider">
              Nombre del proyecto
            </label>
            <input
              className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-4 py-3 text-white text-[14px] outline-none focus:border-[#c8e64a]/50 focus:ring-1 focus:ring-[#c8e64a]/20 transition-all placeholder:text-white/25"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ej: Mi proyecto"
              autoFocus
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium text-white/40 uppercase tracking-wider">
              Carpeta del proyecto
            </label>
            <div
              className={`relative rounded-xl border-2 border-dashed transition-all ${
                dragging
                  ? "border-[#c8e64a]/60 bg-[#c8e64a]/5"
                  : rootPath
                    ? "border-[#c8e64a]/30 bg-[#c8e64a]/5"
                    : "border-white/10 bg-[#2a2a2a] hover:border-white/20 hover:bg-[#2a2a2a]/80"
              }`}
              onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
              onDragLeave={() => setDragging(false)}
              onDrop={handleDrop}
            >
              {rootPath ? (
                <div className="flex items-center gap-3 px-4 py-3">
                  <div className="w-10 h-10 rounded-lg bg-[#c8e64a]/15 flex items-center justify-center shrink-0">
                    <span className="material-symbols-outlined text-[#c8e64a] text-[20px]">
                      folder
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-[14px] text-white font-medium truncate">
                      {rootPath.split(/[/\\]/).pop()}
                    </div>
                    <div className="text-[11px] text-white/40 truncate">
                      {rootPath}
                    </div>
                  </div>
                  <button
                    className="w-7 h-7 rounded-lg flex items-center justify-center text-white/40 hover:text-red-400 hover:bg-red-400/10 transition-colors shrink-0"
                    onClick={() => { setRootPath(""); setName(""); }}
                    type="button"
                  >
                    <span className="material-symbols-outlined text-[16px]">close</span>
                  </button>
                </div>
              ) : (
                <button
                  className="w-full flex flex-col items-center gap-2 px-4 py-6 text-center"
                  onClick={handleSelectFolder}
                  type="button"
                >
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center transition-colors ${
                    dragging ? "bg-[#c8e64a]/20" : "bg-white/5"
                  }`}>
                    <span className={`material-symbols-outlined text-[28px] transition-colors ${
                      dragging ? "text-[#c8e64a]" : "text-white/30"
                    }`}>
                      {dragging ? "cloud_upload" : "folder_open"}
                    </span>
                  </div>
                  <div>
                    <p className="text-[14px] text-white">
                      {dragging ? "Soltá la carpeta aquí" : "Arrastrá una carpeta o hacé clic para seleccionar"}
                    </p>
                    <p className="text-[12px] text-white/30 mt-0.5">
                      {isTauri()
                        ? "Se abrirá el explorador de archivos"
                        : "Escribí la ruta en el campo de abajo"}
                    </p>
                  </div>
                </button>
              )}
            </div>
            <input
              ref={pathInputRef}
              className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-4 py-3 text-white text-[14px] outline-none focus:border-[#c8e64a]/50 focus:ring-1 focus:ring-[#c8e64a]/20 transition-all placeholder:text-white/25"
              value={rootPath}
              onChange={(e) => handlePathChange(e.target.value)}
              placeholder="O escribí la ruta: C:\Users\...\mi-proyecto"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium text-white/40 uppercase tracking-wider">
              Instrucciones del proyecto
            </label>
            <textarea
              className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-4 py-3 text-white text-[14px] outline-none focus:border-[#c8e64a]/50 focus:ring-1 focus:ring-[#c8e64a]/20 transition-all resize-none placeholder:text-white/25"
              rows={3}
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
              placeholder="Contexto adicional para que Ozy entienda el proyecto..."
            />
          </div>

          {error && (
            <div className="flex items-center gap-2 text-[13px] text-red-400 bg-red-400/10 rounded-xl p-3">
              <span className="material-symbols-outlined text-[18px]">error</span>
              {error}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between px-6 py-4 border-t border-white/10 bg-[#1a1a1a]">
          <button
            className="px-4 py-2 text-[13px] text-white/40 hover:text-white rounded-lg hover:bg-white/10 transition-colors"
            onClick={onClose}
          >
            Cancelar
          </button>
          <button
            className="flex items-center gap-2 px-5 py-2.5 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-all text-[13px] font-semibold disabled:opacity-30 disabled:cursor-not-allowed shadow-sm shadow-[#c8e64a]/20"
            onClick={handleSave}
            disabled={!name.trim() || loading}
          >
            {loading ? (
              <>
                <span className="material-symbols-outlined text-[18px] animate-spin">progress_activity</span>
                Guardando...
              </>
            ) : (
              <>
                <span className="material-symbols-outlined text-[18px]">
                  {project ? "check" : "add"}
                </span>
                {project ? "Guardar cambios" : "Crear proyecto"}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
