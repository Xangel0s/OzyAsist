import { useState, useMemo, useRef } from "react";
import { type GraphEdge } from "../../services/api";

interface DependencyGraphProps {
  edges: GraphEdge[];
  onSelectFile?: (filepath: string) => void;
}

interface Node {
  id: string;
  name: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  degree: number;
}

export default function DependencyGraph({
  edges,
  onSelectFile,
}: DependencyGraphProps) {
  const [search, setSearch] = useState("");
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);

  // Extract unique nodes & calculate degrees
  const { nodes, links } = useMemo(() => {
    const nodeMap = new Map<string, Node>();
    const linkList: { source: string; target: string; type: string }[] = [];

    // Helper to get or create
    const getNode = (id: string): Node => {
      if (!nodeMap.has(id)) {
        const angle = Math.random() * Math.PI * 2;
        const radius = 100 + Math.random() * 160;
        nodeMap.set(id, {
          id,
          name: id.split("/").pop() || id,
          x: 250 + Math.cos(angle) * radius,
          y: 200 + Math.sin(angle) * radius,
          vx: 0,
          vy: 0,
          degree: 0,
        });
      }
      return nodeMap.get(id)!;
    };

    edges.forEach((edge) => {
      const src = getNode(edge.from_symbol);
      const tgt = getNode(edge.to_symbol);
      src.degree += 1;
      tgt.degree += 1;
      linkList.push({
        source: edge.from_symbol,
        target: edge.to_symbol,
        type: edge.edge_type,
      });
    });

    return {
      nodes: Array.from(nodeMap.values()),
      links: linkList,
    };
  }, [edges]);

  // Handle wheel zoom
  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    const factor = e.deltaY < 0 ? 1.1 : 0.9;
    setZoom((z) => Math.min(Math.max(z * factor, 0.4), 3.0));
  };

  // Handle pan
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button === 0) {
      setIsPanning(true);
      setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isPanning) {
      setPan({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      });
    }
  };

  const handleMouseUp = () => setIsPanning(false);

  const filteredNodes = useMemo(() => {
    if (!search.trim()) return nodes;
    const q = search.toLowerCase();
    return nodes.filter((n) => n.id.toLowerCase().includes(q));
  }, [nodes, search]);

  const matchedNodeIds = useMemo(() => new Set(filteredNodes.map((n) => n.id)), [filteredNodes]);

  const nodeMap = useMemo(() => {
    const map = new Map<string, Node>();
    nodes.forEach((n) => map.set(n.id, n));
    return map;
  }, [nodes]);

  if (edges.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-6 text-center text-zinc-500 font-mono text-xs select-none bg-[#181818]">
        <div className="w-12 h-12 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center text-zinc-400 mb-3">
          <span className="material-symbols-outlined text-[24px]">account_tree</span>
        </div>
        <span className="text-white/80 font-semibold mb-1">Sin dependencias indexadas</span>
        <span className="text-zinc-500 text-[11px] max-w-xs leading-relaxed">
          Indexa el proyecto o analiza archivos con imports para visualizar la arquitectura de módulos.
        </span>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-[#181818] text-xs font-sans select-none overflow-hidden relative">
      {/* Search & Controls Header */}
      <div className="p-2.5 border-b border-white/10 bg-[#202020] flex items-center gap-2 z-20">
        <div className="relative flex-1">
          <span className="material-symbols-outlined absolute left-2 top-2 text-[14px] text-zinc-500">
            search
          </span>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar módulo o archivo..."
            className="w-full bg-[#2a2a2a] border border-white/10 rounded-lg pl-7 pr-3 py-1 text-white placeholder-zinc-500 text-[11px] font-mono outline-none focus:border-[#d1f107]/50"
          />
        </div>

        <div className="flex items-center gap-1">
          <button
            onClick={() => setZoom((z) => Math.min(z * 1.2, 3))}
            className="w-7 h-7 rounded-lg bg-white/5 hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center border border-white/5"
            title="Acercar"
          >
            <span className="material-symbols-outlined text-[15px]">add</span>
          </button>
          <button
            onClick={() => setZoom((z) => Math.max(z * 0.8, 0.4))}
            className="w-7 h-7 rounded-lg bg-white/5 hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center border border-white/5"
            title="Alejar"
          >
            <span className="material-symbols-outlined text-[15px]">remove</span>
          </button>
          <button
            onClick={() => {
              setZoom(1);
              setPan({ x: 0, y: 0 });
            }}
            className="w-7 h-7 rounded-lg bg-white/5 hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center border border-white/5"
            title="Centrar vista"
          >
            <span className="material-symbols-outlined text-[15px]">restart_alt</span>
          </button>
        </div>
      </div>

      {/* Stats summary bar */}
      <div className="px-3 py-1.5 bg-[#1c1c1c] border-b border-white/5 flex items-center justify-between text-[10px] font-mono text-zinc-500">
        <span>{nodes.length} módulos detectados</span>
        <span>{links.length} conexiones AST</span>
      </div>

      {/* Interactive SVG Canvas */}
      <div
        ref={containerRef}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        className="flex-1 w-full h-full cursor-grab active:cursor-grabbing bg-[#161616] relative overflow-hidden"
      >
        <svg
          className="w-full h-full"
          style={{
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
            transformOrigin: "center center",
            transition: isPanning ? "none" : "transform 0.05s ease-out",
          }}
        >
          <defs>
            <marker
              id="arrow"
              viewBox="0 0 10 10"
              refX="16"
              refY="5"
              markerWidth="5"
              markerHeight="5"
              orient="auto-start-reverse"
            >
              <path d="M 0 0 L 10 5 L 0 10 z" fill="rgba(255,255,255,0.25)" />
            </marker>
          </defs>

          {/* Links */}
          {links.map((link, idx) => {
            const src = nodeMap.get(link.source);
            const tgt = nodeMap.get(link.target);
            if (!src || !tgt) return null;

            const isHighlighted =
              hoveredNode === link.source || hoveredNode === link.target;

            return (
              <line
                key={`link-${idx}`}
                x1={src.x}
                y1={src.y}
                x2={tgt.x}
                y2={tgt.y}
                stroke={isHighlighted ? "#d1f107" : "rgba(255,255,255,0.12)"}
                strokeWidth={isHighlighted ? 2 : 1}
                strokeDasharray={link.type === "import" ? "none" : "3,3"}
                markerEnd="url(#arrow)"
              />
            );
          })}

          {/* Nodes */}
          {nodes.map((node) => {
            const isMatch = matchedNodeIds.has(node.id);
            const isHovered = hoveredNode === node.id;
            const radius = Math.min(8 + node.degree * 2, 20);

            return (
              <g
                key={node.id}
                transform={`translate(${node.x}, ${node.y})`}
                className="cursor-pointer transition-transform duration-150"
                onMouseEnter={() => setHoveredNode(node.id)}
                onMouseLeave={() => setHoveredNode(null)}
                onClick={(e) => {
                  e.stopPropagation();
                  onSelectFile?.(node.id);
                }}
              >
                {/* Node circle */}
                <circle
                  r={radius}
                  fill={
                    isHovered
                      ? "#d1f107"
                      : isMatch
                      ? "#27272a"
                      : "#18181b"
                  }
                  stroke={
                    isHovered
                      ? "#d1f107"
                      : isMatch
                      ? "#3f3f46"
                      : "rgba(255,255,255,0.1)"
                  }
                  strokeWidth={isHovered ? 2.5 : 1.5}
                  className="transition-all"
                />

                {/* Node icon */}
                <text
                  textAnchor="middle"
                  dy=".3em"
                  fill={isHovered ? "#181e00" : "#a1a1aa"}
                  fontSize={radius > 12 ? 10 : 8}
                  fontFamily="monospace"
                  fontWeight="bold"
                >
                  {node.degree}
                </text>

                {/* Node label */}
                <text
                  x={radius + 6}
                  y={4}
                  fill={isHovered ? "#d1f107" : isMatch ? "#ffffff" : "#71717a"}
                  fontSize={11}
                  fontFamily="monospace"
                  fontWeight={isHovered || isMatch ? "bold" : "normal"}
                  className="select-none pointer-events-none"
                >
                  {node.name}
                </text>
              </g>
            );
          })}
        </svg>

        {/* Floating tooltip when node is hovered */}
        {hoveredNode && (
          <div className="absolute bottom-3 left-3 bg-[#181818]/95 border border-[#d1f107]/40 rounded-xl px-3 py-2 text-white font-mono text-[11px] shadow-2xl backdrop-blur-md z-30 pointer-events-none">
            <div className="text-[#d1f107] font-semibold truncate max-w-xs">{hoveredNode}</div>
            <div className="text-zinc-400 text-[10px] mt-0.5">Haz clic para abrir archivo</div>
          </div>
        )}
      </div>
    </div>
  );
}