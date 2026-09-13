import React, { useEffect, useRef, useState } from "react";
import mermaid from "mermaid";

interface MermaidDiagramProps {
  code: string;
}

// Initialize mermaid with dark theme and suppress default DOM error injection
mermaid.initialize({
  startOnLoad: false,
  suppressErrorRendering: true,
  theme: "dark",
  securityLevel: "loose",
  themeVariables: {
    darkMode: true,
    background: "#181818",
    primaryColor: "#2a2a2a",
    primaryTextColor: "#f4f4f5",
    primaryBorderColor: "#d1f107",
    lineColor: "#d1f107",
    secondaryColor: "#222222",
    tertiaryColor: "#1e1e1e",
    fontFamily: "ui-sans-serif, system-ui, sans-serif",
    fontSize: "13px",
  },
});

export const MermaidDiagram: React.FC<MermaidDiagramProps> = ({ code }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [svg, setSvg] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    // Limpiar cualquier residuo de errores que mermaid haya insertado en el DOM
    const cleanupErrorElements = () => {
      try {
        const errorNodes = document.querySelectorAll(
          'svg[id^="dmermaid"], div[id^="dmermaid"], svg[id^="mermaid-"], div.error-icon, div.mermaid-error'
        );
        errorNodes.forEach((node) => {
          if (node.parentElement === document.body) {
            node.remove();
          }
        });
      } catch {}
    };

    const renderDiagram = async () => {
      const trimmed = code.trim();
      if (!trimmed) return;

      try {
        // 1. Validar sintaxis primero antes de intentar renderizar
        const parseResult = await mermaid.parse(trimmed, { suppressErrors: true }).catch(() => false);
        if (parseResult === false) {
          // Si aún no es sintácticamente válido (ej: se está generando en streaming), evitar renderizado
          cleanupErrorElements();
          if (isMounted) {
            setError("Sintaxis incompleta o en generación...");
          }
          return;
        }

        // 2. Renderizar SVG con ID único
        const id = `mermaid-${Math.random().toString(36).substring(2, 9)}`;
        const { svg: renderedSvg } = await mermaid.render(id, trimmed);
        cleanupErrorElements();

        if (isMounted) {
          setSvg(renderedSvg);
          setError(null);
        }
      } catch {
        cleanupErrorElements();
        if (isMounted) {
          setError("Diagrama en generación...");
        }
      }
    };

    renderDiagram();

    return () => {
      isMounted = false;
      cleanupErrorElements();
    };
  }, [code]);

  if (error || !svg) {
    return (
      <div className="my-3 rounded-2xl border border-white/10 bg-[#1e1e1e] p-3 text-xs font-mono text-neutral-300">
        <div className="flex items-center justify-between text-neutral-500 mb-2 border-b border-white/5 pb-1">
          <span className="text-[11px] font-semibold text-[#d1f107]">Diagrama Mermaid</span>
          <span className="text-[10px] text-neutral-400">Código fuente</span>
        </div>
        <pre className="overflow-x-auto text-[12px] text-neutral-300 leading-relaxed font-mono">{code}</pre>
      </div>
    );
  }

  return (
    <div className="my-3 overflow-hidden rounded-2xl border border-white/10 bg-[#161616]/90 p-4 shadow-xl backdrop-blur-md transition-all">
      <div className="flex items-center justify-between border-b border-white/5 pb-2 mb-3 text-[11px] font-mono text-neutral-400">
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-[#d1f107]" />
          <span className="font-semibold text-neutral-200">Diagrama de Flujo</span>
        </div>
        <span className="text-[10px] text-neutral-500 font-sans">Mermaid</span>
      </div>
      <div
        ref={containerRef}
        className="mermaid-svg-container flex justify-center items-center overflow-x-auto py-2 text-center"
        dangerouslySetInnerHTML={{ __html: svg }}
      />
    </div>
  );
};

export default MermaidDiagram;
