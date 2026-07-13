import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import ProjectDetail from "./ProjectDetail";

export default function ProjectsPage() {
  const projects = useProjectsStore((s) => s.projects);
  const loading = useProjectsStore((s) => s.loading);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const createProject = useProjectsStore((s) => s.createProject);
  const [search, setSearch] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState("");
  const [formPath, setFormPath] = useState("");
  const [formInstructions, setFormInstructions] = useState("");

  const filtered = projects.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()),
  );

  const activeProject = projects.find((p) => p.id === activeProjectId);

  const handleCreate = async () => {
    if (!formName.trim()) return;
    const id = await createProject(
      formName.trim(),
      formPath.trim() || undefined,
      formInstructions.trim() || undefined,
    );
    if (id) {
      setFormName("");
      setFormPath("");
      setFormInstructions("");
      setShowForm(false);
    }
  };

  if (activeProject) {
    return <ProjectDetail project={activeProject} />;
  }

  return (
    <div className="flex-1 flex flex-col h-full bg-surface-deep overflow-y-auto">
      <div className="max-w-container-max mx-auto w-full px-gutter py-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-headline-md font-headline-md text-on-surface">
            Proyectos
          </h1>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={() => setShowForm(true)}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nuevo proyecto
          </button>
        </div>

        <div className="relative mb-8">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-text-muted text-[20px]">
            search
          </span>
          <input
            className="w-full bg-surface-container border border-border-subtle rounded-lg py-2.5 pl-10 pr-4 text-on-surface placeholder:text-text-muted outline-none focus:border-primary-container transition-colors text-body-md"
            placeholder="Buscar proyectos..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {showForm && (
          <div className="bg-surface-container rounded-xl border border-border-subtle p-6 mb-8">
            <h2 className="text-body-lg font-medium text-on-surface mb-4">Nuevo proyecto</h2>
            <div className="flex flex-col gap-4">
              <input
                className="w-full bg-surface-deep border border-border-subtle rounded-lg py-2.5 px-4 text-on-surface placeholder:text-text-muted outline-none focus:border-primary-container transition-colors text-body-md"
                placeholder="Nombre del proyecto"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                autoFocus
              />
              <input
                className="w-full bg-surface-deep border border-border-subtle rounded-lg py-2.5 px-4 text-on-surface placeholder:text-text-muted outline-none focus:border-primary-container transition-colors text-body-md"
                placeholder="Ruta del proyecto (opcional)"
                value={formPath}
                onChange={(e) => setFormPath(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
              />
              <textarea
                className="w-full bg-surface-deep border border-border-subtle rounded-lg py-2.5 px-4 text-on-surface placeholder:text-text-muted outline-none focus:border-primary-container transition-colors text-body-md resize-none"
                placeholder="Instrucciones del proyecto (opcional)"
                value={formInstructions}
                onChange={(e) => setFormInstructions(e.target.value)}
                rows={3}
              />
              <div className="flex gap-2 justify-end">
                <button
                  className="px-4 py-2 border border-border-subtle rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
                  onClick={() => { setShowForm(false); setFormName(""); setFormPath(""); setFormInstructions(""); }}
                >
                  Cancelar
                </button>
                <button
                  className="px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
                  onClick={handleCreate}
                >
                  Crear
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {loading && (
            Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="bg-surface-container rounded-xl border border-border-subtle p-6 animate-pulse">
                <div className="w-10 h-10 rounded-lg bg-surface-variant mb-4" />
                <div className="h-5 w-32 bg-surface-variant rounded mb-2" />
                <div className="h-4 w-48 bg-surface-variant rounded mb-4" />
                <div className="flex gap-4">
                  <div className="h-3 w-16 bg-surface-variant rounded" />
                  <div className="h-3 w-12 bg-surface-variant rounded" />
                </div>
              </div>
            ))
          )}
          {!loading && filtered.map((project) => (
            <button
              key={project.id}
              className="bg-surface-container rounded-xl border border-border-subtle p-6 text-left hover:bg-surface-container-high transition-colors"
              onClick={() => setActiveProject(project.id)}
            >
              <div className="w-10 h-10 rounded-lg bg-primary-container/20 flex items-center justify-center mb-4">
                <span className="material-symbols-outlined text-primary-container">
                  inventory_2
                </span>
              </div>
              <h3 className="text-body-lg font-medium text-on-surface mb-2">
                {project.name}
              </h3>
              <p className="text-text-muted text-body-md mb-4 line-clamp-2">
                {project.description}
              </p>
              <div className="flex items-center gap-4 text-label-caps text-text-muted">
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
            <div className="col-span-full text-center text-text-muted text-body-md py-12">
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
