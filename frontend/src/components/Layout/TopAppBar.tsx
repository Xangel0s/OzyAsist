import { useUIStore } from "../../store/uiStore";
import MenuBar from "./MenuBar";

export default function TopAppBar() {
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);

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
    <header className="flex justify-between items-center w-full px-4 h-12 bg-background border-b border-border-subtle flex-shrink-0 z-50">
      <div className="flex items-center gap-2 text-text-muted">
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
          aria-label="Buscar"
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
      <div className="flex items-center gap-2">
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          onClick={() => setActiveView("onboarding")}
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
