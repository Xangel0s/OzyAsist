import { useMemo, useCallback, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  MarkerType,
} from "reactflow";
import "reactflow/dist/style.css";
import type { GraphEdge } from "../../services/api";

interface DependencyGraphProps {
  edges: GraphEdge[];
  onNodeClick?: (filepath: string) => void;
}

function buildGraph(edges: GraphEdge[]) {
  const nodeMap = new Map<string, { imports: Set<string>; importedBy: Set<string> }>();

  for (const edge of edges) {
    if (!edge.from_symbol || !edge.to_symbol) continue;
    if (!nodeMap.has(edge.from_symbol)) {
      nodeMap.set(edge.from_symbol, { imports: new Set(), importedBy: new Set() });
    }
    if (!nodeMap.has(edge.to_symbol)) {
      nodeMap.set(edge.to_symbol, { imports: new Set(), importedBy: new Set() });
    }
    nodeMap.get(edge.from_symbol)!.imports.add(edge.to_symbol);
    nodeMap.get(edge.to_symbol)!.importedBy.add(edge.from_symbol);
  }

  const nodes: Node[] = [];
  const edgeList: Edge[] = [];
  const fileList = Array.from(nodeMap.keys()).sort();

  const cols = Math.max(1, Math.ceil(Math.sqrt(fileList.length)));
  const spacingX = 220;
  const spacingY = 100;

  fileList.forEach((file, i) => {
    if (!file) return;
    const col = i % cols;
    const row = Math.floor(i / cols);
    const data = nodeMap.get(file)!;
    const shortName = file.split(/[/\\]/).pop() || file;

    nodes.push({
      id: file,
      position: { x: col * spacingX, y: row * spacingY },
      data: {
        label: shortName,
        fullpath: file,
        imports: data.imports.size,
        importedBy: data.importedBy.size,
        isEntry: data.importedBy.size === 0 && data.imports.size > 0,
      },
      style: {
        background: "#1e1e1e",
        border: "1px solid rgba(255,255,255,0.1)",
        borderRadius: "8px",
        padding: "8px 12px",
        fontSize: "11px",
        color: "#fff",
        width: "fit-content",
        maxWidth: "180px",
      },
    });
  });

  for (const edge of edges) {
    if (!edge.from_symbol || !edge.to_symbol) continue;
    edgeList.push({
      id: `${edge.from_symbol}->${edge.to_symbol}`,
      source: edge.from_symbol,
      target: edge.to_symbol,
      animated: edge.edge_type === "import",
      style: { stroke: "rgba(255,255,255,0.15)", strokeWidth: 1 },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: "rgba(255,255,255,0.2)",
        width: 12,
        height: 12,
      },
    });
  }

  return { nodes, edgeList };
}

export default function DependencyGraph({ edges, onNodeClick }: DependencyGraphProps) {
  const { nodes: initNodes, edgeList: initEdges } = useMemo(() => buildGraph(edges), [edges]);

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edgeState, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    setNodes(initNodes);
    setEdges(initEdges);
  }, [initNodes, initEdges, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (onNodeClick && node.data?.fullpath) {
        onNodeClick(node.data.fullpath);
      }
    },
    [onNodeClick],
  );

  return (
    <div className="w-full h-full bg-[#1a1a1a] rounded-xl overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edgeState}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        defaultEdgeOptions={{
          style: { stroke: "rgba(255,255,255,0.15)", strokeWidth: 1 },
        }}
        proOptions={{ hideAttribution: true }}
      >
        <Background color="rgba(255,255,255,0.03)" gap={20} />
        <Controls
          style={{ background: "#1e1e1e", borderColor: "rgba(255,255,255,0.1)" }}
        />
        <MiniMap
          maskColor="rgba(0,0,0,0.5)"
          style={{ background: "#1a1a1a", border: "1px solid rgba(255,255,255,0.1)" }}
        />
      </ReactFlow>
    </div>
  );
}