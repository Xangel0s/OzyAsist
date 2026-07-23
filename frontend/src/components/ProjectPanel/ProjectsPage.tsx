import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import { useUIStore } from "../../store/uiStore";
import ProjectDetail from "./ProjectDetail";
import ProjectEditor from "./ProjectEditor";

export default function ProjectsPage() {
  const projects = useProjectsStore((s) => s.projects);
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
    <div className="flex-1 flex flex-col h-full bg-[#1a1a1a] p-8 overflow-y-auto min-h-0 text-white">
      <div className="max-w-[850px] mx-auto w-full flex flex-col gap-6">
        {/* Header Title & Action Buttons Bar */}
        <div className="flex items-center justify-between">
          <h1 className="text-[26px] font-bold tracking-tight text-white font-sans">Proyectos</h1>

          <div className="flex items-center gap-2">
            <button
              className="p-2 bg-white/10 hover:bg-white/15 text-white rounded-lg transition-colors flex items-center justify-center"
              onClick={() => setSearch(search ? "" : "a")}
              title="Buscar proyectos"
            >
              <span className="material-symbols-outlined text-[18px]">search</span>
            </button>

            {/* Sort Selector: Ordenar por Última actualización ⌄ */}
            <div className="relative">
              <button className="flex items-center gap-1.5 px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors border border-white/5">
                <span className="text-white/60">Ordenar por</span>
                <span className="font-semibold">Última actualización</span>
                <span className="material-symbols-outlined text-[16px] text-white/60">expand_more</span>
              </button>
            </div>

            <button
              className="px-4 py-1.5 bg-white text-black hover:bg-white/90 text-[13px] font-semibold rounded-lg transition-colors shadow-sm"
              onClick={() => setShowEditor(true)}
            >
              Nuevo proyecto
            </button>
          </div>
        </div>

        {showEditor && (
          <ProjectEditor
            project={null}
            onClose={() => setShowEditor(false)}
          />
        )}

        {/* Projects Grid or Centered Empty State */}
        {filtered.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 pt-4">
            {filtered.map((project) => (
              <button
                key={project.id}
                className="bg-[#1e1e1e] rounded-2xl border border-white/10 p-6 text-left hover:bg-[#252525] hover:border-white/15 transition-all"
                onClick={() => setActiveProject(project.id)}
              >
                <div className="w-10 h-10 rounded-xl bg-[#d1f107]/15 flex items-center justify-center mb-4">
                  <span className="material-symbols-outlined text-[#d1f107]">
                    inventory_2
                  </span>
                </div>
                <h3 className="text-[15px] font-semibold text-white mb-2">
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
          </div>
        ) : (
          /* Centered Empty State Matching Image 3 */
          <div className="flex flex-col items-center justify-center min-h-[420px] gap-4 text-center max-w-md mx-auto pt-8">
            <div className="w-16 h-16 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center text-white/40 mb-2">
              <span className="material-symbols-outlined text-[36px]">grid_view</span>
            </div>

            <h2 className="text-[20px] font-semibold text-white">
              ¿Quieres comenzar un proyecto?
            </h2>

            <p className="text-white/50 text-[13.5px] leading-relaxed">
              Sube materiales, establece instrucciones personalizadas y organiza conversaciones en un solo espacio.
            </p>

            <button
              className="mt-2 px-5 py-2.5 bg-[#2a2a2a] hover:bg-[#333333] border border-white/10 text-white font-medium text-[13.5px] rounded-xl transition-all shadow-sm"
              onClick={() => setShowEditor(true)}
            >
              Nuevo proyecto
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
