import { useProjectsStore, type Project, type ProjectFile } from "../../store/projectsStore";
import { useUIStore } from "../../store/uiStore";

export default function ProjectDetail({
  project,
}: {
  project: Project;
}) {
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const setEditingProjectId = useUIStore((s) => s.setEditingProjectId);

  const renderFiles = (files: ProjectFile[], depth = 0) => (
    <div>
      {files.map((file) => (
        <div key={file.name}>
          <div
            className="flex items-center gap-2 py-1.5 px-2 rounded hover:bg-white/5 cursor-pointer text-[13px] text-white/70"
            style={{ paddingLeft: `${8 + depth * 16}px` }}
          >
            <span className="material-symbols-outlined text-[16px] text-white/30">
              {file.type === "folder" ? "folder" : "description"}
            </span>
            <span>{file.name}</span>
          </div>
          {file.children && renderFiles(file.children, depth + 1)}
        </div>
      ))}
    </div>
  );

  return (
    <div className="flex-1 flex h-full bg-[#1a1a1a] overflow-y-auto">
      <div className="flex-1 max-w-content-max-width mx-auto w-full px-gutter py-8">
        <button
          className="flex items-center gap-2 text-white/40 hover:text-white transition-colors mb-6 text-[14px]"
          onClick={() => setActiveProject(null)}
        >
          <span className="material-symbols-outlined text-[18px]">arrow_back</span>
          Volver a proyectos
        </button>

        <div className="flex items-start gap-4 mb-8">
          <div className="w-12 h-12 rounded-xl bg-[#c8e64a]/15 flex items-center justify-center">
            <span className="material-symbols-outlined text-[#c8e64a] text-[24px]">
              inventory_2
            </span>
          </div>
          <div className="flex-1">
            <h1 className="text-[20px] font-semibold text-white mb-2">
              {project.name}
            </h1>
            <p className="text-white/40 text-[14px]">{project.description}</p>
          </div>
          <button
            className="px-4 py-2 border border-white/10 rounded-xl text-[13px] text-white/50 hover:text-white hover:bg-white/10 transition-colors"
            onClick={() => setEditingProjectId(project.id)}
          >
            Editar
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <div>
            <h2 className="text-[11px] font-medium text-white/40 uppercase tracking-wider mb-4">
              Instrucciones
            </h2>
            <div className="bg-[#1e1e1e] rounded-xl border border-white/10 p-4 text-[14px] text-white/70">
              {project.instructions || "Sin instrucciones definidas."}
            </div>
          </div>
          <div>
            <h2 className="text-[11px] font-medium text-white/40 uppercase tracking-wider mb-4">
              Estructura de archivos
            </h2>
            <div className="bg-[#1e1e1e] rounded-xl border border-white/10 p-4">
              {renderFiles(project.files ?? [])}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
