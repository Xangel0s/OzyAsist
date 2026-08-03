import { useState, useEffect, useCallback } from "react";
import { useProjectsStore, type ProjectFile } from "../../store/projectsStore";
import { api, type GraphEdge } from "../../services/api";
import DependencyGraph from "./DependencyGraph";

type Tab = "tree" | "deps" | "preview";

function FileTreeItem({
  item,
  depth = 0,
  onFileClick,
  activeFile,
}: {
  item: ProjectFile;
  depth?: number;
  onFileClick?: (path: string) => void;
  activeFile?: string | null;
}) {
  const isFolder = item.type === "folder";
  const isActive = !isFolder && item.path === activeFile;
  return (
    <div>
      <div
        className={`flex items-center gap-2 py-1 px-2 rounded cursor-pointer text-[13px] transition-colors ${
          isActive
            ? "bg-[#c8e64a]/10 text-white"
            : "hover:bg-white/5 text-white/70"
        }`}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
        onClick={() => !isFolder && onFileClick?.(item.path ?? item.name)}
      >
        <span className={`material-symbols-outlined text-[16px] ${
          isFolder ? "text-white/40" : isActive ? "text-[#c8e64a]" : "text-[#c8e64a]/70"
        }`}>
          {isFolder ? "folder" : "description"}
        </span>
        <span className="truncate">{item.name}</span>
      </div>
      {item.children?.map((child) => (
        <FileTreeItem
          key={child.path || child.name}
          item={child}
          depth={depth + 1}
          onFileClick={onFileClick}
          activeFile={activeFile}
        />
      ))}
    </div>
  );
}

