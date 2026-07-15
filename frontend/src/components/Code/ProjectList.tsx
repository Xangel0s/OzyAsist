import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";

interface ProjectListProps {
  onNewProject: () => void;
  onSelectProject: (projectId: string) => void;
}

export default function ProjectList({ onNewProject, onSelectProject }: ProjectListProps) {
  const projects = useProjectsStore((s) => s.projects);
  const loading = useProjectsStore((s) => s.loading);
  const deleteProject = useProjectsStore((s) => s.deleteProject);
  const [search, setSearch] = useState("");

  const filtered = projects.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase())
  );

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    await deleteProject(id);
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-[#1a1a1a] overflow-y-auto">
      <div className="max-w-[800px] mx-auto w-full px-6 py-12">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-[20px] font-semibold text-white">
              Proyectos de código
            </h1>
            <p className="text-[13px] text-white/40 mt-1">
              Selecciona un proyecto para analizar su código con Ozy.
            </p>
          </div>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-colors text-[13px] font-semibold"
            onClick={onNewProject}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nuevo proyecto
          </button>
        </div>

        <div className="relative mb-6">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-white/30 text-[20px]">
            search
          </span>
          <input
            className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl py-2.5 pl-10 pr-4 text-white placeholder:text-white/25 outline-none focus:border-[#c8e64a]/50 transition-colors text-[14px]"
            placeholder="Buscar proyectos..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {loading &&
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="bg-[#1e1e1e] rounded-xl border border-white/10 p-5 animate-pulse">
                <div className="w-10 h-10 rounded-lg bg-white/5 mb-3" />
                <div className="h-4 w-32 bg-white/5 rounded mb-2" />
                <div className="h-3 w-48 bg-white/5 rounded" />
              </div>
            ))}

          {!loading &&
            filtered.map((project) => (
              <button
                key={project.id}
                className="bg-[#1e1e1e] rounded-xl border border-white/10 p-5 text-left hover:bg-[#252525] hover:border-white/15 transition-all relative group"
                onClick={() => onSelectProject(project.id)}
              >
                <div className="w-10 h-10 rounded-lg bg-[#c8e64a]/15 flex items-center justify-center mb-3">
                  <span className="material-symbols-outlined text-[#c8e64a] text-[20px]">code</span>
                </div>
                <h3 className="text-[14px] font-medium text-white mb-1 truncate">
                  {project.name}
                </h3>
                <p className="text-[12px] text-white/40 mb-3 truncate">
                  {project.rootPath || "Sin ruta"}
                </p>
                <div className="flex items-center gap-3">
                  <span className={`text-[10px] font-medium px-2 py-0.5 rounded-full ${
                    project.permissionLevel === "trusted"
                      ? "bg-[#c8e64a]/10 text-[#c8e64a]"
                      : project.permissionLevel === "read_only"
                      ? "bg-blue-400/10 text-blue-400"
                      : "bg-amber-400/10 text-amber-400"
                  }`}>
                    {project.permissionLevel || "sandboxed"}
                  </span>
                  <span className="text-[11px] text-white/30">
                    {new Date(project.createdAt).toLocaleDateString()}
                  </span>
                </div>
                <button
                  className="absolute top-3 right-3 p-1.5 rounded-lg text-white/20 hover:text-red-400 hover:bg-red-400/10 opacity-0 group-hover:opacity-100 transition-all"
                  onClick={(e) => handleDelete(project.id, e)}
                  title="Eliminar proyecto"
                >
                  <span className="material-symbols-outlined text-[16px]">delete</span>
                </button>
              </button>
            ))}

          {!loading && filtered.length === 0 && (
            <div className="col-span-full flex flex-col items-center justify-center py-16 text-white/30">
              <span className="material-symbols-outlined text-[48px] mb-4">code_off</span>
              <p className="text-[15px]">
                {projects.length === 0 ? "No hay proyectos todavía" : "Sin resultados para esta búsqueda"}
              </p>
              {projects.length === 0 && (
                <button
                  className="mt-4 px-4 py-2 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-colors text-[13px] font-semibold"
                  onClick={onNewProject}
                >
                  Analizar mi primer proyecto
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
