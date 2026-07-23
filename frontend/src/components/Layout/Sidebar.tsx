import { useState, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useAuthStore } from "../../store/authStore";
import { useOnClickOutside } from "../../hooks";

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
  const [activeSubmenu, setActiveSubmenu] = useState<"tipo" | "estado" | "actividad" | "agrupar" | null>(null);
  const [typeFilter, setTypeFilter] = useState("Todo");
  const [statusFilter, setStatusFilter] = useState("Activo");
  const [activityFilter, setActivityFilter] = useState("Todo");
  const [groupByFilter, setGroupByFilter] = useState("Ninguno");
  const filterMenuRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(filterMenuRef, () => {
    setShowFilterMenu(false);
    setActiveSubmenu(null);
  });

  const visibleChats = chats.filter((c) => {
    const isNotEmpty = c.messages.length > 0 || c.id === activeChatId;
    const matchesType =
      typeFilter === "Todo" ||
      (typeFilter === "Chat" && c.mode === "chat") ||
      (typeFilter === "Tarea" && c.mode === "code");
    return isNotEmpty && matchesType;
  });

  const handleNewChat = async () => {
    setActiveChat("");
    setActiveView("home");
  };

  return (
    <nav
      className={`bg-surface-container-low border-r border-border-subtle flex flex-col h-full flex-shrink-0 transition-all duration-300 ease-in-out relative z-10 overflow-hidden ${
        sidebarOpen ? "w-[260px] opacity-100" : "w-0 opacity-0 p-0 border-none pointer-events-none"
      }`}
    >
      <div className="p-3 flex flex-col gap-2">
        {/* Top Pill Toggle: Inicio | Code */}
        <div className="flex items-center gap-1 bg-surface-variant p-1 rounded-xl text-[12px]">
          <button
            className={`flex-1 py-1.5 px-3 rounded-lg font-medium transition-all flex items-center justify-center gap-1.5 ${
              activeView === "home" ? "bg-background text-on-surface shadow-sm font-semibold" : "text-text-muted hover:text-on-surface"
            }`}
            onClick={() => setActiveView("home")}
          >
            <span className="material-symbols-outlined text-[16px]">home</span>
            <span>Inicio</span>
          </button>
          <button
            className={`flex-1 py-1.5 px-3 rounded-lg font-medium transition-all flex items-center justify-center gap-1.5 ${
              activeView === "code" ? "bg-background text-on-surface shadow-sm font-semibold" : "text-text-muted hover:text-on-surface"
            }`}
            onClick={() => setActiveView("code")}
          >
            <span className="material-symbols-outlined text-[16px]">code</span>
            <span>Code</span>
          </button>
        </div>

        {/* + Nuevo Button with OzyAsist Vibrant Electric Neon Lime Color */}
        <button
          className="w-full flex items-center justify-center gap-2 px-3 py-2.5 bg-[#d1f107] text-[#181e00] rounded-xl font-bold shadow-sm hover:opacity-90 transition-all text-[13px]"
          onClick={handleNewChat}
        >
          <span className="material-symbols-outlined text-[18px]">add</span>
          <span>Nuevo</span>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-3 flex flex-col gap-4 min-h-0">
        <div className="flex flex-col gap-0.5">
          {/* Proyectos */}
          <button
            className={`flex items-center gap-3 px-3 py-2 rounded-xl text-[13px] transition-all group ${
              activeView === "projects"
                ? "bg-surface-variant text-on-surface font-medium"
                : "text-text-muted hover:bg-surface-variant/50 hover:text-on-surface"
            }`}
            onClick={() => setActiveView("projects")}
          >
            <span className="material-symbols-outlined text-[20px]">inventory_2</span>
            <span>Proyectos</span>
          </button>

          {/* Artefactos */}
          <button
            className={`flex items-center gap-3 px-3 py-2 rounded-xl text-[13px] transition-all group ${
              activeView === "chat"
                ? "bg-surface-variant text-on-surface font-medium"
                : "text-text-muted hover:bg-surface-variant/50 hover:text-on-surface"
            }`}
            onClick={() => {
              setActiveChat("");
              setActiveView("chat");
            }}
          >
            <span className="material-symbols-outlined text-[20px]">schema</span>
            <span>Artefactos</span>
          </button>

          {/* Personalizar (Abre Configuración / SettingsModal) */}
          <button
            className="flex items-center gap-3 px-3 py-2 rounded-xl text-[13px] transition-all text-text-muted hover:bg-surface-variant/50 hover:text-on-surface group"
            onClick={() => useUIStore.getState().openSettings("habilidades")}
          >
            <span className="material-symbols-outlined text-[20px]">work</span>
            <span>Personalizar</span>
          </button>
        </div>

        <div className="flex-1 overflow-hidden flex flex-col min-h-0">
          <div className="flex items-center justify-between px-3 mb-2 text-text-muted text-label-caps relative">
            <button className="flex items-center gap-1 hover:text-on-surface transition-colors font-medium">
              <span>Recientes</span>
              <span className="material-symbols-outlined text-[14px]">expand_more</span>
            </button>

            <div className="flex items-center gap-1">
              {/* Button 1: Pop-out Arrow ↗ (Navigates directly to "Chats y tareas" list) */}
              <button
                className="p-1 rounded-md hover:bg-surface-variant hover:text-on-surface transition-colors text-text-muted"
                onClick={() => {
                  setActiveChat("");
                  setActiveView("chat");
                }}
                title="Ver conversaciones"
              >
                <span className="material-symbols-outlined text-[16px]">north_east</span>
              </button>

              {/* Button 2: Multi-level Tune Filter Menu (Fixed Positioning Outside Sidebar) */}
              <div className="relative" ref={filterMenuRef}>
                <button
                  className={`p-1 rounded-md transition-colors ${
                    showFilterMenu ? "bg-surface-variant text-on-surface" : "hover:bg-surface-variant hover:text-on-surface text-text-muted"
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
                    {/* Category 1: Tipo */}
                    <div className="relative" onMouseEnter={() => setActiveSubmenu("tipo")}>
                      <button className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left">
                        <span>Tipo</span>
                        <div className="flex items-center gap-1 text-white/40 text-[12px]">
                          <span>{typeFilter}</span>
                          <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                        </div>
                      </button>
                      {activeSubmenu === "tipo" && (
                        <div className="fixed left-[437px] top-[210px] w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1 z-[9999]">
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
                              {typeFilter === opt && <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>

                    {/* Category 2: Estado */}
                    <div className="relative" onMouseEnter={() => setActiveSubmenu("estado")}>
                      <button className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left">
                        <span>Estado</span>
                        <div className="flex items-center gap-1 text-white/40 text-[12px]">
                          <span>{statusFilter}</span>
                          <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                        </div>
                      </button>
                      {activeSubmenu === "estado" && (
                        <div className="fixed left-[437px] top-[240px] w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1 z-[9999]">
                          {["Activo", "Archivado", "Todo"].map((opt) => (
                            <button
                              key={opt}
                              className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left"
                              onClick={() => {
                                setStatusFilter(opt);
                                setShowFilterMenu(false);
                                setActiveSubmenu(null);
                              }}
                            >
                              <span>{opt}</span>
                              {statusFilter === opt && <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>

                    {/* Category 3: Última actividad */}
                    <div className="relative" onMouseEnter={() => setActiveSubmenu("actividad")}>
                      <button className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left">
                        <span>Última actividad</span>
                        <div className="flex items-center gap-1 text-white/40 text-[12px]">
                          <span>{activityFilter}</span>
                          <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                        </div>
                      </button>
                      {activeSubmenu === "actividad" && (
                        <div className="fixed left-[437px] top-[270px] w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1 z-[9999]">
                          {["1d", "3d", "7d", "30d", "Todo"].map((opt) => (
                            <button
                              key={opt}
                              className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left"
                              onClick={() => {
                                setActivityFilter(opt);
                                setShowFilterMenu(false);
                                setActiveSubmenu(null);
                              }}
                            >
                              <span>{opt}</span>
                              {activityFilter === opt && <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>

                    <div className="h-px bg-white/10 my-1" />

                    {/* Category 4: Agrupar por */}
                    <div className="relative" onMouseEnter={() => setActiveSubmenu("agrupar")}>
                      <button className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left">
                        <span>Agrupar por</span>
                        <div className="flex items-center gap-1 text-white/40 text-[12px]">
                          <span>{groupByFilter}</span>
                          <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                        </div>
                      </button>
                      {activeSubmenu === "agrupar" && (
                        <div className="fixed left-[437px] top-[310px] w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1 z-[9999]">
                          {["Ninguno", "Fecha", "Proyecto"].map((opt) => (
                            <button
                              key={opt}
                              className="w-full flex items-center justify-between px-3 py-1.5 hover:bg-white/10 transition-colors text-left"
                              onClick={() => {
                                setGroupByFilter(opt);
                                setShowFilterMenu(false);
                                setActiveSubmenu(null);
                              }}
                            >
                              <span>{opt}</span>
                              {groupByFilter === opt && <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto pr-2 flex flex-col gap-0.5">
            {visibleChats.map((chat) => (
              <button
                key={chat.id}
                className={`flex items-center gap-3 px-3 py-1.5 rounded-lg transition-all group truncate text-[13px] w-full text-left ${
                  chat.id === activeChatId && (activeView === "chat" || activeView === "code")
                    ? "text-on-surface bg-surface-variant"
                    : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
                }`}
                    onClick={() => {
                      setActiveChat(chat.id);
                    }}
              >
                <span className="material-symbols-outlined text-[16px] shrink-0 opacity-70 group-hover:opacity-100">
                  chat_bubble
                </span>
                <span className="truncate">{chat.title}</span>
              </button>
            ))}
          </div>
        </div>

        {onboardingStep < ONBOARDING_TOTAL_STEPS && (
          <div className="bg-surface-elevated rounded-xl border border-border-subtle p-3 mt-2">
            <div className="flex items-center justify-between text-[12px] mb-2">
              <span className="font-medium text-on-surface">Comenzar con Ozy</span>
              <span className="text-text-muted">{onboardingStep}/{ONBOARDING_TOTAL_STEPS}</span>
            </div>
            <div className="h-1 w-full bg-surface-variant rounded-full overflow-hidden mb-3">
              <div
                className="h-full bg-primary-container transition-all"
                style={{ width: `${(onboardingStep / ONBOARDING_TOTAL_STEPS) * 100}%` }}
              />
            </div>
            <div className="flex flex-col gap-1">
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1.5 rounded-lg transition-colors ${
                  onboardingStep > 0 ? "opacity-50 line-through" : "hover:bg-surface-variant"
                }`}
                onClick={() => { setOnboardingStep(0); setActiveView("onboarding"); }}
                disabled={onboardingStep > 0}
              >
                <span className={`material-symbols-outlined text-[14px] shrink-0 ${onboardingStep > 0 ? "text-primary-container fill" : "text-text-muted"}`}>
                  {onboardingStep > 0 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span className="text-[12px] text-on-surface">Importar memoria</span>
              </button>
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1.5 rounded-lg transition-colors ${
                  onboardingStep > 1 ? "opacity-50 line-through" : "hover:bg-surface-variant"
                }`}
                onClick={() => { setOnboardingStep(1); setActiveView("onboarding"); }}
                disabled={onboardingStep > 1}
              >
                <span className={`material-symbols-outlined text-[14px] shrink-0 ${onboardingStep > 1 ? "text-primary-container fill" : "text-text-muted"}`}>
                  {onboardingStep > 1 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span className="text-[12px] text-on-surface">Feedback IA</span>
              </button>
              <button
                className={`flex items-center gap-2 text-left w-full px-2 py-1.5 rounded-lg transition-colors ${
                  onboardingStep > 2 ? "opacity-50 line-through" : "hover:bg-surface-variant"
                }`}
                onClick={() => { setOnboardingStep(2); setActiveView("onboarding"); }}
                disabled={onboardingStep > 2}
              >
                <span className={`material-symbols-outlined text-[14px] shrink-0 ${onboardingStep > 2 ? "text-primary-container fill" : "text-text-muted"}`}>
                  {onboardingStep > 2 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span className="text-[12px] text-on-surface">Instrucciones</span>
              </button>
            </div>
          </div>
        )}



        <div className="mt-auto pt-3 pb-4 my-1 px-1 border-t border-border-subtle flex items-center justify-between">
          <button
            className="flex items-center gap-2 text-text-muted hover:text-on-surface transition-colors w-full"
            onClick={() => setActiveView("onboarding")}
          >
            <div className="w-6 h-6 rounded-md bg-surface-variant flex items-center justify-center text-[10px] font-bold text-on-surface">
              {user?.initials}
            </div>
            <span className="text-[13px] font-medium">{user?.name}</span>
            <span className="text-[12px] text-text-muted ml-1">
              · {user?.plan === "free" ? "Free" : "Pro"}
            </span>
            <span className="material-symbols-outlined text-[16px] ml-auto">
              expand_more
            </span>
          </button>
        </div>
      </div>
    </nav>
  );
}
