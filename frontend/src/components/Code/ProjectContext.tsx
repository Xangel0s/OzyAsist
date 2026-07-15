import { useState, useEffect, useCallback } from "react";
import { useProjectsStore, type ProjectFile } from "../../store/projectsStore";
import { api, type GraphEdge } from "../../services/api";
import DependencyGraph from "./DependencyGraph";

type Tab = "tree" | "deps";

interface Neighbor {
  file: string;
  relation: string;
}

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
  const [loadingGraph, setLoadingGraph] = useState(false);
  const [showGraphModal, setShowGraphModal] = useState(false);
  const [neighbors, setNeighbors] = useState<Neighbor[] | null>(null);

  useEffect(() => {
    if (!activeProjectId) {
      setTreeData(null);
      setTreeError(null);
      setSelectedFile(null);
      setFileContent(null);
      setNeighbors(null);
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
    setLoadingGraph(true);
    api.projects.fullGraph(activeProjectId).then((edges) => {
      setGraphEdges(edges);
      setLoadingGraph(false);
    }).catch(() => {
      setGraphEdges([]);
      setLoadingGraph(false);
    });
  }, [activeProjectId, tab]);

  const handleFileClick = useCallback(async (path: string) => {
    if (!activeProjectId) return;
    setSelectedFile(path);
    setFileLoading(true);
    setFileError(null);
    setNeighbors(null);
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

  const handleFileDeps = useCallback(async (path: string) => {
    if (!activeProjectId) return;
    setSelectedFile(path);
    setTab("deps");
    setNeighbors(null);
    try {
      const res = await api.projects.graph(activeProjectId, path);
      setNeighbors(res.neighbors);
    } catch {
      setNeighbors([]);
    }
  }, [activeProjectId]);

  return (
    <>
      <aside className="w-[320px] bg-[#1a1a1a] border-l border-white/10 flex flex-col flex-shrink-0 z-40 hidden xl:flex">
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

        <div className="flex border-b border-white/10">
          <button
            className={`flex-1 py-2.5 text-center text-[11px] font-medium uppercase tracking-wider transition-colors ${
              tab === "tree"
                ? "text-[#c8e64a] border-b-2 border-[#c8e64a]"
                : "text-white/40 hover:text-white/60"
            }`}
            onClick={() => setTab("tree")}
          >
            File Tree
          </button>
          <button
            className={`flex-1 py-2.5 text-center text-[11px] font-medium uppercase tracking-wider transition-colors ${
              tab === "deps"
                ? "text-[#c8e64a] border-b-2 border-[#c8e64a]"
                : "text-white/40 hover:text-white/60"
            }`}
            onClick={() => setTab("deps")}
          >
            Dependencies
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
          ) : (
            <div className="flex-1 flex flex-col min-h-0">
              {selectedFile && (
                <div className="px-4 py-2 border-b border-white/5 flex items-center justify-between">
                  <span className="text-[11px] text-white/40 truncate flex-1 font-mono">
                    {selectedFile}
                  </span>
                  <button
                    className="text-white/30 hover:text-white/60 transition-colors ml-2"
                    onClick={() => setSelectedFile(null)}
                  >
                    <span className="material-symbols-outlined text-[16px]">close</span>
                  </button>
                </div>
              )}

              {loadingGraph ? (
                <div className="flex items-center justify-center py-8">
                  <span className="material-symbols-outlined text-[20px] animate-spin text-white/30">
                    progress_activity
                  </span>
                </div>
              ) : graphEdges.length > 0 ? (
                <div className="flex-1 min-h-0">
                  <DependencyGraph
                    edges={graphEdges}
                    onNodeClick={handleFileClick}
                  />
                </div>
              ) : (
                <div className="text-white/30 text-[13px] text-center py-8 px-4">
                  No hay dependencias registradas. Indexa el proyecto primero.
                </div>
              )}

              {selectedFile && neighbors && (
                <div className="border-t border-white/10 p-4 max-h-[200px] overflow-y-auto">
                  <div className="text-[11px] font-medium text-white/40 mb-2 uppercase tracking-wider">
                    {neighbors.length > 0 ? `Dependencias de ${selectedFile.split(/[/\\]/).pop()}` : "Sin dependencias"}
                  </div>
                  {neighbors.length > 0 && (
                    <div className="flex flex-col gap-1">
                      {neighbors.map((n, i) => (
                        <div
                          key={i}
                          className="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-white/5 text-[12px] cursor-pointer"
                          onClick={() => handleFileClick(n.file)}
                        >
                          <span className={`material-symbols-outlined text-[14px] ${
                            n.relation === "imports" ? "text-[#c8e64a]" : "text-amber-400"
                          }`}>
                            {n.relation === "imports" ? "arrow_upward" : "arrow_downward"}
                          </span>
                          <span className="flex-1 truncate text-white/60">{n.file.split(/[/\\]/).pop()}</span>
                          <span className={`text-[10px] font-medium ${
                            n.relation === "imports" ? "text-[#c8e64a]/60" : "text-amber-400/60"
                          }`}>
                            {n.relation === "imports" ? "imports" : "imported by"}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
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
            <div className="flex-1 min-h-0">
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