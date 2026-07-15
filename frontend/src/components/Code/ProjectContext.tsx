import { useState, useEffect } from "react";
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
}: {
  item: ProjectFile;
  depth?: number;
  onFileClick?: (path: string) => void;
}) {
  const isFolder = item.type === "folder";
  return (
    <div>
      <div
        className={`flex items-center gap-2 hover:bg-white/5 py-1 px-2 rounded cursor-pointer text-[13px] ${
          depth === 0 ? "" : "ml-2"
        }`}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
        onClick={() => !isFolder && onFileClick?.(item.path ?? item.name)}
      >
        <span className={`material-symbols-outlined text-[16px] ${isFolder ? "text-white/40" : "text-[#c8e64a]/70"}`}>
          {isFolder ? "folder" : "description"}
        </span>
        <span className="text-white/70">{item.name}</span>
      </div>
      {item.children?.map((child) => (
        <FileTreeItem key={child.path || child.name} item={child} depth={depth + 1} onFileClick={onFileClick} />
      ))}
    </div>
  );
}

export default function ProjectContext() {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const [treeData, setTreeData] = useState<{ tree: ProjectFile | null; indexed: boolean } | null>(null);
  const [treeError, setTreeError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("tree");
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [neighbors, setNeighbors] = useState<Neighbor[] | null>(null);
  const [graphEdges, setGraphEdges] = useState<GraphEdge[]>([]);
  const [loadingGraph, setLoadingGraph] = useState(false);

  useEffect(() => {
    if (!activeProjectId) {
      setTreeData(null);
      setTreeError(null);
      setNeighbors(null);
      setSelectedFile(null);
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

  const handleFileClick = async (path: string) => {
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
  };

  return (
    <aside className="w-[320px] bg-[#1a1a1a] border-l border-white/10 flex flex-col flex-shrink-0 z-40 hidden xl:flex">
      <div className="flex items-center justify-between p-4 border-b border-white/10">
        <div className="flex items-center gap-2 text-white">
          <span className="material-symbols-outlined text-[#c8e64a]">
            account_tree
          </span>
          <span className="text-[15px] font-semibold">Project Context</span>
        </div>
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

      <div className="flex-1 overflow-hidden">
        {!activeProjectId ? (
          <div className="text-white/30 text-[13px] text-center py-8 px-4">
            Selecciona un proyecto para ver su estructura.
          </div>
        ) : tab === "tree" ? (
          <div className="p-4 overflow-y-auto h-full">
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
              <>
                <div className="text-[11px] font-medium text-white/40 mb-3 uppercase tracking-wider">
                  File Structure
                </div>
                <div className="flex flex-col gap-0.5">
                  {treeData.tree.children?.map((file) => (
                    <FileTreeItem key={file.path || file.name} item={file} onFileClick={handleFileClick} />
                  ))}
                  {(!treeData.tree.children || treeData.tree.children.length === 0) && (
                    <div className="text-white/30 text-[13px]">Directorio vacío</div>
                  )}
                </div>
              </>
            ) : null}
          </div>
        ) : (
          <div className="h-full flex flex-col">
            {selectedFile && (
              <div className="px-4 py-2 border-b border-white/5 flex items-center justify-between">
                <span className="text-[11px] text-white/40 truncate flex-1">
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
  );
}
