import { useState } from "react";
import { useChatStore } from "../../store/chatStore";
import { useProjectsStore } from "../../store/projectsStore";
import { useScrollToBottom } from "../../hooks";
import ChatMessage from "../Chat/ChatMessage";
import CodeInput from "./CodeInput";
import ProjectContext from "./ProjectContext";
import AnalyzeProjectStep from "./AnalyzeProjectStep";
import ProjectList from "./ProjectList";
import ConsentModal from "./ConsentModal";

export default function CodePage() {
  const [showAnalyzer, setShowAnalyzer] = useState(false);
  const consentPending = useChatStore((s) => s.consentPending);
  const resolveConsent = useChatStore((s) => s.resolveConsent);
  const projects = useProjectsStore((s) => s.projects);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const sendMessage = useChatStore((s) => s.sendMessage);
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
          <main className="flex-1 flex flex-col h-full bg-surface-deep relative min-h-0">
            <div className="flex-1 overflow-y-auto px-margin-page py-6 min-h-0 opacity-30 pointer-events-none">
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

  if (projects.length === 0 || showAnalyzer) {
    return <AnalyzeProjectStep onDone={() => setShowAnalyzer(false)} />;
  }

  if (!activeChatId) {
    return <ProjectList onNewProject={() => setShowAnalyzer(true)} />;
  }

  const handleSend = (message: string) => {
    sendMessage(activeChatId, message);
  };

  return (
    <div className="flex flex-1 h-full min-h-0">
      <main className="flex-1 flex flex-col h-full bg-surface-deep relative min-h-0">
        <div className="flex-1 overflow-y-auto px-margin-page py-6 min-h-0">
          <div className="max-w-content-max-width mx-auto flex flex-col gap-6">
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
                    <span className="w-2 h-2 rounded-full bg-text-muted animate-bounce" style={{ animationDelay: "0ms" }} />
                    <span className="w-2 h-2 rounded-full bg-text-muted animate-bounce" style={{ animationDelay: "150ms" }} />
                    <span className="w-2 h-2 rounded-full bg-text-muted animate-bounce" style={{ animationDelay: "300ms" }} />
                  </div>
                </div>
              </div>
            )}
            {(!activeChat || activeChat.messages.length === 0) && !isResponding && (
              <div className="flex-1 flex items-center justify-center text-text-muted text-body-lg">
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
