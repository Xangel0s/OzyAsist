import { useState, useEffect } from "react";
import { useUIStore } from "../../store/uiStore";
import MenuBar from "./MenuBar";
import { wsClient } from "../../services/ws";
import { useToastStore } from "../../store/toastStore";

function WifiSignalIcon({ connected }: { connected: boolean }) {
  if (connected) {
    return (
      <div className="flex items-end gap-[2px] h-3.5 w-3.5 justify-center" title="Señal del Agente: Excelente">
        <span className="w-[2.5px] h-[30%] bg-emerald-400 rounded-sm" />
        <span className="w-[2.5px] h-[55%] bg-emerald-400 rounded-sm" />
        <span className="w-[2.5px] h-[80%] bg-emerald-400 rounded-sm" />
        <span className="w-[2.5px] h-[100%] bg-emerald-400 rounded-sm" />
      </div>
    );
  }
  return (
    <div className="flex items-end gap-[2px] h-3.5 w-3.5 justify-center opacity-60" title="Sin señal del Agente">
      <span className="w-[2.5px] h-[30%] bg-amber-400 rounded-sm animate-pulse" />
      <span className="w-[2.5px] h-[55%] bg-text-muted/40 rounded-sm" />
      <span className="w-[2.5px] h-[80%] bg-text-muted/40 rounded-sm" />
      <span className="w-[2.5px] h-[100%] bg-text-muted/40 rounded-sm" />
    </div>
  );
}

export default function TopAppBar() {
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);
  const coworkMode = useUIStore((s) => s.coworkMode);
  const toggleCoworkMode = useUIStore((s) => s.toggleCoworkMode);
  const [agentConnected, setAgentConnected] = useState(wsClient.isConnected());

  useEffect(() => {
    const unsubscribe = wsClient.subscribeStatus((connected) => {
      setAgentConnected(connected);
    });
    return () => unsubscribe();
  }, []);

  const handleConnectAgent = () => {
    wsClient.connect();
    if (wsClient.isConnected()) {
      useToastStore.getState().show("Conexión con el agente establecida", "success");
    } else {
      useToastStore.getState().show("Intentando reconectar con el Agente...", "info");
    }
  };

  const goBack = () => window.history.back();
  const goForward = () => window.history.forward();
  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen().catch(() => {});
    } else {
      document.exitFullscreen().catch(() => {});
    }
  };

  return (
    <header 
      className="flex justify-between items-center w-full px-4 h-12 bg-background border-b border-border-subtle flex-shrink-0 z-50"
      data-tauri-drag-region
    >
      <div className="flex items-center gap-2 text-text-muted" data-tauri-drag-region>
        <MenuBar />
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
          onClick={toggleSidebar}
          aria-label="Sidebar"
        >
          <span className="material-symbols-outlined text-[20px]">dock_to_right</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
          onClick={() => setSearchOpen(true)}
          aria-label="Búsqueda Inteligente (Ctrl+K)"
          title="Búsqueda Inteligente (Ctrl+K)"
        >
          <span className="material-symbols-outlined text-[20px]">search</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
          onClick={goBack}
          aria-label="Atrás"
        >
          <span className="material-symbols-outlined text-[20px]">arrow_back</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
          onClick={goForward}
          aria-label="Adelante"
        >
          <span className="material-symbols-outlined text-[20px]">arrow_forward</span>
        </button>
      </div>
      <div className="flex items-center gap-3">
        <button
          className="flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-medium bg-[#d1f107]/15 text-[#d1f107] border border-[#d1f107]/30 hover:bg-[#d1f107]/25 transition-all shadow-sm cursor-pointer"
          onClick={() => useUIStore.getState().setVoiceLiveOpen(true)}
          title="Ozy Live — Voz bidireccional continua (Di 'Hey Ozy' o Alt+V)"
        >
          <span className="material-symbols-outlined text-[16px] animate-pulse">mic</span>
          <span className="font-semibold">Ozy Live</span>
        </button>

        <button
          className={`flex items-center justify-center p-2 rounded-full transition-all ${
            agentConnected
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 hover:bg-emerald-500/20"
              : "bg-amber-500/10 text-amber-400 border border-amber-500/20 hover:bg-amber-500/20"
          }`}
          onClick={handleConnectAgent}
          title={agentConnected ? "Agente En Línea (Señal Excelente)" : "Sin Conexión — Clic para reconectar con el Agente"}
        >
          <WifiSignalIcon connected={agentConnected} />
        </button>

        <button
          className={`flex items-center gap-1.5 px-3 py-1 rounded-full text-[12px] font-medium transition-all ${
            coworkMode
              ? "bg-[#c8e64a]/20 text-[#c8e64a] border border-[#c8e64a]/40 shadow-sm shadow-[#c8e64a]/20"
              : "bg-surface-variant text-text-muted hover:text-on-surface"
          }`}
          onClick={toggleCoworkMode}
          title="Modo Cowork (Colaboración interactiva en tiempo real)"
        >
          <span className="material-symbols-outlined text-[16px]">groups</span>
          <span>Cowork</span>
          {coworkMode && <span className="w-2 h-2 rounded-full bg-[#c8e64a] animate-pulse" />}
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          onClick={() => useUIStore.getState().openSettings("general")}
          aria-label="Perfil"
        >
          <span className="material-symbols-outlined text-[20px]">face</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          onClick={toggleFullscreen}
          aria-label="Pantalla completa"
        >
          <span className="material-symbols-outlined text-[20px]">fullscreen</span>
        </button>
      </div>
    </header>
  );
}
