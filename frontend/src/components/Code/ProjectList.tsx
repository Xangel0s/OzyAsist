import { useProjectsStore } from "../../store/projectsStore";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";

interface ProjectListProps {
  onNewProject: () => void;
}

export default function ProjectList({ onNewProject }: ProjectListProps) {
  const projects = useProjectsStore((s) => s.projects);
  const loading = useProjectsStore((s) => s.loading);
  const deleteProject = useProjectsStore((s) => s.deleteProject);
  const createChat = useChatStore((s) => s.createChat);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);

  const handleSelect = async (projectId: string) => {
    const chatId = await createChat("code", projectId);
    if (!chatId) {
      useToastStore.getState().show("Error al crear el chat del proyecto", "error");
      return;
    }
    setActiveProject(projectId);
  };

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    await deleteProject(id);
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-surface-deep overflow-y-auto">
      <div className="max-w-container-max mx-auto w-full px-gutter py-8">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-headline-md font-headline-md text-on-surface">
              Proyectos de código
            </h1>
            <p className="text-body-md text-text-muted mt-1">
              Selecciona un proyecto para analizar su código con Ozy.
            </p>
          </div>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={onNewProject}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Analizar otro proyecto
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {loading &&
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="bg-surface-container rounded-xl border border-border-subtle p-6 animate-pulse">
                <div className="w-10 h-10 rounded-lg bg-surface-variant mb-4" />
                <div className="h-5 w-32 bg-surface-variant rounded mb-2" />
                <div className="h-4 w-48 bg-surface-variant rounded mb-4" />
                <div className="h-3 w-16 bg-surface-variant rounded" />
              </div>
            ))}

          {!loading &&
            projects.map((project) => (
              <button
                key={project.id}
                className="bg-surface-container rounded-xl border border-border-subtle p-6 text-left hover:bg-surface-container-high transition-colors relative group"
                onClick={() => handleSelect(project.id)}
              >
                <div className="w-10 h-10 rounded-lg bg-primary-container/20 flex items-center justify-center mb-4">
                  <span className="material-symbols-outlined text-primary-container">code</span>
                </div>
                <h3 className="text-body-lg font-medium text-on-surface mb-1 truncate">
                  {project.name}
                </h3>
                <p className="text-label-caps text-text-muted mb-3 truncate">
                  {project.rootPath || "Sin ruta"}
                </p>
                <div className="flex items-center gap-3">
                  <span className={`text-label-caps px-2 py-0.5 rounded-full ${
                    project.permissionLevel === "trusted"
                      ? "bg-lime-400/10 text-lime-400"
                      : project.permissionLevel === "read_only"
                      ? "bg-blue-400/10 text-blue-400"
                      : "bg-amber-400/10 text-amber-400"
                  }`}>
                    {project.permissionLevel || "sandboxed"}
                  </span>
                  <span className="text-label-caps text-text-muted">
                    {new Date(project.createdAt).toLocaleDateString()}
                  </span>
                </div>
                <button
                  className="absolute top-3 right-3 p-1.5 rounded-lg text-text-muted hover:text-red-400 hover:bg-red-400/10 opacity-0 group-hover:opacity-100 transition-all"
                  onClick={(e) => handleDelete(project.id, e)}
                  title="Eliminar proyecto"
                >
                  <span className="material-symbols-outlined text-[16px]">delete</span>
                </button>
              </button>
            ))}

          {!loading && projects.length === 0 && (
            <div className="col-span-full flex flex-col items-center justify-center py-16 text-text-muted">
              <span className="material-symbols-outlined text-[48px] mb-4">code_off</span>
              <p className="text-body-lg">No hay proyectos todavía</p>
              <button
                className="mt-4 px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
                onClick={onNewProject}
              >
                Analizar mi primer proyecto
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
