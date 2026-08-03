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
        background: "#222222",
        border: "1px solid rgba(209,241,7,0.35)",
        borderRadius: "8px",
        padding: "10px 14px",
        fontSize: "12px",
        color: "#ffffff",
        width: 180,
        fontWeight: 500,
        boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
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
      style: { stroke: "rgba(209,241,7,0.5)", strokeWidth: 2 },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: "rgba(209,241,7,0.6)",
        width: 16,
        height: 16,
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
    <div className="w-full h-full bg-[#181818] rounded-xl overflow-hidden relative">
      <ReactFlow
        nodes={nodes}
        edges={edgeState}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        defaultEdgeOptions={{
          style: { stroke: "rgba(209,241,7,0.5)", strokeWidth: 2 },
        }}
        proOptions={{ hideAttribution: true }}
      >
        <Background color="rgba(209,241,7,0.06)" gap={20} />
        <Controls
          style={{ background: "#222222", borderColor: "rgba(255,255,255,0.1)", borderRadius: "8px" }}
        />
        <MiniMap
          nodeColor="#d1f107"
          maskColor="rgba(0,0,0,0.6)"
          style={{ background: "#141414", border: "1px solid rgba(255,255,255,0.1)", borderRadius: "8px" }}
        />
      </ReactFlow>
    </div>
  );
}