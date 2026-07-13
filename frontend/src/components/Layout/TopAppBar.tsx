import { useUIStore } from "../../store/uiStore";
import MenuBar from "./MenuBar";

export default function TopAppBar() {
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);

  return (
    <header className="flex justify-between items-center w-full px-4 h-12 bg-background border-b border-border-subtle flex-shrink-0 z-50">
      <div className="flex items-center gap-2 text-text-muted">
        <MenuBar />
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
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
          aria-label="Atrás"
        >
          <span className="material-symbols-outlined text-[20px]">arrow_back</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center"
          aria-label="Adelante"
        >
          <span className="material-symbols-outlined text-[20px]">arrow_forward</span>
        </button>
      </div>
      <div className="flex items-center gap-2">
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          aria-label="Perfil"
        >
          <span className="material-symbols-outlined text-[20px]">face</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          aria-label="Minimizar"
        >
          <span className="material-symbols-outlined text-[20px]">minimize</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          aria-label="Pantalla completa"
        >
          <span className="material-symbols-outlined text-[20px]">fullscreen</span>
        </button>
        <button
          className="hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center text-text-muted"
          aria-label="Cerrar"
        >
          <span className="material-symbols-outlined text-[20px]">close</span>
        </button>
      </div>
    </header>
  );
}
