import { useState, useEffect } from "react";
import { useProjectsStore, type ProjectFile } from "../../store/projectsStore";
import { api } from "../../services/api";

function FileTreeItem({
  item,
  depth = 0,
}: {
  item: ProjectFile;
  depth?: number;
}) {
  const isFolder = item.type === "folder";
  return (
    <div>
      <div
        className={`flex items-center gap-2 hover:bg-surface-variant py-1 px-2 rounded cursor-pointer text-code-sm ${
          depth === 0 ? "" : "ml-2"
        }`}
        style={{ paddingLeft: `${8 + depth * 16}px` }}
      >
        <span
          className={`material-symbols-outlined text-[16px] ${
            isFolder ? "" : "text-secondary"
          }`}
        >
          {isFolder ? "folder" : "description"}
        </span>
        <span>{item.name}</span>
      </div>
      {item.children?.map((child) => (
        <FileTreeItem key={child.path || child.name} item={child} depth={depth + 1} />
      ))}
    </div>
  );
}

interface TreeResponse {
  tree: ProjectFile | null;
  indexed: boolean;
}

export default function ProjectContext() {
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const [treeData, setTreeData] = useState<TreeResponse | null>(null);

  useEffect(() => {
    if (!activeProjectId) {
      setTreeData(null);
      return;
    }
    api.projects.tree(activeProjectId).then((res) => {
      setTreeData(res);
    }).catch(() => {
      setTreeData(null);
    });
  }, [activeProjectId]);

  return (
    <aside className="w-[320px] bg-surface-container-low border-l border-border-subtle flex flex-col flex-shrink-0 z-40 hidden xl:flex">
      <div className="flex items-center justify-between p-4 border-b border-border-subtle">
        <div className="flex items-center gap-2 text-on-surface">
          <span className="material-symbols-outlined text-primary-container">
            account_tree
          </span>
          <span className="text-lg font-headline-md">Project Context</span>
        </div>
        <button className="text-text-muted hover:text-on-surface p-1 rounded-md hover:bg-surface-variant transition-colors">
          <span className="material-symbols-outlined">close</span>
        </button>
      </div>
      <div className="p-4 flex-1 overflow-y-auto">
        {!activeProjectId ? (
          <div className="text-text-muted text-body-md text-center py-8">
            Selecciona un proyecto para ver su estructura.
          </div>
        ) : !treeData ? (
          <div className="flex items-center justify-center py-8">
            <span className="material-symbols-outlined text-[20px] animate-spin text-text-muted">
              progress_activity
            </span>
          </div>
        ) : !treeData.indexed ? (
          <div className="text-text-muted text-body-md text-center py-8">
            Proyecto sin indexar. Usa el botón Analizar para escanear la estructura.
          </div>
        ) : treeData.tree ? (
          <>
            <div className="text-label-caps text-text-muted mb-3 uppercase tracking-wider">
              File Structure
            </div>
            <div className="flex flex-col gap-1">
              {treeData.tree.children?.map((file) => (
                <FileTreeItem key={file.path || file.name} item={file} />
              ))}
              {(!treeData.tree.children || treeData.tree.children.length === 0) && (
                <div className="text-text-muted text-body-sm">Directorio vacío</div>
              )}
            </div>
          </>
        ) : null}
      </div>
    </aside>
  );
}
