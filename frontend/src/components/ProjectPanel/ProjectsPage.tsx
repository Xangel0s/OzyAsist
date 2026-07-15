import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import { useUIStore } from "../../store/uiStore";
import ProjectDetail from "./ProjectDetail";
import ProjectEditor from "./ProjectEditor";

export default function ProjectsPage() {
  const projects = useProjectsStore((s) => s.projects);
  const loading = useProjectsStore((s) => s.loading);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const editingProjectId = useUIStore((s) => s.editingProjectId);
  const setEditingProjectId = useUIStore((s) => s.setEditingProjectId);
  const [search, setSearch] = useState("");
  const [showEditor, setShowEditor] = useState(false);

  const filtered = projects.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  );

  const activeProject = projects.find((p) => p.id === activeProjectId);
  const editingProject = editingProjectId ? projects.find((p) => p.id === editingProjectId) : null;

  if (editingProject) {
    return (
      <div className="flex-1 flex flex-col h-full bg-[#1a1a1a] overflow-y-auto">
        <div className="max-w-container-max mx-auto w-full px-gutter py-8">
          <button
            className="flex items-center gap-2 text-white/40 hover:text-white transition-colors mb-6 text-[14px]"
            onClick={() => setEditingProjectId(null)}
          >
            <span className="material-symbols-outlined text-[18px]">arrow_back</span>
            Cancelar edición
          </button>
          <h1 className="text-[20px] font-semibold text-white mb-6">Editar proyecto</h1>
          <div className="flex flex-col gap-4 max-w-lg">
            <ProjectEditor
              project={editingProject}
              onClose={() => setEditingProjectId(null)}
            />
          </div>
        </div>
      </div>
    );
  }

  if (activeProject) {
    return <ProjectDetail project={activeProject} />;
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-[#1a1a1a] overflow-y-auto">
      <div className="max-w-container-max mx-auto w-full px-gutter py-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-[20px] font-semibold text-white">
            Proyectos
          </h1>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-colors text-[13px] font-semibold"
            onClick={() => setShowEditor(true)}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nuevo proyecto
          </button>
        </div>

        {showEditor && (
          <ProjectEditor
            project={null}
            onClose={() => setShowEditor(false)}
          />
        )}

        <div className="relative mb-8">
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

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {loading && (
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="bg-[#1e1e1e] rounded-xl border border-white/10 p-6 animate-pulse">
                <div className="w-10 h-10 rounded-lg bg-white/5 mb-4" />
                <div className="h-5 w-32 bg-white/5 rounded mb-2" />
                <div className="h-4 w-48 bg-white/5 rounded mb-4" />
                <div className="flex gap-4">
                  <div className="h-3 w-16 bg-white/5 rounded" />
                  <div className="h-3 w-12 bg-white/5 rounded" />
                </div>
              </div>
            ))
          )}
          {!loading && filtered.map((project) => (
            <button
              key={project.id}
              className="bg-[#1e1e1e] rounded-xl border border-white/10 p-6 text-left hover:bg-[#252525] hover:border-white/15 transition-all"
              onClick={() => setActiveProject(project.id)}
            >
              <div className="w-10 h-10 rounded-lg bg-[#c8e64a]/15 flex items-center justify-center mb-4">
                <span className="material-symbols-outlined text-[#c8e64a]">
                  inventory_2
                </span>
              </div>
              <h3 className="text-[15px] font-medium text-white mb-2">
                {project.name}
              </h3>
              <p className="text-white/40 text-[13px] mb-4 line-clamp-2">
                {project.description}
              </p>
              <div className="flex items-center gap-4 text-[11px] font-medium text-white/30 uppercase tracking-wider">
                <span className="flex items-center gap-1">
                  <span className="material-symbols-outlined text-[14px]">description</span>
                  {project.files != null ? `${project.files.length} archivos` : "—"}
                </span>
                <span className="flex items-center gap-1">
                  <span className="material-symbols-outlined text-[14px]">chat</span>
                  {project.chatIds != null ? `${project.chatIds.length} chats` : "—"}
                </span>
              </div>
            </button>
          ))}
          {!loading && filtered.length === 0 && (
            <div className="col-span-full text-center text-white/30 text-[14px] py-12">
              {projects.length === 0
                ? "No hay proyectos aún. Crea uno para empezar."
                : "Sin resultados para esta búsqueda."}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
