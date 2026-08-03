import { useState } from "react";
import { useToastStore } from "../../../store/toastStore";

interface MarketplaceModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function MarketplaceModal({ isOpen, onClose }: MarketplaceModalProps) {
  const toast = useToastStore((s) => s.show);
  const [activeTab, setActiveTab] = useState<"habilidades" | "conectores" | "plugins">("conectores");
  const [searchQuery, setSearchQuery] = useState("");

  if (!isOpen) return null;

  const connectorDirectoryItems = [
    { name: "PostgreSQL MCP", desc: "Conecta a bases de datos relacionales PostgreSQL", author: "Community", installed: true },
    { name: "SQLite Explorer", desc: "Servidor MCP para inspeccionar y consultar SQLite", author: "Ozy Inc", installed: true },
    { name: "GitHub MCP", desc: "Integración completa con issues y pull requests", author: "ModelContextProtocol", installed: false },
    { name: "Sentry Error Tracker", desc: "Inspecciona excepciones y logs en tiempo real", author: "Community", installed: false },
  ];

  return (
    <div
      className="fixed inset-0 bg-black/75 backdrop-blur-md z-[10000] flex items-center justify-center p-6"
      onClick={onClose}
    >
      <div
        className="w-full max-w-4xl h-[80vh] bg-[#222222] border border-white/10 rounded-3xl flex overflow-hidden shadow-2xl text-white animate-fadeIn relative"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Left Sidebar */}
        <div className="w-56 bg-[#1a1a1a] border-r border-white/10 p-5 flex flex-col gap-6 shrink-0 select-none">
          <h2 className="text-[20px] font-sans font-bold text-white">Directorio</h2>

          <div className="flex flex-col gap-1 text-[13px]">
            <button
              className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-left font-medium transition-colors ${
                activeTab === "habilidades" ? "bg-white/10 text-white" : "text-white/50 hover:text-white hover:bg-white/5"
              }`}
              onClick={() => setActiveTab("habilidades")}
            >
              <span className="material-symbols-outlined text-[18px]">handyman</span>
              <span>Habilidades</span>
            </button>
            <button
              className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-left font-medium transition-colors ${
                activeTab === "conectores" ? "bg-white/10 text-white" : "text-white/50 hover:text-white hover:bg-white/5"
              }`}
              onClick={() => setActiveTab("conectores")}
            >
              <span className="material-symbols-outlined text-[18px]">power</span>
              <span>Conectores</span>
            </button>
            <button
              className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-left font-medium transition-colors ${
                activeTab === "plugins" ? "bg-white/10 text-white" : "text-white/50 hover:text-white hover:bg-white/5"
              }`}
              onClick={() => setActiveTab("plugins")}
            >
              <span className="material-symbols-outlined text-[18px]">extension</span>
              <span>Plugins</span>
            </button>
          </div>
        </div>

        {/* Right Main Panel */}
        <div className="flex-1 flex flex-col p-6 overflow-y-auto gap-5">
          <div className="flex items-center justify-between">
            <input
              type="text"
              placeholder="Buscar en el directorio..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="bg-[#1a1a1a] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none w-72"
            />
            <button
              className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
              onClick={onClose}
            >
              <span className="material-symbols-outlined text-[22px]">close</span>
            </button>
          </div>

          <div className="flex flex-col gap-3">
            {connectorDirectoryItems
              .filter((item) => item.name.toLowerCase().includes(searchQuery.toLowerCase()))
              .map((item) => (
                <div key={item.name} className="flex items-center justify-between p-4 bg-[#1a1a1a] border border-white/10 rounded-2xl">
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-[14px] text-white">{item.name}</span>
                      <span className="text-[11px] text-white/40 bg-white/5 px-2 py-0.5 rounded">{item.author}</span>
                    </div>
                    <div className="text-[12px] text-white/60">{item.desc}</div>
                  </div>
                  <button
                    className={`px-4 py-1.5 rounded-xl text-[12px] font-medium transition-colors ${
                      item.installed ? "bg-white/10 text-white/60 cursor-default" : "bg-[#d1f107] text-[#181e00] font-bold hover:opacity-90"
                    }`}
                    onClick={() => {
                      if (!item.installed) {
                        toast(`Instalando ${item.name}...`, "info");
                      }
                    }}
                  >
                    {item.installed ? "Instalado" : "Instalar"}
                  </button>
                </div>
              ))}
          </div>
        </div>
      </div>
    </div>
  );
}
