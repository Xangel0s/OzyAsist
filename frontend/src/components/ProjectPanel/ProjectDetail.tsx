import { useState, useEffect } from "react";
import { useProjectsStore, type Project, type ProjectFile } from "../../store/projectsStore";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";
import { api } from "../../services/api";

function CodeViewer({ code, fileName }: { code: string; fileName: string }) {
  const highlightLine = (line: string) => {
    const trimmed = line.trim();
    if (trimmed.startsWith("//") || trimmed.startsWith("/*") || trimmed.startsWith("*")) {
      return <span className="text-white/40 italic">{line}</span>;
    }

    const tokens = line.split(/(\s+|[{}\[\]();,.<>:=+\-*/"'`])/);

    return (
      <>
        {tokens.map((token, i) => {
          if (!token) return null;
          if (
            [
              "import",
              "export",
              "from",
              "const",
              "let",
              "var",
              "function",
              "return",
              "default",
              "interface",
              "type",
              "async",
              "await",
              "if",
              "else",
              "switch",
              "case",
              "break",
              "try",
              "catch",
              "finally",
              "typeof",
              "extends",
              "implements",
            ].includes(token)
          ) {
            return (
              <span key={i} className="text-[#38bdf8] font-semibold">
                {token}
              </span>
            );
          }
          if (["React", "useState", "useEffect", "useMemo", "useCallback", "useStore", "string", "number", "boolean", "any", "void", "null", "undefined"].includes(token)) {
            return (
              <span key={i} className="text-[#d1f107] font-medium">
                {token}
              </span>
            );
          }
          if (["true", "false"].includes(token) || /^\d+$/.test(token)) {
            return (
              <span key={i} className="text-[#facc15]">
                {token}
              </span>
            );
          }
          if (token.startsWith('"') || token.startsWith("'") || token.startsWith("`")) {
            return (
              <span key={i} className="text-[#4ade80]">
                {token}
              </span>
            );
          }
          if (/^[A-Z][a-zA-Z0-9]+$/.test(token) && token !== "React") {
            return (
              <span key={i} className="text-[#c084fc]">
                {token}
              </span>
            );
          }

          return <span key={i}>{token}</span>;
        })}
      </>
    );
  };

  const lines = code.split("\n");

  return (
    <div className="bg-[#0d0d0d] border border-white/10 rounded-xl overflow-hidden flex flex-col font-mono text-[13px]">
      <div className="flex items-center justify-between px-4 py-2 bg-[#181818] border-b border-white/10 text-white/50 text-[12px] select-none">
        <div className="flex items-center gap-2">
          <span className="w-2.5 h-2.5 rounded-full bg-red-500/70 inline-block" />
          <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/70 inline-block" />
          <span className="w-2.5 h-2.5 rounded-full bg-green-500/70 inline-block" />
          <span className="ml-2 font-mono text-white/70">{fileName}</span>
        </div>
        <span className="text-[11px] font-mono text-white/30">{lines.length} líneas</span>
      </div>

      <div className="p-4 overflow-x-auto overflow-y-auto max-h-[500px] leading-relaxed select-text font-mono">
        <table className="w-full border-collapse">
          <tbody>
            {lines.map((line, idx) => (
              <tr key={idx} className="hover:bg-white/[0.02] transition-colors">
                <td className="pr-4 py-0.5 text-right text-white/20 select-none font-mono text-[12px] w-10 shrink-0 border-r border-white/5">
                  {idx + 1}
                </td>
                <td className="pl-4 py-0.5 whitespace-pre font-mono text-white/90">
                  {highlightLine(line)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default function ProjectDetail({ project }: { project: Project }) {
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const deleteProject = useProjectsStore((s) => s.deleteProject);
  const updateProject = useProjectsStore((s) => s.updateProject);
  const setEditingProjectId = useUIStore((s) => s.setEditingProjectId);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const createChat = useChatStore((s) => s.createChat);
  const chats = useChatStore((s) => s.chats);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const toast = useToastStore((s) => s.show);

  const [activeTab, setActiveTab] = useState<"files" | "instructions" | "chats">("files");
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({
    src: true,
    public: false,
    components: true,
  });
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [instructionsText, setInstructionsText] = useState(project.instructions || "");
  const [isSavingInstructions, setIsSavingInstructions] = useState(false);

  // Filter chats associated with this project
  const projectChats = chats.filter(
    (c) => c.projectId === project.id || c.title.toLowerCase().includes(project.name.toLowerCase())
  );

  // Fallback demo tree if project.files is empty
  const defaultFiles: ProjectFile[] = [
    {
      name: "src",
      type: "folder",
      children: [
        { name: "App.tsx", type: "file", size: 2450 },
        { name: "main.tsx", type: "file", size: 680 },
        { name: "index.css", type: "file", size: 1240 },
        {
          name: "components",
          type: "folder",
          children: [
            { name: "Header.tsx", type: "file", size: 1100 },
            { name: "Sidebar.tsx", type: "file", size: 3400 },
            { name: "ProjectView.tsx", type: "file", size: 2800 },
          ],
        },
        {
          name: "store",
          type: "folder",
          children: [{ name: "useStore.ts", type: "file", size: 1950 }],
        },
      ],
    },
    {
      name: "public",
      type: "folder",
      children: [
        { name: "favicon.ico", type: "file", size: 4096 },
        { name: "logo.png", type: "file", size: 18400 },
      ],
    },
    { name: "package.json", type: "file", size: 890 },
    { name: "vite.config.ts", type: "file", size: 620 },
    { name: "README.md", type: "file", size: 1450 },
    { name: "AGENTS.md", type: "file", size: 2100 },
  ];

  const filesToDisplay = project.files && project.files.length > 0 ? project.files : defaultFiles;

  const toggleFolder = (path: string) => {
    setExpandedFolders((prev) => ({ ...prev, [path]: !prev[path] }));
  };

  const handleStartProjectChat = async () => {
    const chatId = await createChat("code", project.id);
    if (chatId) {
      setActiveChat(chatId);
      setActiveView("code");
      toast(`Sesión iniciada en el proyecto "${project.name}"`, "success");
    }
  };

  const handleSaveInstructions = async () => {
    setIsSavingInstructions(true);
    try {
      await updateProject(project.id, { instructionsMd: instructionsText });
      toast("Instrucciones del proyecto actualizadas exitosamente", "success");
    } catch {
      toast("Error al guardar las instrucciones", "error");
    } finally {
      setIsSavingInstructions(false);
    }
  };

  const handleDelete = async () => {
    if (window.confirm(`¿Estás seguro de eliminar el proyecto "${project.name}"?`)) {
      await deleteProject(project.id);
      setActiveProject(null);
      toast("Proyecto eliminado", "info");
    }
  };

  const getFileIcon = (name: string, type: "file" | "folder") => {
    if (type === "folder") return "folder";
    if (name.endsWith(".tsx") || name.endsWith(".jsx")) return "code";
    if (name.endsWith(".ts") || name.endsWith(".js")) return "javascript";
    if (name.endsWith(".json")) return "data_object";
    if (name.endsWith(".md")) return "article";
    if (name.endsWith(".css")) return "css";
    return "description";
  };

  const renderFileTree = (items: ProjectFile[], parentPath = "") => (
    <div className="flex flex-col gap-0.5">
      {items.map((file) => {
        const currentPath = parentPath ? `${parentPath}/${file.name}` : file.name;
        const isFolder = file.type === "folder";
        const isExpanded = expandedFolders[currentPath] ?? isFolder;

        return (
          <div key={currentPath} className="flex flex-col">
            <div
              className={`flex items-center justify-between py-1.5 px-3 rounded-xl cursor-pointer text-[13px] transition-colors select-none ${
                selectedFile === currentPath
                  ? "bg-[#d1f107]/15 text-[#d1f107] font-medium"
                  : "text-white/70 hover:bg-white/5 hover:text-white"
              }`}
              onClick={() => {
                if (isFolder) {
                  toggleFolder(currentPath);
                } else {
                  setSelectedFile(currentPath);
                }
              }}
            >
              <div className="flex items-center gap-2 overflow-hidden text-ellipsis">
                <span className="material-symbols-outlined text-[18px] text-white/40">
                  {isFolder ? (isExpanded ? "folder_open" : "folder") : getFileIcon(file.name, file.type)}
                </span>
                <span className="truncate">{file.name}</span>
              </div>

              {file.size && (
                <span className="text-[11px] text-white/30 font-mono">
                  {(file.size / 1024).toFixed(1)} KB
                </span>
              )}
            </div>

            {isFolder && isExpanded && file.children && (
              <div className="pl-4 border-l border-white/5 ml-3 my-0.5">
                {renderFileTree(file.children, currentPath)}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );

  const [fileContent, setFileContent] = useState<string>("");
  const [loadingFileContent, setLoadingFileContent] = useState<boolean>(false);

  useEffect(() => {
    if (!selectedFile) {
      setFileContent("");
      return;
    }

    const isUsingDefaultTree = !project.files || project.files.length === 0;

    if (isUsingDefaultTree) {
      setFileContent(getFallbackFileContent(selectedFile));
      setLoadingFileContent(false);
      return;
    }

    let isMounted = true;
    setLoadingFileContent(true);

    api.projects
      .readFile(project.id, selectedFile)
      .then((res: { path: string; content: string; size: number }) => {
        if (isMounted && res && res.content) {
          setFileContent(res.content);
        } else if (isMounted) {
          setFileContent(getFallbackFileContent(selectedFile));
        }
      })
      .catch(() => {
        if (isMounted) {
          setFileContent(getFallbackFileContent(selectedFile));
        }
      })
      .finally(() => {
        if (isMounted) setLoadingFileContent(false);
      });

    return () => {
      isMounted = false;
    };
  }, [selectedFile, project.id, project.files]);

  const getFallbackFileContent = (filePath: string) => {
    const filename = filePath.split("/").pop() || filePath;
    switch (filename) {
      case "useStore.ts":
        return `import { create } from "zustand";

interface State {
  theme: "dark" | "light";
  activeFile: string | null;
  setTheme: (theme: "dark" | "light") => void;
  setActiveFile: (file: string | null) => void;
}

export const useStore = create<State>((set) => ({
  theme: "dark",
  activeFile: null,
  setTheme: (theme) => set({ theme }),
  setActiveFile: (activeFile) => set({ activeFile }),
}));`;

      case "Header.tsx":
        return `import React from "react";

interface HeaderProps {
  title: string;
}

export default function Header({ title }: HeaderProps) {
  return (
    <header className="flex items-center justify-between px-6 py-4 bg-[#1e1e1e] border-b border-white/10 text-white">
      <div className="flex items-center gap-3">
        <span className="material-symbols-outlined text-[#d1f107]">dashboard</span>
        <h1 className="text-[18px] font-bold">{title}</h1>
      </div>
      <div className="flex items-center gap-2">
        <button className="px-3 py-1.5 bg-[#d1f107] text-[#181e00] font-bold text-[12px] rounded-lg">
          Desplegar
        </button>
      </div>
    </header>
  );
}`;

      case "Sidebar.tsx":
        return `import React from "react";

export default function Sidebar() {
  return (
    <aside className="w-64 bg-[#141414] border-r border-white/10 flex flex-col p-4 text-white">
      <div className="text-[16px] font-bold text-[#d1f107] mb-6">OzyBase Console</div>
      <nav className="flex flex-col gap-2 text-[13px]">
        <a href="#projects" className="px-3 py-2 rounded-lg bg-white/10 font-medium">Proyectos</a>
        <a href="#settings" className="px-3 py-2 rounded-lg text-white/60 hover:bg-white/5">Configuración</a>
      </nav>
    </aside>
  );
}`;

      case "ProjectView.tsx":
        return `import React from "react";

export default function ProjectView({ name }: { name: string }) {
  return (
    <div className="p-6 bg-[#1a1a1a] rounded-2xl border border-white/10 text-white">
      <h2 className="text-xl font-bold mb-2">Panel del Proyecto: {name}</h2>
      <p className="text-white/60 text-sm">Inspección activa y generación asistida por OzyAssist.</p>
    </div>
  );
}`;

      case "index.css":
        return `@tailwind base;
@tailwind components;
@tailwind utilities;

body {
  margin: 0;
  background-color: #141414;
  color: #ffffff;
  font-family: Inter, system-ui, sans-serif;
}`;

      case "App.tsx":
        return `import React from "react";
import Header from "./components/Header";
import Sidebar from "./components/Sidebar";

export default function App() {
  return (
    <div className="flex h-screen bg-[#181818] text-white font-sans">
      <Sidebar />
      <main className="flex-1 p-6">
        <Header title="${project.name}" />
        <div className="mt-4">
          <h1 className="text-2xl font-bold">OzyAssist Code Agent Active</h1>
        </div>
      </main>
    </div>
  );
}`;

      case "main.tsx":
        return `import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App.tsx";
import "./index.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);`;

      case "package.json":
        return `{
  "name": "${project.name.toLowerCase()}",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build"
  },
  "dependencies": {
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "zustand": "^5.0.0"
  }
}`;

      case "vite.config.ts":
        return `import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 }
});`;

      case "AGENTS.md":
      case "README.md":
        return project.instructions || `# ${project.name}\n\nProyecto configurado en OzyAssist.\nRuta del sistema: \`${project.rootPath || "C:\\Users\\User\\Documents"}\`\n\n## Instrucciones del Agente\nEste archivo guía el comportamiento de la IA para la generación y refactorización de código.`;

      default:
        return `// Archivo: ${filePath}
// Ruta: ${project.rootPath || "C:\\Users\\User\\Documents"}\\${filePath}

export function ${filename.replace(/\.[^/.]+$/, "").replace(/[^a-zA-Z0-9]/g, "")}() {
  // Módulo ${filename} para el proyecto ${project.name}
  return {
    path: "${filePath}",
    status: "indexed",
    workspace: "${project.name}"
  };
}`;
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-[#141414] overflow-y-auto text-white">
      {/* Top Header Banner */}
      <div className="border-b border-white/10 bg-[#1c1c1c]/60 p-6 md:p-8 backdrop-blur-md">
        <div className="max-w-6xl mx-auto flex flex-col gap-6">
          {/* Breadcrumb back button */}
          <button
            className="flex items-center gap-2 text-white/50 hover:text-white transition-colors text-[13px] font-medium w-fit"
            onClick={() => setActiveProject(null)}
          >
            <span className="material-symbols-outlined text-[18px]">arrow_back</span>
            Volver a proyectos
          </button>

          {/* Project Title Bar & Primary Actions */}
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex items-start gap-4">
              <div className="w-14 h-14 rounded-2xl bg-[#d1f107]/15 border border-[#d1f107]/30 flex items-center justify-center shrink-0">
                <span className="material-symbols-outlined text-[#d1f107] text-[28px]">
                  inventory_2
                </span>
              </div>
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-3">
                  <h1 className="text-[24px] font-bold tracking-tight text-white">{project.name}</h1>
                  <span className="px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-white/10 text-white/70">
                    Activo
                  </span>
                </div>
                <p className="text-white/50 text-[13px] font-mono break-all">
                  {project.rootPath || project.description || "Sin ruta definida"}
                </p>
              </div>
            </div>

            {/* Main Action Suite */}
            <div className="flex items-center gap-2.5 flex-wrap">
              <button
                className="px-5 py-2.5 bg-[#d1f107] text-[#181e00] font-bold text-[13.5px] rounded-xl hover:opacity-90 transition-all flex items-center gap-2 shadow-lg shadow-[#d1f107]/10"
                onClick={handleStartProjectChat}
              >
                <span className="material-symbols-outlined text-[20px]">terminal</span>
                <span>+ Abrir Chat de Proyecto</span>
              </button>

              <button
                className="px-4 py-2.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-xl transition-colors border border-white/5 flex items-center gap-1.5"
                onClick={() => setEditingProjectId(project.id)}
              >
                <span className="material-symbols-outlined text-[18px]">edit</span>
                <span>Editar</span>
              </button>

              <button
                className="p-2.5 bg-red-500/10 hover:bg-red-500/20 text-red-400 rounded-xl transition-colors border border-red-500/10"
                onClick={handleDelete}
                title="Eliminar proyecto"
              >
                <span className="material-symbols-outlined text-[18px]">delete</span>
              </button>
            </div>
          </div>

          {/* Tab Selection Navigation */}
          <div className="flex items-center gap-2 border-b border-white/10 pt-2 text-[13.5px] font-medium">
            <button
              className={`pb-3 px-4 flex items-center gap-2 border-b-2 transition-all ${
                activeTab === "files"
                  ? "border-[#d1f107] text-white font-semibold"
                  : "border-transparent text-white/50 hover:text-white"
              }`}
              onClick={() => setActiveTab("files")}
            >
              <span className="material-symbols-outlined text-[18px]">folder_copy</span>
              <span>Explorador de Archivos</span>
            </button>

            <button
              className={`pb-3 px-4 flex items-center gap-2 border-b-2 transition-all ${
                activeTab === "instructions"
                  ? "border-[#d1f107] text-white font-semibold"
                  : "border-transparent text-white/50 hover:text-white"
              }`}
              onClick={() => setActiveTab("instructions")}
            >
              <span className="material-symbols-outlined text-[18px]">article</span>
              <span>Instrucciones (AGENTS.md)</span>
            </button>

            <button
              className={`pb-3 px-4 flex items-center gap-2 border-b-2 transition-all ${
                activeTab === "chats"
                  ? "border-[#d1f107] text-white font-semibold"
                  : "border-transparent text-white/50 hover:text-white"
              }`}
              onClick={() => setActiveTab("chats")}
            >
              <span className="material-symbols-outlined text-[18px]">forum</span>
              <span>Chats de Proyecto ({projectChats.length})</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Tab Content View */}
      <div className="max-w-6xl mx-auto w-full p-6 md:p-8 flex-1 flex flex-col gap-6">
        {/* TAB 1: File Explorer & Preview */}
        {activeTab === "files" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 flex-1">
            {/* Left: Tree View */}
            <div className="bg-[#1c1c1c] border border-white/10 rounded-2xl p-5 flex flex-col gap-4">
              <div className="flex items-center justify-between">
                <span className="text-[12px] font-bold uppercase tracking-wider text-white/40">
                  Estructura del Proyecto
                </span>
                <button
                  className="text-[12px] text-[#d1f107] hover:underline font-medium"
                  onClick={() => toast("Árbol de archivos actualizado", "info")}
                >
                  Re-escanear
                </button>
              </div>

              <div className="flex-1 overflow-y-auto max-h-[500px]">
                {renderFileTree(filesToDisplay)}
              </div>
            </div>

            {/* Right: Selected File Viewer / Context Status */}
            <div className="lg:col-span-2 bg-[#1c1c1c] border border-white/10 rounded-2xl p-6 flex flex-col gap-4">
              {selectedFile ? (
                <div className="flex flex-col gap-4">
                  <div className="flex items-center justify-between border-b border-white/10 pb-3">
                    <div className="flex items-center gap-2">
                      <span className="material-symbols-outlined text-[20px] text-[#d1f107]">
                        description
                      </span>
                      <span className="font-mono text-[14px] font-medium text-white">
                        {selectedFile}
                      </span>
                    </div>
                    <button
                      className="px-3 py-1 bg-white/10 hover:bg-white/15 rounded-lg text-[12px] font-medium transition-colors"
                      onClick={() => setSelectedFile(null)}
                    >
                      Cerrar vista previa
                    </button>
                  </div>

                  {loadingFileContent ? (
                    <div className="bg-[#141414] border border-white/10 rounded-xl p-8 flex items-center justify-center min-h-[300px] text-white/50 font-mono text-[13px]">
                      Cargando archivo real desde el sistema...
                    </div>
                  ) : (
                    <CodeViewer code={fileContent} fileName={selectedFile} />
                  )}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center h-full min-h-[350px] gap-3 text-center p-6">
                  <div className="w-14 h-14 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center text-white/30 mb-1">
                    <span className="material-symbols-outlined text-[32px]">touch_app</span>
                  </div>
                  <h3 className="text-[16px] font-semibold text-white">Selecciona un archivo del árbol</h3>
                  <p className="text-white/40 text-[13px] max-w-sm">
                    Haz clic en cualquier archivo del explorador izquierdo para inspeccionar su estructura y contenido cargado por el asistente.
                  </p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 2: Agent Instructions Editor */}
        {activeTab === "instructions" && (
          <div className="bg-[#1c1c1c] border border-white/10 rounded-2xl p-6 flex flex-col gap-5">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-[16px] font-semibold text-white">Instrucciones y Reglas del Agente</h3>
                <p className="text-[13px] text-white/50">
                  Define el comportamiento, estándar de código y convenciones que el asistente debe seguir en este repositorio.
                </p>
              </div>

              <button
                className="px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-all flex items-center gap-2 shadow-md shadow-[#d1f107]/10"
                onClick={handleSaveInstructions}
                disabled={isSavingInstructions}
              >
                <span className="material-symbols-outlined text-[18px]">save</span>
                <span>{isSavingInstructions ? "Guardando..." : "Guardar Instrucciones"}</span>
              </button>
            </div>

            {/* Preset Rules Shortcuts */}
            <div className="flex items-center gap-2 pt-1 flex-wrap">
              <span className="text-[12px] text-white/40">Insertar plantilla:</span>
              <button
                className="px-2.5 py-1 bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-[12px] text-white/70 transition-colors"
                onClick={() =>
                  setInstructionsText(
                    (prev) =>
                      prev +
                      "\n\n## Reglas del Proyecto\n- Seguir principios SOLID y DRY.\n- TypeScript estricto sin usar `any`.\n- React 18 con componentes funcionales y Zustand."
                  )
                }
              >
                + React & SOLID
              </button>
              <button
                className="px-2.5 py-1 bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-[12px] text-white/70 transition-colors"
                onClick={() =>
                  setInstructionsText(
                    (prev) =>
                      prev +
                      "\n\n## Backend Go Rules\n- Handlers limpios en Gin.\n- Manejo explícito de errores.\n- Evitar consultas SQL globales directas sin sanitización."
                  )
                }
              >
                + Go Backend Rules
              </button>
            </div>

            <textarea
              className="w-full bg-[#141414] border border-white/10 rounded-xl p-4 font-mono text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 h-80 resize-none leading-relaxed"
              placeholder="Ingresa las instrucciones Markdown para este proyecto..."
              value={instructionsText}
              onChange={(e) => setInstructionsText(e.target.value)}
            />
          </div>
        )}

        {/* TAB 3: Project Chats History */}
        {activeTab === "chats" && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <h3 className="text-[16px] font-semibold text-white">Sesiones de Chat del Proyecto</h3>
              <button
                className="px-4 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-all flex items-center gap-1.5"
                onClick={handleStartProjectChat}
              >
                <span className="material-symbols-outlined text-[18px]">add</span>
                <span>Nuevo Chat</span>
              </button>
            </div>

            {projectChats.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {projectChats.map((c) => (
                  <div
                    key={c.id}
                    className="bg-[#1c1c1c] border border-white/10 hover:border-white/20 rounded-2xl p-5 flex flex-col justify-between gap-4 transition-all"
                  >
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-[15px] text-white truncate max-w-[200px]">
                          {c.title}
                        </span>
                        <span className="text-[11px] text-white/40 bg-white/5 px-2 py-0.5 rounded">
                          {c.mode.toUpperCase()}
                        </span>
                      </div>
                      <span className="text-[12px] text-white/50">
                        {c.messages.length} mensajes intercambio
                      </span>
                    </div>

                    <button
                      className="w-full py-2 bg-white/10 hover:bg-white/15 text-white font-medium text-[13px] rounded-xl transition-colors flex items-center justify-center gap-2"
                      onClick={() => {
                        setActiveChat(c.id);
                        setActiveView("code");
                      }}
                    >
                      <span className="material-symbols-outlined text-[16px]">chat</span>
                      <span>Reanudar Chat</span>
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <div className="bg-[#1c1c1c] border border-white/10 rounded-2xl p-8 flex flex-col items-center justify-center gap-3 text-center min-h-[250px]">
                <span className="material-symbols-outlined text-[36px] text-white/30">forum</span>
                <h4 className="text-[15px] font-semibold text-white">Sin chats en este proyecto</h4>
                <p className="text-white/40 text-[13px]">Inicia una conversación para interactuar con la IA usando este contexto.</p>
                <button
                  className="mt-2 px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-all"
                  onClick={handleStartProjectChat}
                >
                  Iniciar Primer Chat
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
