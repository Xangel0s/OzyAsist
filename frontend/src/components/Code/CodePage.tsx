import { useState } from "react";
import { useChatStore } from "../../store/chatStore";
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
  const [view, setView] = useState<CodeView>("projects");
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const consentPending = useChatStore((s) => s.consentPending);
  const resolveConsent = useChatStore((s) => s.resolveConsent);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const activeChat = chats.find((c) => c.id === activeChatId);
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
      setView("chats");
    }} />;
  }

  if (view === "projects") {
    return (
      <ProjectList
        onNewProject={() => setShowAnalyzer(true)}
        onSelectProject={(projectId) => {
          setSelectedProjectId(projectId);
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
      <div className="flex flex-1 h-full min-h-0">
        <main className="flex-1 flex flex-col h-full bg-[#1a1a1a] relative min-h-0">
          <div className="flex items-center gap-3 px-4 py-3 border-b border-white/10">
            <button
              className="w-8 h-8 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
              onClick={() => {
                setView("chats");
              }}
            >
              <span className="material-symbols-outlined text-[20px]">arrow_back</span>
            </button>
            <div className="flex-1 min-w-0">
              <div className="text-[14px] font-medium text-white truncate">
                {activeChat?.title || "Chat de código"}
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0">
            <div className="max-w-[800px] mx-auto flex flex-col gap-6">
              {activeChat?.messages.map((msg) => (
                <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
              ))}
              {isResponding && (
                <div className="flex justify-start">
                  <div className="flex gap-4 max-w-[90%]">
                    <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 overflow-hidden">
                      <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
                    </div>
                    <div className="flex gap-1.5 py-3">
                      <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "0ms" }} />
                      <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "150ms" }} />
                      <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "300ms" }} />
                    </div>
                  </div>
                </div>
              )}
              {(!activeChat || activeChat.messages.length === 0) && !isResponding && (
                <div className="flex-1 flex items-center justify-center text-white/30 text-[15px]">
                  Envía un mensaje para comenzar el análisis de código
                </div>
              )}
              <div ref={bottomRef} />
            </div>
          </div>
          <CodeInput onSend={handleSend} />
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
        setView("chats");
      }}
    />
  );
}
