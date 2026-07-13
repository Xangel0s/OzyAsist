import { useState, useRef, useEffect } from "react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useAuthStore } from "../../store/authStore";

interface MenuItem {
  label: string;
  shortcut?: string;
  action: () => void;
  divider?: boolean;
}

interface MenuGroup {
  label: string;
  icon: string;
  items: MenuItem[];
}

export default function MenuBar() {
  const [open, setOpen] = useState(false);
  const [subMenu, setSubMenu] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const createChat = useChatStore((s) => s.createChat);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    const clickHandler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setSubMenu(null);
      }
    };
    const keyHandler = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key === "n") { e.preventDefault(); createChat("chat"); }
      if (mod && e.key === "f") { e.preventDefault(); setSearchOpen(true); }
      if (mod && e.key === "r") { e.preventDefault(); window.location.reload(); }
      if (mod && e.key === "w") { e.preventDefault(); window.close(); }
    };
    document.addEventListener("mousedown", clickHandler);
    window.addEventListener("keydown", keyHandler);
    return () => {
      document.removeEventListener("mousedown", clickHandler);
      window.removeEventListener("keydown", keyHandler);
    };
  }, [createChat, setSearchOpen]);

  const close = () => { setOpen(false); setSubMenu(null); };

  const menus: MenuGroup[] = [
    {
      label: "Archivo", icon: "folder",
      items: [
        { label: "Nueva conversación", shortcut: "Ctrl+N", action: () => { createChat("chat"); close(); } },
        { label: "Configuración...", shortcut: "Ctrl+,", action: () => { close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Cerrar ventana", shortcut: "Ctrl+W", action: () => { window.close(); close(); } },
        { label: "Salir", action: () => { window.close(); close(); } },
      ],
    },
    {
      label: "Editar", icon: "edit",
      items: [
        { label: "Deshacer", shortcut: "Ctrl+Z", action: () => { document.execCommand("undo"); close(); } },
        { label: "Rehacer", shortcut: "Ctrl+Shift+Z", action: () => { document.execCommand("redo"); close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Cortar", shortcut: "Ctrl+X", action: () => { document.execCommand("cut"); close(); } },
        { label: "Copiar", shortcut: "Ctrl+C", action: () => { document.execCommand("copy"); close(); } },
        { label: "Pegar", shortcut: "Ctrl+V", action: () => { document.execCommand("paste"); close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Seleccionar todo", shortcut: "Ctrl+A", action: () => { document.execCommand("selectAll"); close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Buscar", shortcut: "Ctrl+F", action: () => { setSearchOpen(true); close(); } },
        { label: "Buscar siguiente", shortcut: "Ctrl+G", action: () => { setSearchOpen(true); close(); } },
        { label: "Buscar anterior", shortcut: "Ctrl+Shift+G", action: () => { setSearchOpen(true); close(); } },
      ],
    },
    {
      label: "Ver", icon: "visibility",
      items: [
        { label: "Recargar", shortcut: "Ctrl+R", action: () => { window.location.reload(); close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Tamaño real", shortcut: "Ctrl+0", action: () => { close(); } },
        { label: "Acercar", shortcut: "Ctrl++", action: () => { close(); } },
        { label: "Alejar", shortcut: "Ctrl+-", action: () => { close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Copiar URL", action: () => { close(); } },
      ],
    },
    {
      label: "Ayuda", icon: "help",
      items: [
        { label: "Abrir documentación", action: () => { window.open("https://opencode.ai", "_blank"); close(); } },
        { label: "Buscar actualizaciones...", action: () => { close(); } },
        { label: "Solución de problemas", action: () => { close(); } },
        { label: "Obtener ayuda", action: () => { close(); } },
        { label: "", action: () => {}, divider: true },
        { label: "Acerca de...", action: () => { close(); } },
      ],
    },
  ];

  return (
    <div ref={ref} className="relative">
      <button
        className={`hover:bg-surface-variant transition-colors p-1.5 rounded-lg flex items-center justify-center ${
          open ? "bg-surface-variant text-on-surface" : "text-text-muted"
        }`}
        onClick={() => { setOpen(!open); setSubMenu(null); }}
        aria-label="Menú"
      >
        <span className="material-symbols-outlined text-[20px]">menu</span>
      </button>

      {open && (
        <div
          className="absolute top-full left-0 mt-1 w-56 bg-surface-elevated border border-border-subtle rounded-xl shadow-xl shadow-black/30 py-1 z-50"
          onMouseLeave={() => setSubMenu(null)}
        >
          {menus.map((menu) => (
            <div
              key={menu.label}
              className="relative"
              onMouseEnter={() => setSubMenu(menu.label)}
            >
              <button
                className={`w-full flex items-center gap-3 px-3 py-1.5 text-[13px] text-on-surface hover:bg-surface-variant transition-colors text-left ${
                  subMenu === menu.label ? "bg-surface-variant" : ""
                }`}
              >
                <span className="material-symbols-outlined text-[16px] text-text-muted">
                  {menu.icon}
                </span>
                <span className="flex-1">{menu.label}</span>
                <span className="material-symbols-outlined text-[14px] text-text-muted">
                  chevron_right
                </span>
              </button>

              {subMenu === menu.label && (
                <div className="absolute left-full top-0 ml-1 w-56 bg-surface-elevated border border-border-subtle rounded-xl shadow-xl shadow-black/30 py-1 z-50">
                  {menu.items.map((item, i) =>
                    item.divider ? (
                      <div key={i} className="h-px bg-border-subtle my-1" />
                    ) : (
                      <button
                        key={i}
                        className="w-full flex items-center justify-between px-3 py-1.5 text-[13px] text-on-surface hover:bg-surface-variant transition-colors text-left"
                        onClick={() => { item.action(); setSubMenu(null); }}
                      >
                        <span>{item.label}</span>
                        {item.shortcut && (
                          <span className="text-text-muted text-[11px] ml-4">{item.shortcut}</span>
                        )}
                      </button>
                    ),
                  )}
                </div>
              )}
            </div>
          ))}

          <div className="h-px bg-border-subtle my-1" />

          <button
            className="w-full flex items-center gap-3 px-3 py-1.5 text-[13px] text-on-surface hover:bg-surface-variant transition-colors text-left"
            onClick={() => { setActiveView("onboarding"); close(); }}
          >
            <span className="material-symbols-outlined text-[16px] text-text-muted">face</span>
            <span>{user?.name || "Usuario"}</span>
            <span className="ml-auto text-text-muted text-[11px]">{user?.plan === "free" ? "Free" : "Pro"}</span>
          </button>
          <button
            className="w-full flex items-center gap-3 px-3 py-1.5 text-[13px] text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
            onClick={() => { logout(); close(); }}
          >
            <span className="material-symbols-outlined text-[16px]">logout</span>
            <span>Cerrar sesión</span>
          </button>
        </div>
      )}
    </div>
  );
}