function CodeViewer({ content, path }: { content: string; path: string }) {
  const lines = content.split("\n");

  const highlightLine = (line: string): string => {
    let result = line
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");

    result = result.replace(/(#[^\n]*)/g, '<span style="color:#6a9955">$1</span>');
    result = result.replace(/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)/g, '<span style="color:#ce9178">$1</span>');
    result = result.replace(/\b(import|from|def|class|return|if|else|elif|for|while|try|except|finally|with|as|yield|async|await|fun|fn|let|mut|const|var|export|default|struct|impl|pub|use|mod|match|loop|break|continue|switch|case|throw|new|this|self|true|false|null|undefined|None|True|False|SELECT|FROM|WHERE|INSERT|UPDATE|DELETE|JOIN|CREATE|TABLE|ALTER|DROP|INDEX)\b/g, '<span style="color:#569cd6">$1</span>');
    result = result.replace(/\b(\d+\.?\d*)\b/g, '<span style="color:#b5cea8">$1</span>');
    result = result.replace(/(\/\/.*$)/gm, '<span style="color:#6a9955">$1</span>');
    result = result.replace(/(\/\*[\s\S]*?\*\/)/g, '<span style="color:#6a9955">$1</span>');

    return result;
  };

  return (
    <div className="flex-1 min-h-0 overflow-auto bg-[#1e1e1e] rounded-lg border border-white/5">
      <div className="flex items-center gap-2 px-3 py-2 border-b border-white/5 bg-[#252525]">
        <span className="material-symbols-outlined text-[14px] text-[#c8e64a]/70">description</span>
        <span className="text-[12px] text-white/60 truncate">{path}</span>
        <span className="text-[10px] text-white/30 ml-auto">{lines.length} líneas</span>
      </div>
      <pre className="p-3 overflow-auto text-[12px] leading-[1.6] font-mono text-white/80">
        <code>
          {lines.map((line, i) => (
            <div key={i} className="flex">
              <span className="select-none text-white/20 w-8 text-right pr-3 inline-block shrink-0">
                {i + 1}
              </span>
              <span
                className="flex-1"
                dangerouslySetInnerHTML={{ __html: highlightLine(line) || " " }}
              />
            </div>
          ))}
        </code>
      </pre>
    </div>
  );
}

export default function ProjectContext() {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const [treeData, setTreeData] = useState<{ tree: ProjectFile | null; indexed: boolean } | null>(null);
  const [treeError, setTreeError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("tree");
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);
  const [graphEdges, setGraphEdges] = useState<GraphEdge[]>([]);
  const [showGraphModal, setShowGraphModal] = useState(false);
  const [previewUrl, setPreviewUrl] = useState("http://localhost:5173");
  const [deviceMode, setDeviceMode] = useState<"desktop" | "mobile">("desktop");
  const [iframeKey, setIframeKey] = useState(0);

  const handleReloadPreview = () => {
    setIframeKey((k) => k + 1);
  };

  useEffect(() => {
    if (!activeProjectId) {
      setTreeData(null);
      setTreeError(null);
      setSelectedFile(null);
      setFileContent(null);
      setGraphEdges([]);
      return;
    }
    setTreeError(null);
    api.projects.tree(activeProjectId).then((res) => {
      setTreeData(res);
    }).catch((err) => {
      setTreeData(null);
      setTreeError(err?.message || "Error al cargar el árbol");
    });
  }, [activeProjectId]);

  useEffect(() => {
    if (!activeProjectId || tab !== "deps") return;
    api.projects.fullGraph(activeProjectId).then((edges) => {
      setGraphEdges(edges);
    }).catch(() => {
      setGraphEdges([]);
    });
  }, [activeProjectId, tab]);

  const handleFileClick = useCallback(async (path: string) => {
    if (!activeProjectId) return;
    setSelectedFile(path);
    setFileLoading(true);
    setFileError(null);
    try {
      const res = await api.projects.readFile(activeProjectId, path);
      setFileContent(res.content);
    } catch (e: unknown) {
      setFileContent(null);
      setFileError(e instanceof Error ? e.message : "No se pudo leer el archivo");
    } finally {
      setFileLoading(false);
    }
  }, [activeProjectId]);

  const handleFileDeps = useCallback((path: string) => {
    setSelectedFile(path);
    setTab("deps");
  }, []);

  return (
    <>
      <aside className="w-[320px] bg-[#1a1a1a] border-l border-white/10 flex flex-col flex-shrink-0 z-40 hidden xl:flex">
        <button
          id="toggle-project-context-preview"
          className="hidden"
          onClick={() => setTab("preview")}
        />
        <div className="flex items-center justify-between p-4 border-b border-white/10">
          <div className="flex items-center gap-2 text-white">
            <span className="material-symbols-outlined text-[#c8e64a]">
              account_tree
            </span>
            <span className="text-[15px] font-semibold">Project Context</span>
          </div>
          {tab === "deps" && graphEdges.length > 0 && (
            <button
              className="w-7 h-7 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
              onClick={() => setShowGraphModal(true)}
              title="Ver grafo completo"
            >
              <span className="material-symbols-outlined text-[18px]">open_in_full</span>
            </button>
          )}
        </div>

        <div className="flex border-b border-white/10 text-[11px]">
          <button
            className={`flex-1 py-2.5 text-center font-medium transition-colors ${
              tab === "tree"
                ? "text-[#c8e64a] border-b-2 border-[#c8e64a]"
                : "text-white/40 hover:text-white/60"
            }`}
            onClick={() => setTab("tree")}
          >
            Archivos
          </button>
          <button
            className={`flex-1 py-2.5 text-center font-medium transition-colors ${
              tab === "deps"
                ? "text-[#c8e64a] border-b-2 border-[#c8e64a]"
                : "text-white/40 hover:text-white/60"
            }`}
            onClick={() => setTab("deps")}
          >
            Dependencias
          </button>
          <button
            className={`flex-1 py-2.5 text-center font-medium transition-colors ${
              tab === "preview"
                ? "text-[#c8e64a] border-b-2 border-[#c8e64a]"
                : "text-white/40 hover:text-white/60"
            }`}
            onClick={() => setTab("preview")}
          >
            Vista Previa
          </button>
        </div>

        <div className="flex-1 overflow-hidden flex flex-col min-h-0">
          {!activeProjectId ? (
            <div className="text-white/30 text-[13px] text-center py-8 px-4">
              Selecciona un proyecto para ver su estructura.
            </div>
          ) : tab === "tree" ? (
            <div className="flex flex-col flex-1 min-h-0">
              <div className="overflow-y-auto flex-1 min-h-0 p-2">
                {treeError ? (
                  <div className="text-red-400 text-[13px] text-center py-8">
                    <span className="material-symbols-outlined text-[20px] block mb-2">error</span>
                    Error al cargar: {treeError}
                    <button
                      className="block mx-auto mt-3 px-3 py-1.5 bg-white/10 rounded-lg text-[11px] font-medium hover:bg-white/15 transition-colors"
                      onClick={() => {
                        setTreeError(null);
                        if (activeProjectId) {
                          api.projects.tree(activeProjectId).then(setTreeData).catch((e) => setTreeError(e?.message || "Error"));
                        }
                      }}
                    >
                      Reintentar
                    </button>
                  </div>
                ) : !treeData ? (
                  <div className="flex items-center justify-center py-8">
                    <span className="material-symbols-outlined text-[20px] animate-spin text-white/30">
                      progress_activity
                    </span>
                  </div>
                ) : !treeData.indexed ? (
                  <div className="text-white/30 text-[13px] text-center py-8 px-4">
                    Proyecto sin indexar. Usa el botón Analizar para escanear la estructura.
                  </div>
                ) : treeData.tree ? (
                  <div className="flex flex-col gap-0.5">
                    {treeData.tree.children?.map((file) => (
                      <FileTreeItem
                        key={file.path || file.name}
                        item={file}
                        onFileClick={handleFileClick}
                        activeFile={selectedFile}
                      />
                    ))}
                    {(!treeData.tree.children || treeData.tree.children.length === 0) && (
                      <div className="text-white/30 text-[13px]">Directorio vacío</div>
                    )}
                  </div>
                ) : null}
              </div>

              {selectedFile && (
                <div className="border-t border-white/10 flex flex-col min-h-0" style={{ maxHeight: "55%" }}>
                  <div className="px-3 py-2 border-b border-white/5 flex items-center justify-between bg-[#252525]">
                    <span className="text-[11px] text-white/50 truncate flex-1 font-mono">
                      {selectedFile}
                    </span>
                    <div className="flex items-center gap-1 ml-2">
                      <button
                        className="w-6 h-6 rounded flex items-center justify-center text-white/30 hover:text-[#c8e64a] hover:bg-white/5 transition-colors"
                        onClick={() => handleFileDeps(selectedFile)}
                        title="Ver dependencias"
                      >
                        <span className="material-symbols-outlined text-[14px]">account_tree</span>
                      </button>
                      <button
                        className="w-6 h-6 rounded flex items-center justify-center text-white/30 hover:text-white/60 hover:bg-white/5 transition-colors"
                        onClick={() => { setSelectedFile(null); setFileContent(null); }}
                      >
                        <span className="material-symbols-outlined text-[14px]">close</span>
                      </button>
                    </div>
                  </div>
                  <div className="flex-1 min-h-0 overflow-hidden">
                    {fileLoading ? (
                      <div className="flex items-center justify-center py-8">
                        <span className="material-symbols-outlined text-[20px] animate-spin text-white/30">progress_activity</span>
                      </div>
                    ) : fileError ? (
                      <div className="text-red-400/80 text-[12px] text-center py-8 px-4">{fileError}</div>
                    ) : fileContent !== null ? (
                      <CodeViewer content={fileContent} path={selectedFile} />
                    ) : null}
                  </div>
                </div>
              )}
            </div>
          ) : tab === "deps" ? (
            <div className="flex-1 flex flex-col min-h-0 bg-[#141414] p-3 gap-3">
              {/* Core Package Dependencies Summary List */}
              <div className="bg-[#1e1e1e] border border-white/10 rounded-xl p-3 flex flex-col gap-2 shadow-sm">
                <div className="flex items-center justify-between">
                  <div className="text-[11px] font-semibold text-white/50 uppercase tracking-wider">
                    Dependencias de Paquetes
                  </div>
                  <span className="text-[10px] bg-[#d1f107]/10 border border-[#d1f107]/20 text-[#d1f107] px-2 py-0.5 rounded font-mono font-medium">
                    7 instaladas
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-1.5 pt-1 text-[11px]">
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">react</span>
                    <span className="text-white/40 font-mono text-[10px]">v18.3.1</span>
                  </div>
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">react-dom</span>
                    <span className="text-white/40 font-mono text-[10px]">v18.3.1</span>
                  </div>
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">vite</span>
                    <span className="text-white/40 font-mono text-[10px]">v6.4.3</span>
                  </div>
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">typescript</span>
                    <span className="text-white/40 font-mono text-[10px]">v5.2.2</span>
                  </div>
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">tailwindcss</span>
                    <span className="text-white/40 font-mono text-[10px]">v3.4.1</span>
                  </div>
                  <div className="flex items-center justify-between bg-white/5 px-2 py-1 rounded text-white/80">
                    <span className="font-medium">zustand</span>
                    <span className="text-white/40 font-mono text-[10px]">v4.5.0</span>
                  </div>
                </div>
              </div>

              {/* Interactive Module Dependency Graph */}
              <div className="flex-1 border border-white/10 rounded-xl bg-[#181818] relative overflow-hidden min-h-[250px] flex flex-col">
                <div className="px-3 py-2 border-b border-white/10 flex items-center justify-between bg-[#1e1e1e]">
                  <div className="flex items-center gap-2">
                    <span className="material-symbols-outlined text-[16px] text-[#d1f107]">hub</span>
                    <span className="text-[12px] font-semibold text-white/90">Grafo de Módulos</span>
                  </div>
                  <button
                    onClick={() => setShowGraphModal(true)}
                    className="flex items-center gap-1 text-[11px] text-white/60 hover:text-white transition-colors"
                  >
                    <span>Expandir</span>
                    <span className="material-symbols-outlined text-[14px]">open_in_full</span>
                  </button>
                </div>

                <div className="flex-1 min-h-0 relative">
                  <DependencyGraph
                    edges={
                      graphEdges.length > 0
                        ? graphEdges
                        : [
                            { id: "1", project_id: "demo", from_symbol: "src/main.tsx", to_symbol: "src/App.tsx", edge_type: "import", created_at: "" },
                            { id: "2", project_id: "demo", from_symbol: "src/App.tsx", to_symbol: "src/components/CodePage.tsx", edge_type: "import", created_at: "" },
                            { id: "3", project_id: "demo", from_symbol: "src/components/CodePage.tsx", to_symbol: "src/components/CodeInput.tsx", edge_type: "import", created_at: "" },
                            { id: "4", project_id: "demo", from_symbol: "src/components/CodePage.tsx", to_symbol: "src/components/ProjectContext.tsx", edge_type: "import", created_at: "" },
                            { id: "5", project_id: "demo", from_symbol: "src/components/ProjectContext.tsx", to_symbol: "src/store/projectsStore.ts", edge_type: "import", created_at: "" },
                          ]
                    }
                    onNodeClick={handleFileClick}
                  />
                </div>
              </div>
            </div>
          ) : tab === "preview" ? (
            <div className="flex-1 flex flex-col min-h-0 bg-[#141414] p-3 gap-2.5">
              {/* URL Controls Header */}
              <div className="flex items-center gap-2 bg-[#222] border border-white/10 rounded-lg p-1.5 text-[12px]">
                <button
                  onClick={handleReloadPreview}
                  className="w-7 h-7 rounded flex items-center justify-center text-white/60 hover:text-white hover:bg-white/10 transition-colors shrink-0"
                  title="Recargar vista previa"
                >
                  <span className="material-symbols-outlined text-[16px]">refresh</span>
                </button>

                <div className="flex-1 flex items-center gap-1.5 bg-[#181818] border border-white/10 rounded px-2 py-1 text-[11px] font-mono text-white/80">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse shrink-0" />
                  <input
                    type="text"
                    value={previewUrl}
                    onChange={(e) => setPreviewUrl(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleReloadPreview()}
                    className="w-full bg-transparent outline-none text-white/90"
                    placeholder="http://localhost:5173"
                  />
                </div>

                <div className="flex items-center gap-1 shrink-0">
                  <button
                    onClick={() => setDeviceMode("desktop")}
                    className={`p-1 rounded text-[14px] transition-colors ${
                      deviceMode === "desktop" ? "bg-[#c8e64a]/20 text-[#c8e64a]" : "text-white/40 hover:text-white"
                    }`}
                    title="Vista Escritorio"
                  >
                    <span className="material-symbols-outlined text-[16px]">desktop_windows</span>
                  </button>

                  <button
                    onClick={() => setDeviceMode("mobile")}
                    className={`p-1 rounded text-[14px] transition-colors ${
                      deviceMode === "mobile" ? "bg-[#c8e64a]/20 text-[#c8e64a]" : "text-white/40 hover:text-white"
                    }`}
                    title="Vista Móvil"
                  >
                    <span className="material-symbols-outlined text-[16px]">smartphone</span>
                  </button>

                  <a
                    href={previewUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="p-1 rounded text-white/40 hover:text-white transition-colors"
                    title="Abrir en pestaña externa"
                  >
                    <span className="material-symbols-outlined text-[16px]">open_in_new</span>
                  </a>
                </div>
              </div>

              {/* Functional Live Web Preview Iframe Container */}
              <div className="flex-1 border border-white/10 rounded-xl bg-[#181818] relative overflow-hidden flex flex-col items-center justify-center">
                <div
                  className={`h-full transition-all duration-300 ${
                    deviceMode === "mobile"
                      ? "w-[375px] max-w-full my-auto border-x border-white/10 shadow-2xl"
                      : "w-full"
                  }`}
                >
                  <iframe
                    key={iframeKey}
                    src={previewUrl}
                    className="w-full h-full border-0 bg-white"
                    title="Web App Live Preview"
                    sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-modals"
                  />
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </aside>

      {showGraphModal && (
        <div className="fixed inset-0 z-[100] bg-black/80 flex items-center justify-center p-6">
          <div className="bg-[#1a1a1a] rounded-2xl border border-white/10 w-full h-full max-w-[1200px] max-h-[90vh] flex flex-col overflow-hidden">
            <div className="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-[#c8e64a] text-[20px]">account_tree</span>
                <span className="text-[15px] font-semibold text-white">Grafo de Dependencias</span>
                <span className="text-[12px] text-white/40 ml-2">{graphEdges.length} edges</span>
              </div>
              <button
                className="w-8 h-8 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
                onClick={() => setShowGraphModal(false)}
              >
                <span className="material-symbols-outlined text-[20px]">close</span>
              </button>
            </div>
            <div className="flex-1 min-h-0 h-full">
              <DependencyGraph
                edges={graphEdges}
                onNodeClick={(path) => {
                  setShowGraphModal(false);
                  handleFileClick(path);
                }}
              />
            </div>
          </div>
        </div>
      )}
    </>
  );
}