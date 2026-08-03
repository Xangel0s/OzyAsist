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
    return <AnalyzeProjectStep onDone={(projectId) => {
      setShowAnalyzer(false);
      setSelectedProjectId(projectId);
      setActiveProject(projectId);
      setView("chats");
    }} />;
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

  const [bypassPermissions, setBypassPermissions] = useState(true);
  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === selectedProjectId);

  if (view === "chat" && activeChatId) {
    const handleSend = (message: string) => {
      sendMessage(activeChatId, message);
    };

    return (
      <div className="flex flex-1 h-full min-h-0">
        <main className="flex-1 flex flex-col h-full bg-[#181818] relative min-h-0">
          {/* Claude Code Header Bar */}
          <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/10 bg-[#141414]">
            <div className="flex items-center gap-3 min-w-0">
              <button
                className="w-7 h-7 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors shrink-0"
                onClick={() => setView("chats")}
                title="Volver a los chats del proyecto"
              >
                <span className="material-symbols-outlined text-[18px]">arrow_back</span>
              </button>
              
              <div className="flex items-center gap-1.5 bg-white/5 border border-white/10 px-2.5 py-1 rounded-lg text-[13px] font-medium text-white cursor-pointer hover:bg-white/10 transition-colors">
                <span className="truncate">{activeProject?.name || "fishapp"}</span>
                <span className="material-symbols-outlined text-[16px] text-white/40">expand_more</span>
              </div>
            </div>

            <button
              onClick={() => {
                const el = document.getElementById("toggle-project-context-preview");
                el?.click();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 text-white/80 hover:text-white hover:bg-white/10 text-[12px] font-medium transition-colors"
            >
              <span className="material-symbols-outlined text-[16px] text-[#d1f107]">play_arrow</span>
              Vista previa
            </button>
          </div>

          {/* Amber Permission Banner */}
          {bypassPermissions && (
            <div className="bg-amber-950/80 border-b border-amber-500/30 px-4 py-2 text-[12px] text-amber-200/90 flex items-center justify-between shadow-sm">
              <div className="flex items-center gap-2 truncate">
                <span className="font-semibold text-amber-400">Modo de omisión de permisos:</span>
                <span className="truncate">Ozy puede realizar acciones sin preguntar, incluyendo modificar o eliminar archivos.</span>
              </div>
              <button
                onClick={() => setBypassPermissions(false)}
                className="underline text-amber-300 hover:text-white text-[11px] shrink-0 ml-2"
              >
                Desactivar
              </button>
            </div>
          )}

          {/* Chat Feed */}
          <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0">
            <div className="max-w-[850px] mx-auto flex flex-col gap-6">
              {activeChat?.messages.map((msg) => (
                <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
              ))}

              {isResponding && (
                <div className="flex flex-col gap-3 font-mono text-[13px]">
                  {/* Edit Chip */}
                  <div className="flex items-center gap-2 text-white/60 bg-white/5 border border-white/10 px-3 py-1.5 rounded-lg w-fit">
                    <span className="text-white/40">Edit</span>
                    <span className="text-white/90">src/components/CodeView.tsx</span>
                    <span className="text-emerald-400 font-semibold">+8</span>
                    <span className="text-rose-400 font-semibold">-12</span>
                  </div>

                  {/* Claude Code Update Todos Checklist Card */}
                  <div className="bg-[#202020] border border-white/10 rounded-xl p-4 flex flex-col gap-2 shadow-lg">
                    <div className="text-[12px] font-semibold text-white/80 uppercase tracking-wider mb-1">
                      Update Todos
                    </div>
                    <div className="flex flex-col gap-1.5 text-white/70 text-[13px]">
                      <div className="flex items-center gap-2 line-through text-white/40">
                        <span className="material-symbols-outlined text-[16px] text-emerald-400">check_box</span>
                        Instalar dependencias clave de desarrollo
                      </div>
                      <div className="flex items-center gap-2 line-through text-white/40">
                        <span className="material-symbols-outlined text-[16px] text-emerald-400">check_box</span>
                        Crear utilidad de inspección AST
                      </div>
                      <div className="flex items-center gap-2 text-white font-medium">
                        <span className="material-symbols-outlined text-[16px] text-amber-400 animate-pulse">check_box_outline_blank</span>
                        Modificar componente de vista previa
                      </div>
                      <div className="flex items-center gap-2 text-white/40">
                        <span className="material-symbols-outlined text-[16px]">check_box_outline_blank</span>
                        Verificar build de producción
                      </div>
                    </div>
                  </div>

                  {/* Status Indicator */}
                  <div className="flex items-center justify-between text-[11px] text-white/40 pt-1">
                    <div className="flex items-center gap-2 text-amber-400">
                      <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-ping" />
                      <span>· Procesando...</span>
                    </div>
                    <div>3m 44s · 3.7k tokens</div>
                  </div>
                </div>
              )}

              {(!activeChat || activeChat.messages.length === 0) && !isResponding && (
                <div className="flex-1 flex items-center justify-center text-white/30 text-[14px]">
                  Escribí una instrucción para comenzar a programar con Ozy
                </div>
              )}
              <div ref={bottomRef} />
            </div>
          </div>

          <CodeInput
            onSend={handleSend}
            bypassPermissions={bypassPermissions}
            onToggleBypassPermissions={setBypassPermissions}
          />
        </main>
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
