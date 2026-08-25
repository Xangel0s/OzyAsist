import { useState, useEffect, useCallback, useMemo } from "react";
import { useProjectsStore, type ProjectFile } from "../../store/projectsStore";
import { api, type GraphEdge } from "../../services/api";
import DependencyGraph from "./DependencyGraph";
import GitPanel from "./GitPanel";
import DiffViewer from "./DiffViewer";
import { useToastStore } from "../../store/toastStore";

export type WorkbenchTab = "diffs" | "git" | "tree" | "deps" | "preview";

interface OpenFileTab {
  path: string;
  name: string;
  content: string;
}

function FileTreeNodeItem({
  item,
  depth = 0,
  openFolders,
  toggleFolder,
  onFileClick,
  activeFile,
}: {
  item: ProjectFile;
  depth?: number;
  openFolders: Set<string>;
  toggleFolder: (path: string) => void;
  onFileClick: (path: string) => void;
  activeFile: string | null;
}) {
  const isFolder = item.type === "folder";
  const itemPath = item.path || item.name;
  const isOpen = openFolders.has(itemPath);
  const isActive = !isFolder && itemPath === activeFile;

  const getFileIcon = (filename: string) => {
    const ext = filename.split(".").pop()?.toLowerCase();
    switch (ext) {
      case "go":
        return <span className="text-cyan-400 font-mono text-[11px] font-bold">GO</span>;
      case "ts":
      case "tsx":
        return <span className="text-sky-400 font-mono text-[11px] font-bold">TS</span>;
      case "js":
      case "jsx":
        return <span className="text-amber-400 font-mono text-[11px] font-bold">JS</span>;
      case "json":
        return <span className="text-yellow-500 font-mono text-[11px] font-bold">{"{}"}</span>;
      case "md":
        return <span className="material-symbols-outlined text-[15px] text-zinc-400">description</span>;
      case "css":
        return <span className="text-blue-400 font-mono text-[11px] font-bold">#</span>;
      default:
        return <span className="material-symbols-outlined text-[15px] text-zinc-400">article</span>;
    }
  };

  return (
    <div>
      <div
        className={`flex items-center gap-2 py-1 px-2 rounded-lg cursor-pointer text-[12px] font-mono transition-colors group ${
          isActive
            ? "bg-[#d1f107]/15 text-[#d1f107] font-semibold border border-[#d1f107]/30"
            : "hover:bg-white/5 text-zinc-300"
        }`}
        style={{ paddingLeft: `${8 + depth * 14}px` }}
        onClick={() => {
          if (isFolder) {
            toggleFolder(itemPath);
          } else {
            onFileClick(itemPath);
          }
        }}
      >
        {isFolder ? (
          <span className="material-symbols-outlined text-[16px] text-amber-400 shrink-0">
            {isOpen ? "folder_open" : "folder"}
          </span>
        ) : (
          <div className="w-4 flex items-center justify-center shrink-0">
            {getFileIcon(item.name)}
          </div>
        )}
        <span className="truncate">{item.name}</span>
      </div>

      {isFolder && isOpen && item.children && (
        <div>
          {item.children.map((child) => (
            <FileTreeNodeItem
              key={child.path || child.name}
              item={child}
              depth={depth + 1}
              openFolders={openFolders}
              toggleFolder={toggleFolder}
              onFileClick={onFileClick}
              activeFile={activeFile}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function CodeEditorView({
  tab,
  onClose,
}: {
  tab: OpenFileTab;
  onClose: (path: string) => void;
}) {
  const toast = useToastStore((s) => s.show);
  const lines = tab.content.split("\n");

  const handleCopy = () => {
    navigator.clipboard.writeText(tab.content);
    toast("Código copiado al portapapeles", "info");
  };

  const highlightLine = (line: string): string => {
    let result = line
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");

    result = result.replace(/(#[^\n]*)/g, '<span style="color:#6a9955">$1</span>');
    result = result.replace(/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)/g, '<span style="color:#ce9178">$1</span>');
    result = result.replace(
      /\b(import|from|def|class|return|if|else|elif|for|while|try|except|finally|with|as|yield|async|await|fun|fn|let|mut|const|var|export|default|struct|impl|pub|use|mod|match|loop|break|continue|switch|case|throw|new|this|self|true|false|null|undefined|None|True|False|package|type|func|interface)\b/g,
      '<span style="color:#569cd6">$1</span>'
    );
    result = result.replace(/\b(\d+\.?\d*)\b/g, '<span style="color:#b5cea8">$1</span>');
    result = result.replace(/(\/\/.*$)/gm, '<span style="color:#6a9955">$1</span>');
    return result;
  };

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-[#1e1e1e] border-t border-white/5 font-mono">
      {/* Code Header bar */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-[#242424] border-b border-white/5 text-[11px] text-zinc-400 select-none">
        <div className="flex items-center gap-2 truncate">
          <span className="material-symbols-outlined text-[15px] text-[#d1f107]">description</span>
          <span className="text-white/90 font-medium truncate">{tab.path}</span>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <span className="text-[10px] text-zinc-500 font-mono">{lines.length} líneas</span>
          <button
            onClick={handleCopy}
            className="p-1 rounded hover:bg-white/10 text-zinc-400 hover:text-white transition-colors"
            title="Copiar contenido"
          >
            <span className="material-symbols-outlined text-[15px]">content_copy</span>
          </button>
          <button
            onClick={() => onClose(tab.path)}
            className="p-1 rounded hover:bg-white/10 text-zinc-400 hover:text-white transition-colors"
            title="Cerrar archivo"
          >
            <span className="material-symbols-outlined text-[15px]">close</span>
          </button>
        </div>
      </div>

      {/* Full Code Scroll Area */}
      <div className="flex-1 overflow-auto p-2 text-[11px] leading-[1.6] select-text scrollbar-thin">
        <pre className="m-0">
          <code>
            {lines.map((line, i) => (
              <div key={i} className="flex hover:bg-white/[0.02]">
                <span className="select-none text-zinc-600 w-9 text-right pr-3 shrink-0 text-[10px]">
                  {i + 1}
                </span>
                <span
                  className="flex-1 whitespace-pre"
                  dangerouslySetInnerHTML={{ __html: highlightLine(line) || " " }}
                />
              </div>
            ))}
          </code>
        </pre>
      </div>
    </div>
  );
}

export default function ProjectContext() {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === activeProjectId);

  const [treeData, setTreeData] = useState<{ tree: ProjectFile | null; indexed: boolean } | null>(null);
  const [tab, setTab] = useState<WorkbenchTab>("diffs");
  const [openFolders, setOpenFolders] = useState<Set<string>>(new Set(["src", "backend", "frontend", "internal"]));
  const [fileFilter, setFileFilter] = useState("");
  const [openTabs, setOpenTabs] = useState<OpenFileTab[]>([]);
  const [activeFilePath, setActiveFilePath] = useState<string | null>(null);
  const [graphEdges, setGraphEdges] = useState<GraphEdge[]>([]);

  // Smart Preview state
  const [previewUrl, setPreviewUrl] = useState("http://localhost:1420");
  const [deviceMode, setDeviceMode] = useState<"desktop" | "tablet" | "mobile">("desktop");
  const [iframeKey, setIframeKey] = useState(0);

  // Sync preview URL when active project changes
  useEffect(() => {
    if (!activeProject) return;
    const isSelf =
      activeProject.name.toLowerCase().includes("ozyas") ||
      activeProject.rootPath?.toLowerCase().includes("ozyasis");
    if (isSelf) {
      setPreviewUrl("http://localhost:1420");
    } else {
      setPreviewUrl("http://localhost:5173");
    }
  }, [activeProjectId, activeProject?.name, activeProject?.rootPath]);

  // Load Tree
  const fetchTree = useCallback(() => {
    if (!activeProjectId) {
      setTreeData(null);
      return;
    }
    api.projects
      .tree(activeProjectId)
      .then((res) => {
        setTreeData(res);
      })
      .catch(() => {
        setTreeData(null);
      });
  }, [activeProjectId]);

  useEffect(() => {
    fetchTree();
  }, [fetchTree]);

  // Load Graph
  useEffect(() => {
    if (!activeProjectId || tab !== "deps") return;
    api.projects
      .fullGraph(activeProjectId)
      .then((edges) => setGraphEdges(edges))
      .catch(() => setGraphEdges([]));
  }, [activeProjectId, tab]);

  // Switch tab event listener
  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const handler = (e: any) => {
      if (e.detail?.tab) setTab(e.detail.tab);
      if (e.detail?.path) handleOpenFile(e.detail.path);
    };
    window.addEventListener("switch-project-tab", handler);
    return () => window.removeEventListener("switch-project-tab", handler);
  }, [activeProjectId]);

  const toggleFolder = (folderPath: string) => {
    setOpenFolders((prev) => {
      const next = new Set(prev);
      if (next.has(folderPath)) next.delete(folderPath);
      else next.add(folderPath);
      return next;
    });
  };

  const handleOpenFile = async (path: string) => {
    if (!activeProjectId) return;
    setActiveFilePath(path);
    setTab("tree");

    // Check if already open
    const existing = openTabs.find((t) => t.path === path);
    if (existing) return;

    try {
      const res = await api.projects.readFile(activeProjectId, path);
      const name = path.split("/").pop() || path;
      setOpenTabs((prev) => [...prev, { path, name, content: res.content }]);
    } catch {
      // Fallback
    }
  };

  const handleCloseTab = (path: string) => {
    setOpenTabs((prev) => {
      const next = prev.filter((t) => t.path !== path);
      if (activeFilePath === path) {
        setActiveFilePath(next.length > 0 ? next[next.length - 1].path : null);
      }
      return next;
    });
  };

  const activeOpenTab = useMemo(() => {
    return openTabs.find((t) => t.path === activeFilePath) || openTabs[0] || null;
  }, [openTabs, activeFilePath]);

  return (
    <aside className="w-[420px] 2xl:w-[480px] bg-[#181818] border-l border-white/10 flex shrink-0 z-30 hidden lg:flex font-sans select-none">
      {/* Workbench Main Content Body */}
      <div className="flex-1 flex flex-col min-h-0 bg-[#181818]">
        {/* Top Header Bar */}
        <div className="flex items-center justify-between px-3 py-2 border-b border-white/10 bg-[#202020]">
          <div className="flex items-center gap-2 text-white font-sans">
            <span className="material-symbols-outlined text-[#d1f107] text-[18px]">
              {tab === "git" ? "fork_right" : tab === "diffs" ? "difference" : tab === "tree" ? "folder_open" : tab === "deps" ? "account_tree" : "play_circle"}
            </span>
            <span className="text-[13px] font-semibold tracking-tight capitalize">
              {tab === "tree" ? "Archivos" : tab === "deps" ? "Grafo" : tab}
            </span>
          </div>

          <div className="flex items-center gap-1 text-zinc-400">
            <button
              onClick={() => {
                const nextTab: Record<WorkbenchTab, WorkbenchTab> = {
                  git: "diffs",
                  diffs: "tree",
                  tree: "deps",
                  deps: "preview",
                  preview: "git",
                };
                setTab(nextTab[tab]);
              }}
              className="p-1 rounded hover:text-white hover:bg-white/5 transition-colors"
              title="Cambiar pestaña"
            >
              <span className="material-symbols-outlined text-[15px]">view_carousel</span>
            </button>
          </div>
        </div>

        {/* Tab content area */}
        <div className="flex-1 flex flex-col min-h-0 bg-[#181818]">
        {/* Diffs Tab */}
        {tab === "diffs" && <DiffViewer />}

        {/* Git Tab */}
        {tab === "git" && <GitPanel />}

        {/* File Explorer & Code Tab */}
        {tab === "tree" && (
          <div className="flex-1 flex flex-col min-h-0 bg-[#181818]">
            {/* Open Tabs Header */}
            {openTabs.length > 0 && (
              <div className="flex items-center gap-1 px-2 py-1 bg-[#202020] border-b border-white/5 overflow-x-auto scrollbar-none">
                {openTabs.map((t) => (
                  <div
                    key={t.path}
                    onClick={() => setActiveFilePath(t.path)}
                    className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-mono cursor-pointer transition-all shrink-0 ${
                      t.path === activeFilePath
                        ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 font-semibold"
                        : "bg-white/5 hover:bg-white/10 text-zinc-400 hover:text-white"
                    }`}
                  >
                    <span className="truncate max-w-[110px]">{t.name}</span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCloseTab(t.path);
                      }}
                      className="text-zinc-500 hover:text-rose-400 p-0.5 rounded"
                    >
                      <span className="material-symbols-outlined text-[12px]">close</span>
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* If an active tab is selected, show Code Editor View */}
            {activeOpenTab ? (
              <CodeEditorView tab={activeOpenTab} onClose={handleCloseTab} />
            ) : (
              <div className="flex-1 flex flex-col min-h-0">
                {/* Search File Filter */}
                <div className="p-2 border-b border-white/5 bg-[#202020]">
                  <div className="relative">
                    <span className="material-symbols-outlined absolute left-2 top-2 text-[14px] text-zinc-500">
                      search
                    </span>
                    <input
                      type="text"
                      value={fileFilter}
                      onChange={(e) => setFileFilter(e.target.value)}
                      placeholder="Filtrar archivos..."
                      className="w-full bg-[#2a2a2a] border border-white/10 rounded-lg pl-7 pr-3 py-1 text-white placeholder-zinc-500 text-[11px] font-mono outline-none focus:border-[#d1f107]/50"
                    />
                  </div>
                </div>

                {/* Tree items */}
                <div className="flex-1 overflow-y-auto p-2 scrollbar-thin">
                  {treeData?.tree ? (
                    <FileTreeNodeItem
                      item={treeData.tree}
                      openFolders={openFolders}
                      toggleFolder={toggleFolder}
                      onFileClick={handleOpenFile}
                      activeFile={activeFilePath}
                    />
                  ) : (
                    <div className="flex flex-col items-center justify-center p-8 text-zinc-500 font-mono text-[11px] text-center">
                      <span className="material-symbols-outlined text-[24px] text-zinc-600 mb-2">folder_off</span>
                      <span>No se pudo cargar el árbol de archivos</span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Dependency Graph Tab */}
        {tab === "deps" && (
          <DependencyGraph
            edges={graphEdges}
            onSelectFile={(path) => handleOpenFile(path)}
          />
        )}

        {/* Intelligent Preview Tab */}
        {tab === "preview" && (
          <div className="flex-1 flex flex-col min-h-0 bg-[#181818]">
            {/* Preview Toolbar */}
            <div className="p-2 border-b border-white/10 bg-[#202020] flex items-center justify-between gap-2">
              <div className="flex items-center gap-1 bg-black/40 border border-white/10 rounded-lg px-2 py-1 flex-1 min-w-0">
                <span className="w-2 h-2 rounded-full bg-emerald-400 shrink-0" />
                <input
                  type="text"
                  value={previewUrl}
                  onChange={(e) => setPreviewUrl(e.target.value)}
                  className="bg-transparent text-white text-[11px] font-mono outline-none w-full truncate"
                />
              </div>

              <div className="flex items-center gap-1 shrink-0">
                <button
                  onClick={() => setIframeKey((k) => k + 1)}
                  className="w-7 h-7 rounded-lg bg-white/5 hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center border border-white/5 transition-colors"
                  title="Recargar vista"
                >
                  <span className="material-symbols-outlined text-[15px]">refresh</span>
                </button>

                <button
                  onClick={() => window.open(previewUrl, "_blank")}
                  className="w-7 h-7 rounded-lg bg-white/5 hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center border border-white/5 transition-colors"
                  title="Abrir en pestaña externa"
                >
                  <span className="material-symbols-outlined text-[15px]">open_in_new</span>
                </button>
              </div>
            </div>

            {/* Device Viewport Bar */}
            <div className="px-3 py-1.5 bg-[#1c1c1c] border-b border-white/5 flex items-center justify-center gap-2">
              <button
                onClick={() => setDeviceMode("desktop")}
                className={`p-1 rounded flex items-center gap-1 text-[11px] font-mono transition-colors ${
                  deviceMode === "desktop"
                    ? "bg-[#d1f107]/15 text-[#d1f107]"
                    : "text-zinc-400 hover:text-white"
                }`}
                title="Desktop"
              >
                <span className="material-symbols-outlined text-[15px]">desktop_windows</span>
              </button>

              <button
                onClick={() => setDeviceMode("tablet")}
                className={`p-1 rounded flex items-center gap-1 text-[11px] font-mono transition-colors ${
                  deviceMode === "tablet"
                    ? "bg-[#d1f107]/15 text-[#d1f107]"
                    : "text-zinc-400 hover:text-white"
                }`}
                title="Tablet"
              >
                <span className="material-symbols-outlined text-[15px]">tablet</span>
              </button>

              <button
                onClick={() => setDeviceMode("mobile")}
                className={`p-1 rounded flex items-center gap-1 text-[11px] font-mono transition-colors ${
                  deviceMode === "mobile"
                    ? "bg-[#d1f107]/15 text-[#d1f107]"
                    : "text-zinc-400 hover:text-white"
                }`}
                title="Mobile"
              >
                <span className="material-symbols-outlined text-[15px]">smartphone</span>
              </button>
            </div>

            {/* Iframe Viewport Container */}
            <div className="flex-1 flex items-center justify-center p-2 bg-[#161616] overflow-hidden">
              <div
                style={{
                  width:
                    deviceMode === "mobile"
                      ? "375px"
                      : deviceMode === "tablet"
                      ? "768px"
                      : "100%",
                  height: "100%",
                }}
                className="bg-white rounded-lg shadow-2xl overflow-hidden border border-white/10 transition-all duration-300 relative"
              >
                <iframe
                  key={iframeKey}
                  src={previewUrl}
                  title="App Live Preview"
                  className="w-full h-full border-none bg-white"
                />
              </div>
            </div>
          </div>
        )}
        </div>
      </div>

      {/* Rightmost Vertical Tool Rail (OpenChamber style) */}
      <div className="w-12 border-l border-white/10 bg-[#161616] flex flex-col items-center py-2 gap-2 shrink-0 select-none">
        <button
          onClick={() => setTab("git")}
          className={`w-8 h-8 rounded-xl flex items-center justify-center transition-all ${
            tab === "git"
              ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 shadow-sm"
              : "text-zinc-500 hover:text-white hover:bg-white/5"
          }`}
          title="Control de versiones Git"
        >
          <span className="material-symbols-outlined text-[18px]">fork_right</span>
        </button>

        <button
          onClick={() => setTab("diffs")}
          className={`w-8 h-8 rounded-xl flex items-center justify-center transition-all ${
            tab === "diffs"
              ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 shadow-sm"
              : "text-zinc-500 hover:text-white hover:bg-white/5"
          }`}
          title="Visor de Diffs"
        >
          <span className="material-symbols-outlined text-[18px]">difference</span>
        </button>

        <button
          onClick={() => setTab("tree")}
          className={`w-8 h-8 rounded-xl flex items-center justify-center transition-all ${
            tab === "tree"
              ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 shadow-sm"
              : "text-zinc-500 hover:text-white hover:bg-white/5"
          }`}
          title="Explorador de Archivos"
        >
          <span className="material-symbols-outlined text-[18px]">folder_open</span>
        </button>

        <button
          onClick={() => setTab("deps")}
          className={`w-8 h-8 rounded-xl flex items-center justify-center transition-all ${
            tab === "deps"
              ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 shadow-sm"
              : "text-zinc-500 hover:text-white hover:bg-white/5"
          }`}
          title="Grafo de Dependencias"
        >
          <span className="material-symbols-outlined text-[18px]">account_tree</span>
        </button>

        <button
          onClick={() => setTab("preview")}
          className={`w-8 h-8 rounded-xl flex items-center justify-center transition-all ${
            tab === "preview"
              ? "bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 shadow-sm"
              : "text-zinc-500 hover:text-white hover:bg-white/5"
          }`}
          title="Live Preview"
        >
          <span className="material-symbols-outlined text-[18px]">play_circle</span>
        </button>
      </div>
    </aside>
  );
}