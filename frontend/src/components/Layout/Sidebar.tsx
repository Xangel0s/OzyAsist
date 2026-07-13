import { useState } from "react";
import { useUIStore, type View } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useAuthStore } from "../../store/authStore";

const navItems: { view: View; label: string; icon: string }[] = [
  { view: "home", label: "Home", icon: "home" },
  { view: "code", label: "Code", icon: "code" },
  { view: "chat", label: "Chat", icon: "chat" },
  { view: "projects", label: "Proyectos", icon: "inventory_2" },
  { view: "skills", label: "Skills", icon: "settings_suggest" },
  { view: "connectors", label: "Connectors", icon: "power" },
];

export default function Sidebar() {
  const activeView = useUIStore((s) => s.activeView);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const onboardingStep = useUIStore((s) => s.onboardingStep);
  const setOnboardingStep = useUIStore((s) => s.setOnboardingStep);
  const chats = useChatStore((s) => s.chats);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const createChat = useChatStore((s) => s.createChat);
  const user = useAuthStore((s) => s.user);
  const [sidebarMode, setSidebarMode] = useState(false);

  const handleNavClick = (view: View) => {
    setActiveView(view);
  };

  const handleNewChat = () => {
    createChat("chat");
  };

  return (
    <nav className="bg-surface-container-low border-r border-border-subtle flex flex-col h-full flex-shrink-0 w-[260px] relative z-10">
      <div className="flex flex-col p-4 gap-3 h-full">
        <div className="flex gap-1 mb-1 bg-surface-elevated p-1 rounded-xl border border-border-subtle">
          <button
            className={`flex-1 rounded-lg py-1.5 px-3 flex items-center justify-center gap-2 transition-all text-[13px] font-medium ${
              activeView === "home"
                ? "bg-surface-container-high text-on-surface"
                : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
            }`}
            onClick={() => handleNavClick("home")}
          >
            <span className="material-symbols-outlined text-[18px]">home</span>
            <span>Home</span>
          </button>
          <button
            className={`flex-1 rounded-lg py-1.5 px-3 flex items-center justify-center gap-2 transition-all text-[13px] font-medium ${
              activeView === "code"
                ? "bg-surface-container-high text-on-surface"
                : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
            }`}
            onClick={() => handleNavClick("code")}
          >
            <span className="material-symbols-outlined text-[18px]">code</span>
            <span>Code</span>
          </button>
        </div>

        <button
          className="w-full flex items-center gap-2 px-3 py-2 bg-primary-container text-on-primary hover:bg-primary-fixed-dim rounded-lg transition-all text-body-md font-medium"
          onClick={handleNewChat}
        >
          <span className="material-symbols-outlined text-[18px]">add</span>
          <span>+ Nuevo</span>
        </button>

        <div className="flex flex-col gap-0.5">
          {navItems.slice(2).map((item) => (
            <button
              key={item.view}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-all text-body-md group ${
                activeView === item.view
                  ? "text-on-surface bg-surface-variant"
                  : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
              }`}
              onClick={() => handleNavClick(item.view)}
            >
              <span
                className={`material-symbols-outlined text-[18px] transition-colors ${
                  activeView === item.view
                    ? "text-primary-container"
                    : "group-hover:text-primary-container"
                }`}
              >
                {item.icon}
              </span>
              <span>{item.label}</span>
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-hidden flex flex-col">
          <div className="flex items-center justify-between px-3 mb-2 text-text-muted text-label-caps">
            <span>Recientes</span>
            <button className="hover:text-on-surface transition-colors">
              <span className="material-symbols-outlined text-[14px]">tune</span>
            </button>
          </div>
          <div className="flex-1 overflow-y-auto pr-2 flex flex-col gap-0.5">
            {chats.map((chat) => (
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

        {onboardingStep < 3 && (
          <div className="bg-surface-elevated rounded-xl border border-border-subtle p-4 flex flex-col gap-3">
            <div className="flex justify-between items-center text-[13px]">
              <span className="font-medium text-on-surface">
                Comenzar con Ozy
              </span>
              <span className="text-text-muted">{onboardingStep} / 3</span>
            </div>
            <div className="h-1 w-full bg-surface-variant rounded-full overflow-hidden">
              <div
                className="h-full bg-primary-container transition-all"
                style={{ width: `${(onboardingStep / 3) * 100}%` }}
              />
            </div>
            <div className="flex flex-col gap-3 mt-1">
              <button
                className={`flex items-start gap-3 text-left w-full ${
                  onboardingStep > 0 ? "opacity-50 line-through" : "cursor-pointer hover:opacity-80"
                }`}
                onClick={() => { setOnboardingStep(0); setActiveView("onboarding"); }}
                disabled={onboardingStep > 0}
              >
                <span
                  className={`material-symbols-outlined text-[18px] shrink-0 ${
                    onboardingStep > 0 ? "text-primary-container fill" : "text-text-muted"
                  }`}
                >
                  {onboardingStep > 0 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span className="text-[13px] leading-tight">
                  Trae tu memoria de otra IA
                </span>
              </button>
              <button
                className={`flex items-start gap-3 text-left w-full ${
                  onboardingStep > 1 ? "opacity-50 line-through" : "cursor-pointer hover:opacity-80"
                }`}
                onClick={() => { setOnboardingStep(1); setActiveView("onboarding"); }}
                disabled={onboardingStep > 1}
              >
                <span
                  className={`material-symbols-outlined text-[18px] shrink-0 ${
                    onboardingStep > 1 ? "text-primary-container fill" : "text-text-muted"
                  }`}
                >
                  {onboardingStep > 1 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <span className="text-[13px] leading-tight">
                  Recibe feedback de IA
                </span>
              </button>
              <button
                className={`flex items-start gap-3 text-left w-full ${
                  onboardingStep > 2 ? "opacity-50 line-through" : "cursor-pointer hover:opacity-80"
                }`}
                onClick={() => { setOnboardingStep(2); setActiveView("onboarding"); }}
                disabled={onboardingStep > 2}
              >
                <span
                  className={`material-symbols-outlined text-[18px] shrink-0 ${
                    onboardingStep > 2 ? "text-primary-container fill" : "text-text-muted"
                  }`}
                >
                  {onboardingStep > 2 ? "check_circle" : "radio_button_unchecked"}
                </span>
                <div className="flex flex-col">
                  <span className="text-[13px] font-medium text-on-surface leading-tight">
                    Define tus instrucciones
                  </span>
                  <span className="text-[12px] text-text-muted leading-snug mt-1">
                    Reglas y preferencias que Ozy recordará
                  </span>
                </div>
              </button>
            </div>
          </div>
        )}

        <button
          className={`flex items-center gap-2 px-3 py-2 rounded-lg transition-all text-[13px] w-full ${
            sidebarMode
              ? "text-primary-container bg-surface-variant"
              : "text-text-muted hover:text-on-surface hover:bg-surface-variant"
          }`}
          onClick={() => setSidebarMode(!sidebarMode)}
        >
          <span className="material-symbols-outlined text-[18px]">visibility</span>
          <span>Modo Sidebar</span>
          <span className={`ml-auto w-8 h-4 rounded-full transition-colors relative ${
            sidebarMode ? "bg-primary-container" : "bg-surface-variant"
          }`}>
            <div className={`w-3 h-3 rounded-full bg-white absolute top-0.5 transition-all ${
              sidebarMode ? "left-[18px]" : "left-0.5"
            }`} />
          </span>
        </button>

        <div className="mt-auto pt-4 border-t border-border-subtle flex items-center justify-between">
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
