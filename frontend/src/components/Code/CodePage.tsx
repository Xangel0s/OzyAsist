import { useState, useEffect } from "react";
import { useChatStore } from "../../store/chatStore";
import { useProjectsStore } from "../../store/projectsStore";
import { useScrollToBottom } from "../../hooks";
import ChatMessage from "../Chat/ChatMessage";
import CodeInput from "./CodeInput";
import ProjectContext from "./ProjectContext";
import AnalyzeProjectStep from "./AnalyzeProjectStep";
import ProjectList from "./ProjectList";
import ProjectChats from "./ProjectChats";
import ConsentModal from "./ConsentModal";
import BottomTerminalDrawer from "./BottomTerminalDrawer";

type CodeView = "projects" | "chats" | "chat";

export default function CodePage() {
  const [showAnalyzer, setShowAnalyzer] = useState(false);
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const consentPending = useChatStore((s) => s.consentPending);
  const resolveConsent = useChatStore((s) => s.resolveConsent);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const agentState = useChatStore((s) => s.agentState);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const activeChat = chats.find((c) => c.id === activeChatId);

  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(
    () => activeProjectId || activeChat?.projectId || null
  );
  const [view, setView] = useState<CodeView>(() => {
    if (activeChatId) return "chat";
    if (activeProjectId) return "chats";
    return "projects";
  });

  const [bypassPermissions, setBypassPermissions] = useState(true);
  const [isTerminalOpen, setIsTerminalOpen] = useState(false);

  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === selectedProjectId);

  useEffect(() => {
    if (activeChatId) {
      setView("chat");
      if (activeChat?.projectId) {
        setSelectedProjectId(activeChat.projectId);
        setActiveProject(activeChat.projectId);
      }
    }
  }, [activeChatId, activeChat?.projectId, setActiveProject]);

  const bottomRef = useScrollToBottom([
    activeChat?.messages.length,
    isResponding,
  ]);

  if (consentPending) {
    return (
      <>
        <ConsentModal
          intent={consentPending.intent}
          onResolve={(decision) => resolveConsent(decision)}
        />
        <div className="flex flex-1 h-full min-h-0">
          <main className="flex-1 flex flex-col h-full bg-[#1a1a1a] relative min-h-0">
            <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0 opacity-30 pointer-events-none">
              {activeChat?.messages.map((msg) => (
                <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
              ))}
            </div>
          </main>
          <ProjectContext />
        </div>
      </>
    );
  }

  if (showAnalyzer) {
    return (
      <AnalyzeProjectStep
        onDone={(projectId) => {
          setShowAnalyzer(false);
          setSelectedProjectId(projectId);
          setActiveProject(projectId);
          setView("chats");
        }}
      />
    );
  }

  if (view === "projects") {
    return (
      <ProjectList
        onNewProject={() => setShowAnalyzer(true)}
        onSelectProject={(projectId) => {
          setSelectedProjectId(projectId);
          setActiveProject(projectId);
          setView("chats");
        }}
      />
    );
  }

  if (view === "chats" && selectedProjectId) {
    return (
      <ProjectChats
        projectId={selectedProjectId}
        onBack={() => {
          setView("projects");
          setSelectedProjectId(null);
          setActiveProject(null);
        }}
        onSelectChat={(chatId) => {
          setActiveChat(chatId);
          setView("chat");
        }}
      />
    );
  }

  if (view === "chat" && activeChatId) {
    const handleSend = (message: string) => {
      sendMessage(activeChatId, message);
    };

    return (
      <div className="flex flex-1 h-full min-h-0 bg-[#1a1a1a] overflow-hidden">
        {/* Main Code & Chat Workspace */}
        <main className="flex-1 flex flex-col h-full bg-[#1a1a1a] relative min-h-0">
          {/* Header Bar */}
          <div className="flex items-center justify-between px-4 py-2 border-b border-white/10 bg-[#1a1a1a] z-30 font-sans select-none shrink-0">
            <div className="flex items-center gap-3 min-w-0">
              <button
                className="w-7 h-7 rounded-lg flex items-center justify-center text-zinc-400 hover:text-white hover:bg-white/5 transition-colors shrink-0"
                onClick={() => setView("chats")}
                title="Volver a los chats del proyecto"
              >
                <span className="material-symbols-outlined text-[18px]">arrow_back</span>
              </button>

              <div className="flex items-center gap-2 min-w-0">
                <div className="w-7 h-7 rounded-lg bg-[#d1f107]/15 border border-[#d1f107]/30 flex items-center justify-center text-[#d1f107] shrink-0">
                  <span className="material-symbols-outlined text-[16px]">code</span>
                </div>
                <div className="flex flex-col min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-white text-[13px] truncate">
                      {activeChat?.title || "Sesión de código"}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 text-[10px] text-zinc-400 font-mono">
                    <span className="text-[#d1f107] font-semibold">{activeProject?.name || "Proyecto"}</span>
                    <span>•</span>
                    <span className="text-zinc-500 truncate max-w-[200px]">{activeProject?.rootPath || ""}</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              {/* Terminal Toggle Button */}
              <button
                onClick={() => setIsTerminalOpen((v) => !v)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-mono transition-colors border ${
                  isTerminalOpen
                    ? "bg-[#d1f107]/15 text-[#d1f107] border-[#d1f107]/30 font-semibold"
                    : "bg-white/5 hover:bg-white/10 text-zinc-300 border-white/10"
                }`}
                title="Alternar panel de terminal inferior"
              >
                <span className="material-symbols-outlined text-[14px]">terminal</span>
                <span>Terminal</span>
              </button>

              {/* Autonomous Mode Toggle Pill in Header */}
              <button
                onClick={() => setBypassPermissions(!bypassPermissions)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-mono transition-colors border ${
                  bypassPermissions
                    ? "bg-amber-500/10 text-amber-300 border-amber-500/30 font-medium"
                    : "bg-white/5 text-zinc-400 border-white/10 hover:text-white"
                }`}
                title={bypassPermissions ? "Modo autónomo (sin pedir confirmación)" : "Modo seguro (pedir confirmación)"}
              >
                <span className={`material-symbols-outlined text-[14px] ${bypassPermissions ? "text-amber-400" : "text-zinc-500"}`}>
                  {bypassPermissions ? "bolt" : "shield"}
                </span>
                <span>{bypassPermissions ? "Autónomo" : "Seguro"}</span>
              </button>

              {/* Local status pill */}
              <div className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-white/5 border border-white/10 text-[11px] font-mono text-zinc-300">
                <span className="w-2 h-2 rounded-full bg-emerald-400" />
                <span>Local</span>
              </div>

              {/* Preview Button */}
              <button
                onClick={() => {
                  window.dispatchEvent(
                    new CustomEvent("switch-project-tab", { detail: { tab: "preview" } })
                  );
                }}
                className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-[#d1f107]/10 border border-[#d1f107]/30 text-[#d1f107] hover:bg-[#d1f107]/20 text-[11px] font-mono font-semibold transition-colors"
              >
                <span className="material-symbols-outlined text-[14px]">play_arrow</span>
                <span>Preview</span>
              </button>
            </div>
          </div>

          {/* Friendly Chat Feed */}
          <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0 bg-[#1a1a1a] scrollbar-thin">
            <div className="max-w-[850px] mx-auto flex flex-col gap-4">
              {/* Empty State with Action Cards */}
              {(!activeChat || activeChat.messages.length === 0) && !isResponding && (
                <div className="flex flex-col items-center justify-center py-12 text-center select-none">
                  <div className="w-14 h-14 rounded-2xl bg-[#d1f107]/10 border border-[#d1f107]/20 flex items-center justify-center mb-4 shadow-xl shadow-[#d1f107]/5">
                    <img src="/ozybaselogo.png" alt="Ozy" className="w-9 h-9 object-contain" />
                  </div>
                  <h2 className="text-[20px] font-bold text-white font-sans tracking-tight mb-2">
                    ¿En qué trabajamos en este proyecto?
                  </h2>
                  <p className="text-[13px] text-zinc-400 max-w-md mb-8 leading-relaxed">
                    Inspección de repositorio, resolución de incidencias, generación de tests, aplicación de diffs y ejecución de comandos en local.
                  </p>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3 w-full max-w-lg">
                    {[
                      {
                        icon: "search",
                        title: "Auditar repositorio",
                        desc: "Revisar arquitectura, dependencias y mejoras",
                        prompt: "Realiza una auditoría completa del proyecto y sugiere mejoras clave.",
                      },
                      {
                        icon: "account_tree",
                        title: "Plan de desarrollo",
                        desc: "Estructurar siguientes pasos y tareas pendientes",
                        prompt: "Genera un plan de desarrollo detallado con las siguientes funciones a implementar.",
                      },
                      {
                        icon: "terminal",
                        title: "Verificar build y tests",
                        desc: "Compilar y correr la suite de pruebas",
                        prompt: "Verifica si el proyecto compila correctamente y ejecuta los tests unitarios.",
                      },
                      {
                        icon: "difference",
                        title: "Revisar cambios de Git",
                        desc: "Examinar diffs pendientes y sugerir commit",
                        prompt: "Revisa los cambios pendientes en el repositorio de Git y resume los diffs.",
                      },
                    ].map((card, i) => (
                      <button
                        key={i}
                        onClick={() => handleSend(card.prompt)}
                        className="flex flex-col text-left p-3.5 rounded-xl bg-white/[0.03] hover:bg-white/[0.07] border border-white/5 hover:border-[#d1f107]/30 transition-all group"
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <span className="material-symbols-outlined text-[16px] text-zinc-400 group-hover:text-[#d1f107] transition-colors">
                            {card.icon}
                          </span>
                          <span className="font-semibold text-white text-[12px] group-hover:text-[#d1f107] transition-colors">
                            {card.title}
                          </span>
                        </div>
                        <span className="text-zinc-400 text-[11px] leading-snug">
                          {card.desc}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Chat Messages */}
              {activeChat?.messages.map((msg) => (
                <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
              ))}

              {/* Agent Active Thinking & Tool Activity Indicator */}
              {isResponding && (
                <div className="flex items-center gap-3 p-3 rounded-2xl bg-white/[0.03] border border-white/10 max-w-md font-sans">
                  <div className="w-8 h-8 rounded-full overflow-hidden flex-shrink-0 bg-black/40 flex items-center justify-center">
                    <img src="/ozybaselogo.png" alt="Ozy" className="w-5 h-5 object-contain" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 text-[12px] text-white/90 font-medium">
                      <span className="w-2 h-2 rounded-full bg-[#d1f107] animate-pulse" />
                      <span>
                        {agentState === "thinking" && "Ozy está analizando el contexto y planeando..."}
                        {agentState === "executing" && "Ozy está ejecutando herramientas en el proyecto..."}
                        {agentState === "awaiting" && "Ozy requiere tu confirmación..."}
                        {agentState === "idle" && "Ozy está procesando código..."}
                      </span>
                    </div>
                    <div className="flex gap-1 mt-1">
                      <span className="w-1.5 h-1.5 rounded-full bg-white/40 animate-bounce" style={{ animationDelay: "0ms" }} />
                      <span className="w-1.5 h-1.5 rounded-full bg-white/40 animate-bounce" style={{ animationDelay: "150ms" }} />
                      <span className="w-1.5 h-1.5 rounded-full bg-white/40 animate-bounce" style={{ animationDelay: "300ms" }} />
                    </div>
                  </div>
                </div>
              )}

              <div ref={bottomRef} />
            </div>
          </div>

          {/* Bottom Terminal Drawer (VSCode style) */}
          <BottomTerminalDrawer
            isOpen={isTerminalOpen}
            onToggle={() => setIsTerminalOpen((v) => !v)}
          />

          {/* Code Input */}
          <div className="shrink-0">
            <CodeInput
              onSend={handleSend}
              bypassPermissions={bypassPermissions}
              onToggleBypassPermissions={setBypassPermissions}
              onTogglePlan={() => handleSend("Generar un plan detallado de tareas para este proyecto")}
            />
          </div>
        </main>

        {/* Right Workbench Context Panel */}
        <ProjectContext />
      </div>
    );
  }

  return (
    <ProjectList
      onNewProject={() => setShowAnalyzer(true)}
      onSelectProject={(projectId) => {
        setSelectedProjectId(projectId);
        setActiveProject(projectId);
        setView("chats");
      }}
    />
  );
}
