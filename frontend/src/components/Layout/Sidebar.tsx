import { useState, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useAuthStore } from "../../store/authStore";
import { useOnClickOutside } from "../../hooks";
import SecurityPinModal from "../Auth/SecurityPinModal";

const ONBOARDING_TOTAL_STEPS = 3;

export default function Sidebar() {
  const sidebarOpen = useUIStore((s) => s.sidebarOpen);
  const activeView = useUIStore((s) => s.activeView);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const onboardingStep = useUIStore((s) => s.onboardingStep);
  const setOnboardingStep = useUIStore((s) => s.setOnboardingStep);
  const chats = useChatStore((s) => s.chats);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const user = useAuthStore((s) => s.user);

  const [showFilterMenu, setShowFilterMenu] = useState(false);
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const [showPinModal, setShowPinModal] = useState(false);
  const [activeSubmenu, setActiveSubmenu] = useState<"tipo" | null>(null);
  const [typeFilter, setTypeFilter] = useState("Todo");
  const filterMenuRef = useRef<HTMLDivElement>(null);
  const profileMenuRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(filterMenuRef, () => {
    setShowFilterMenu(false);
    setActiveSubmenu(null);
  });

  useOnClickOutside(profileMenuRef, () => {
    setShowProfileMenu(false);
  });

  const visibleChats = chats.filter((c) => {
    if (typeFilter === "Chat") return c.mode === "chat";
    if (typeFilter === "Tarea") return c.mode === "code";
    return true;
  });

  const handleNewChat = async () => {
    setActiveChat("");
    setActiveView("home");
  };

  return (
    <nav
      className={`bg-[#181818] border-r border-white/10 flex flex-col h-full flex-shrink-0 transition-all duration-300 ease-in-out relative z-10 overflow-hidden select-none font-sans ${
        sidebarOpen ? "w-[260px] opacity-100" : "w-0 opacity-0 p-0 border-none pointer-events-none"
      }`}
    >
      <div className="p-3 flex flex-col gap-2">
        {/* + New Session Button (Full Brand Lime Capsule) */}
        <button
          className="w-full flex items-center justify-between px-3 py-2.5 bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold rounded-xl shadow-lg shadow-[#d1f107]/10 transition-all text-[12px] font-sans group active:scale-[0.99]"
          onClick={handleNewChat}
        >
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[18px] text-[#181e00] font-bold">add</span>
            <span className="font-bold text-[#181e00]">New session</span>
          </div>
          <span className="text-[10px] text-[#181e00]/70 bg-black/10 px-1.5 py-0.5 rounded border border-black/10 font-sans font-semibold">
            ⌘N
          </span>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-3 flex flex-col gap-3 min-h-0 scrollbar-thin">
        <div className="flex flex-col gap-0.5 font-sans">
          {/* Inicio */}
          <button
            className={`flex items-center gap-2.5 px-3 py-1.5 rounded-xl text-[12px] transition-all group ${
              activeView === "home" && !activeChatId
                ? "bg-white/10 text-white font-medium"
                : "text-zinc-400 hover:bg-white/5 hover:text-white"
            }`}
            onClick={() => {
              setActiveChat("");
              setActiveView("home");
            }}
          >
            <span className="material-symbols-outlined text-[18px] text-zinc-400 group-hover:text-lime-400">home</span>
            <span>Inicio</span>
          </button>

          {/* Proyectos */}
          <button
            className={`flex items-center gap-2.5 px-3 py-1.5 rounded-xl text-[12px] transition-all group ${
              activeView === "projects"
                ? "bg-white/10 text-white font-medium"
                : "text-zinc-400 hover:bg-white/5 hover:text-white"
            }`}
            onClick={() => setActiveView("projects")}
          >
            <span className="material-symbols-outlined text-[18px] text-zinc-400 group-hover:text-lime-400">inventory_2</span>
            <span>Proyectos</span>
          </button>

          {/* Artefactos */}
          <button
            className={`flex items-center gap-2.5 px-3 py-1.5 rounded-xl text-[12px] transition-all group ${
              activeView === "chat"
                ? "bg-white/10 text-white font-medium"
                : "text-zinc-400 hover:bg-white/5 hover:text-white"
            }`}
            onClick={() => {
              setActiveChat("");
              setActiveView("chat");
            }}
          >
            <span className="material-symbols-outlined text-[18px] text-zinc-400 group-hover:text-lime-400">schema</span>
            <span>Artefactos</span>
          </button>

          {/* Personalizar (SettingsModal) */}
          <button
            className="flex items-center gap-2.5 px-3 py-1.5 rounded-xl text-[12px] transition-all text-zinc-400 hover:bg-white/5 hover:text-white group"
            onClick={() => useUIStore.getState().openSettings("habilidades")}
          >
            <span className="material-symbols-outlined text-[18px] text-zinc-400 group-hover:text-lime-400">settings</span>
            <span>Personalizar</span>
          </button>
        </div>

        <div className="flex-1 overflow-hidden flex flex-col min-h-0">
          <div className="flex items-center justify-between px-3 mb-2 text-zinc-400 text-[11px] font-sans font-medium relative">
            <button className="flex items-center gap-1 hover:text-white transition-colors">
              <span>Recientes</span>
              <span className="material-symbols-outlined text-[14px]">expand_more</span>
            </button>

            <div className="flex items-center gap-1">
              <button
                className="p-1 rounded-md hover:bg-white/5 hover:text-white transition-colors text-zinc-400"
                onClick={() => setActiveView("chats")}
                title="Ver chats y tareas"
              >
                <span className="material-symbols-outlined text-[16px]">north_east</span>
              </button>

              <div className="relative" ref={filterMenuRef}>
                <button
                  className={`p-1 rounded-md transition-colors ${
                    showFilterMenu ? "bg-white/10 text-white" : "hover:bg-white/5 hover:text-white text-zinc-400"
                  }`}
                  onClick={() => setShowFilterMenu(!showFilterMenu)}
                  title="Filtros y opciones"
                >
                  <span className="material-symbols-outlined text-[16px]">tune</span>
                </button>

                {showFilterMenu && (
                  <div
                    className="fixed left-[245px] top-[210px] w-48 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1 z-[9999] text-[13px] text-white select-none"
                    onMouseLeave={() => setActiveSubmenu(null)}
                  >
                    <div className="relative" onMouseEnter={() => setActiveSubmenu("tipo")}>
                      <button className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left">
                        <span>Tipo</span>
                        <div className="flex items-center gap-1 text-white/40 text-[12px]">
                          <span>{typeFilter}</span>
                          <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                        </div>
                      </button>
                      {activeSubmenu === "tipo" && (
                        <div className="fixed left-[430px] top-[210px] pl-2 z-[9999]">
                          <div className="w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1">
                            {["Todo", "Chat", "Tarea"].map((opt) => (
                              <button
                                key={opt}
                                className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left"
                                onClick={() => {
                                  setTypeFilter(opt);
                                  setShowFilterMenu(false);
                                  setActiveSubmenu(null);
                                }}
                              >
                                <span>{opt}</span>
                                {typeFilter === opt && <span className="material-symbols-outlined text-[16px] text-lime-400">check</span>}
                              </button>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto flex flex-col gap-0.5 scrollbar-thin">
            {visibleChats.length === 0 ? (
              <div className="px-3 py-4 text-center text-[11px] text-zinc-500 font-sans">
                Sin conversaciones recientes
              </div>
            ) : (
              visibleChats.map((c) => (
                <div
                  key={c.id}
                  className={`flex items-center justify-between px-3 py-1.5 rounded-lg text-[12px] text-left transition-colors truncate group cursor-pointer ${
                    activeChatId === c.id
                      ? "bg-white/10 text-white font-medium"
                      : "text-zinc-400 hover:bg-white/5 hover:text-white"
                  }`}
                  onClick={() => {
                    setActiveChat(c.id);
                    setActiveView("home");
                  }}
                >
                  <span className="truncate pr-2 flex-1">{c.title || "Nueva conversación"}</span>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      useChatStore.getState().deleteChat(c.id);
                    }}
                    className="opacity-0 group-hover:opacity-100 p-0.5 rounded hover:text-rose-400 hover:bg-white/10 transition-all text-neutral-500 shrink-0"
                    title="Eliminar conversación"
                  >
                    <span className="material-symbols-outlined text-[14px]">delete</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Onboarding Checklist progress (if active) */}
        {onboardingStep <= 2 && (
          <div className="bg-[#202020] border border-white/5 rounded-xl p-2.5 flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-semibold text-white/90">Comenzar con Ozy</span>
              <span className="text-[10px] text-zinc-400 font-mono">{onboardingStep}/{ONBOARDING_TOTAL_STEPS}</span>
            </div>
            <div className="flex flex-col gap-1">
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1 rounded-lg transition-colors text-[11px] ${
                  onboardingStep > 0 ? "opacity-50 line-through text-zinc-500" : "hover:bg-white/5 text-zinc-300"
                }`}
                onClick={() => { setOnboardingStep(0); setActiveView("onboarding"); }}
                disabled={onboardingStep > 0}
              >
                <span className="material-symbols-outlined text-[13px]">
                  {onboardingStep > 0 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span>Importar memoria</span>
              </button>
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1 rounded-lg transition-colors text-[11px] ${
                  onboardingStep > 1 ? "opacity-50 line-through text-zinc-500" : "hover:bg-white/5 text-zinc-300"
                }`}
                onClick={() => { setOnboardingStep(1); setActiveView("onboarding"); }}
                disabled={onboardingStep > 1}
              >
                <span className="material-symbols-outlined text-[13px]">
                  {onboardingStep > 1 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span>Feedback IA</span>
              </button>
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1 rounded-lg transition-colors text-[11px] ${
                  onboardingStep > 2 ? "opacity-50 line-through text-zinc-500" : "hover:bg-white/5 text-zinc-300"
                }`}
                onClick={() => { setOnboardingStep(2); setActiveView("onboarding"); }}
                disabled={onboardingStep > 2}
              >
                <span className="material-symbols-outlined text-[13px]">
                  {onboardingStep > 2 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span>Instrucciones</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Bottom Profile Bar with Popup Menu */}
      <div className="mt-auto pt-2 pb-2 px-2 border-t border-white/10 flex flex-col relative" ref={profileMenuRef}>
        {showProfileMenu && (
          <div className="absolute bottom-full mb-2 left-2 w-64 bg-[#222222] border border-white/10 rounded-2xl shadow-2xl p-1.5 z-[9999] text-white text-[12px] animate-fadeIn select-none font-sans">
            <div className="px-3 py-2 text-white font-medium text-[13px] border-b border-white/10 truncate flex items-center justify-between">
              <span className="truncate">{user?.name || "Angel"}</span>
              <span className="text-[10px] bg-lime-500/20 text-lime-400 px-1.5 py-0.5 rounded font-mono shrink-0 ml-2">Local</span>
            </div>

            <div className="py-1">
              <button
                className="w-full flex items-center justify-between px-3 py-1.5 rounded-lg hover:bg-white/10 transition-colors text-left"
                onClick={() => {
                  setShowProfileMenu(false);
                  useUIStore.getState().openSettings("general");
                }}
              >
                <div className="flex items-center gap-2">
                  <span className="material-symbols-outlined text-[16px] text-zinc-400">settings</span>
                  <span>Configuración</span>
                </div>
                <span className="text-[10px] text-zinc-500 font-mono">Ctrl+,</span>
              </button>

              <button
                className="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-white/10 transition-colors text-left"
                onClick={() => {
                  setShowProfileMenu(false);
                  useAuthStore.getState().lockSession();
                }}
              >
                <span className="material-symbols-outlined text-[16px] text-[#d1f107]">switch_account</span>
                <span>Cambiar de Perfil</span>
              </button>

              <button
                className="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-white/10 transition-colors text-left"
                onClick={() => {
                  setShowProfileMenu(false);
                  setShowPinModal(true);
                }}
              >
                <span className="material-symbols-outlined text-[16px] text-amber-400">lock</span>
                <span>Seguridad (PIN de 6 dígitos)</span>
              </button>

              <button
                className="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-white/10 transition-colors text-left"
                onClick={() => {
                  setShowProfileMenu(false);
                  useAuthStore.getState().lockSession();
                }}
              >
                <span className="material-symbols-outlined text-[16px] text-zinc-400">lock_clock</span>
                <span>Bloquear Sesión</span>
              </button>
            </div>

            <div className="h-px bg-white/10 my-0.5" />

            <div className="py-0.5">
              <button
                className="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-rose-500/20 text-rose-400 transition-colors text-left"
                onClick={() => {
                  setShowProfileMenu(false);
                  useAuthStore.getState().logout();
                }}
              >
                <span className="material-symbols-outlined text-[16px]">logout</span>
                <span>Cerrar sesión</span>
              </button>
            </div>
          </div>
        )}

        <button
          className="flex items-center gap-2 text-zinc-400 hover:text-white transition-colors w-full px-2 py-1.5 rounded-xl hover:bg-white/5"
          onClick={() => setShowProfileMenu(!showProfileMenu)}
        >
          <div
            style={{
              backgroundColor: user?.avatarColor ? `${user.avatarColor}25` : "#d1f10725",
              color: user?.avatarColor || "#d1f107",
            }}
            className="w-6 h-6 rounded-lg border border-white/10 flex items-center justify-center text-[10px] font-bold uppercase font-mono"
          >
            {user?.initials || "OZ"}
          </div>
          <span className="text-[13px] font-medium truncate text-white/90">{user?.name || "Angel"}</span>
          <span className="text-[12px] text-zinc-500">
            · {user?.plan === "free" ? "Free" : "Pro"}
          </span>
          {user?.hasPin && (
            <span className="material-symbols-outlined text-[13px] text-amber-400 ml-0.5">lock</span>
          )}
          <span className="material-symbols-outlined text-[16px] ml-auto text-zinc-500">
            expand_more
          </span>
        </button>
      </div>

      <SecurityPinModal
        isOpen={showPinModal}
        onClose={() => setShowPinModal(false)}
      />
    </nav>
  );
}
