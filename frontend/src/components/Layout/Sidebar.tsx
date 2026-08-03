import { useState, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useAuthStore } from "../../store/authStore";
import { useToastStore } from "../../store/toastStore";
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
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const [activeProfileSubmenu, setActiveProfileSubmenu] = useState<"idioma" | "ayuda" | "mas_info" | null>(null);
  const [activeSubmenu, setActiveSubmenu] = useState<"tipo" | "estado" | "actividad" | "agrupar" | null>(null);
  const [typeFilter, setTypeFilter] = useState("Todo");
  const [statusFilter, setStatusFilter] = useState("Activo");
  const [activityFilter, setActivityFilter] = useState("Todo");
  const [groupByFilter, setGroupByFilter] = useState("Ninguno");
  const filterMenuRef = useRef<HTMLDivElement>(null);
  const profileMenuRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(filterMenuRef, () => {
    setShowFilterMenu(false);
    setActiveSubmenu(null);
  });

  useOnClickOutside(profileMenuRef, () => {
    setShowProfileMenu(false);
    setActiveProfileSubmenu(null);
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
                onClick={() => setActiveView("chats")}
                title="Ver chats y tareas"
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
                                {typeFilter === opt && <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>}
                              </button>
                            ))}
                          </div>
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
                        <div className="fixed left-[430px] top-[240px] pl-2 z-[9999]">
                          <div className="w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1">
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
                        <div className="fixed left-[430px] top-[270px] pl-2 z-[9999]">
                          <div className="w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1">
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
                        <div className="fixed left-[430px] top-[310px] pl-2 z-[9999]">
                          <div className="w-36 bg-[#282828] border border-white/10 rounded-xl shadow-2xl py-1">
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
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto pr-2 flex flex-col gap-0.5 custom-scrollbar">
            {visibleChats.slice(0, 10).map((chat) => (
              <button
                key={chat.id}
                className={`flex items-center gap-3 px-3 py-1.5 rounded-lg transition-all group truncate text-[13px] w-full text-left ${
                  chat.id === activeChatId && (activeView === "chat" || activeView === "code")
                    ? "text-on-surface bg-surface-variant"
                    : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
                }`}
                onClick={() => {
                  setActiveChat(chat.id);
                  if (activeView !== "code") setActiveView("chat");
                }}
              >
                <span className="material-symbols-outlined text-[16px] shrink-0 opacity-70 group-hover:opacity-100">
                  {chat.mode === "code" ? "code" : "chat_bubble"}
                </span>
                <span className="truncate">{chat.title}</span>
              </button>
            ))}
            {visibleChats.length > 10 && (
              <button
                className="flex items-center gap-2 px-3 py-2 text-[13px] text-text-muted hover:text-on-surface hover:bg-surface-variant/50 rounded-lg transition-all w-full text-left mt-1 font-medium group"
                onClick={() => setActiveView("chats")}
              >
                <span>Ver todo</span>
              </button>
            )}
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
      </div>

        <div className="mt-auto pt-3 pb-3 my-1 px-1 border-t border-border-subtle flex flex-col relative" ref={profileMenuRef}>
          {showProfileMenu && (
            <div className="absolute bottom-full mb-2 left-0 w-64 bg-[#222222] border border-white/10 rounded-2xl shadow-2xl p-1.5 z-[9999] text-white text-[13px] animate-fadeIn select-none">
              {/* Registered Username Header */}
              <div className="px-3 py-2 text-white font-medium text-[13px] border-b border-white/10 truncate flex items-center justify-between">
                <span className="truncate">{user?.name || "Usuario OzyAssist"}</span>
                <span className="text-[10px] bg-[#d1f107]/20 text-[#d1f107] px-1.5 py-0.5 rounded font-mono shrink-0 ml-2">Open Source</span>
              </div>

              {/* Group 1: Config, Idioma, Ayuda */}
              <div className="py-1">
                <button
                  className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 transition-colors text-left"
                  onClick={() => {
                    setShowProfileMenu(false);
                    useUIStore.getState().openSettings("general");
                  }}
                >
                  <div className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-[18px] opacity-70">settings</span>
                    <span>Configuración</span>
                  </div>
                  <span className="text-[11px] text-white/40 font-mono">Ctrl+,</span>
                </button>

                {/* Submenu Trigger: Idioma */}
                <div className="relative" onMouseEnter={() => setActiveProfileSubmenu("idioma")}>
                  <button
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors text-left ${
                      activeProfileSubmenu === "idioma" ? "bg-white/10 text-white" : "hover:bg-white/10"
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] opacity-70">language</span>
                      <span>Idioma</span>
                    </div>
                    <span className="material-symbols-outlined text-[16px] text-white/40">chevron_right</span>
                  </button>

                  {activeProfileSubmenu === "idioma" && (
                    <div className="absolute left-[260px] bottom-0 w-52 bg-[#282828] border border-white/10 rounded-2xl shadow-2xl py-1 px-1 z-[10000] text-[13px] animate-fadeIn">
                      <button
                        className="w-full flex items-center justify-between px-3 py-1.5 rounded-lg hover:bg-white/10 text-left font-medium text-[#d1f107]"
                        onClick={() => {
                          useToastStore.getState().show("Idioma activo: Español", "success");
                          setActiveProfileSubmenu(null);
                        }}
                      >
                        <span>Español</span>
                        <span className="material-symbols-outlined text-[16px]">check</span>
                      </button>
                      <button
                        className="w-full flex items-center justify-between px-3 py-1.5 rounded-lg hover:bg-white/10 text-left text-white/60"
                        onClick={() => {
                          useToastStore.getState().show("English language coming soon", "info");
                          setActiveProfileSubmenu(null);
                        }}
                      >
                        <span>English</span>
                      </button>
                    </div>
                  )}
                </div>

                {/* Submenu Trigger: Obtener ayuda */}
                <div className="relative" onMouseEnter={() => setActiveProfileSubmenu("ayuda")}>
                  <button
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors text-left ${
                      activeProfileSubmenu === "ayuda" ? "bg-white/10 text-white" : "hover:bg-white/10"
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] opacity-70">help_outline</span>
                      <span>Obtener ayuda</span>
                    </div>
                    <span className="material-symbols-outlined text-[16px] text-white/40">chevron_right</span>
                  </button>

                  {activeProfileSubmenu === "ayuda" && (
                    <div className="absolute left-[260px] bottom-0 w-56 bg-[#282828] border border-white/10 rounded-2xl shadow-2xl py-1 px-1 z-[10000] text-[13px] animate-fadeIn">
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          setShowProfileMenu(false);
                          setActiveProfileSubmenu(null);
                          useUIStore.getState().setActiveView("onboarding");
                        }}
                      >
                        <div className="flex items-center gap-2">
                          <span className="material-symbols-outlined text-[16px] opacity-70">school</span>
                          <span>Guía de inicio</span>
                        </div>
                      </button>
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          window.open("https://github.com/Xangel0s/OzyAsist", "_blank");
                        }}
                      >
                        <div className="flex items-center gap-2">
                          <span className="material-symbols-outlined text-[16px] opacity-70">menu_book</span>
                          <span>Documentación GitHub</span>
                        </div>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          window.open("https://github.com/Xangel0s/OzyAsist/issues", "_blank");
                        }}
                      >
                        <div className="flex items-center gap-2">
                          <span className="material-symbols-outlined text-[16px] opacity-70">bug_report</span>
                          <span>Reportar un problema</span>
                        </div>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>
                    </div>
                  )}
                </div>
              </div>

              <div className="h-px bg-white/10 my-1" />

              {/* Group 2: Extensiones & Más información */}
              <div className="py-1">
                <button
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-white/10 transition-colors text-left"
                  onClick={() => {
                    setShowProfileMenu(false);
                    useUIStore.getState().openSettings("plugins");
                  }}
                >
                  <span className="material-symbols-outlined text-[18px] opacity-70">download</span>
                  <span>Aplicaciones y extensiones</span>
                </button>

                {/* Submenu Trigger: Más información (Open Source / Anthropic style) */}
                <div className="relative" onMouseEnter={() => setActiveProfileSubmenu("mas_info")}>
                  <button
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg transition-colors text-left ${
                      activeProfileSubmenu === "mas_info" ? "bg-white/10 text-white" : "hover:bg-white/10"
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] opacity-70">info</span>
                      <span>Más información</span>
                    </div>
                    <span className="material-symbols-outlined text-[16px] text-white/40">chevron_right</span>
                  </button>

                  {activeProfileSubmenu === "mas_info" && (
                    <div className="absolute left-[260px] bottom-0 w-64 bg-[#282828] border border-white/10 rounded-2xl shadow-2xl py-1.5 px-1 z-[10000] text-[13px] animate-fadeIn">
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          window.open("https://github.com/Xangel0s/OzyAsist", "_blank");
                        }}
                      >
                        <span>Acerca de OzyAssist</span>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          useToastStore.getState().show("OzyAssist v0.1.0 - Open Source Assistant", "info");
                        }}
                      >
                        <span>Tutoriales & Cursos</span>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>
                      
                      <div className="h-px bg-white/10 my-1" />

                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          window.open("https://github.com/Xangel0s/OzyAsist/blob/main/LICENSE", "_blank");
                        }}
                      >
                        <span>Licencia Open Source</span>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>
                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          useToastStore.getState().show("Ejecución 100% Local & Privada", "success");
                        }}
                      >
                        <span>Privacidad Local</span>
                        <span className="material-symbols-outlined text-[14px] text-white/40">open_in_new</span>
                      </button>

                      <div className="h-px bg-white/10 my-1" />

                      <button
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg hover:bg-white/10 text-left"
                        onClick={() => {
                          setShowProfileMenu(false);
                          setActiveProfileSubmenu(null);
                          useUIStore.getState().setSearchOpen(true);
                        }}
                      >
                        <span>Atajos de teclado</span>
                        <span className="text-[11px] text-white/40 font-mono">Ctrl+/</span>
                      </button>
                    </div>
                  )}
                </div>
              </div>

              <div className="h-px bg-white/10 my-1" />

              {/* Group 3: Logout */}
              <div className="py-1">
                <button
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-red-500/20 text-red-400 transition-colors text-left"
                  onClick={() => {
                    setShowProfileMenu(false);
                    setActiveProfileSubmenu(null);
                    useAuthStore.getState().logout();
                  }}
                >
                  <span className="material-symbols-outlined text-[18px]">logout</span>
                  <span>Cerrar sesión</span>
                </button>
              </div>
            </div>
          )}

          <button
            className="flex items-center gap-2 text-text-muted hover:text-on-surface transition-colors w-full px-2 py-1.5 rounded-lg hover:bg-surface-variant/50"
            onClick={() => {
              setShowProfileMenu(!showProfileMenu);
              setActiveProfileSubmenu(null);
            }}
          >
            <div className="w-6 h-6 rounded-md bg-surface-variant flex items-center justify-center text-[10px] font-bold text-on-surface uppercase">
              {user?.initials || "SP"}
            </div>
            <span className="text-[13px] font-medium truncate">{user?.name || "skini peet"}</span>
            <span className="text-[12px] text-text-muted">
              · {user?.plan === "free" ? "Free" : "Pro"}
            </span>
            <span className="material-symbols-outlined text-[16px] ml-auto">
              expand_more
            </span>
          </button>
        </div>
    </nav>
  );
}
