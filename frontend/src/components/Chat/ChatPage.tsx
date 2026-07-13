import { useState, useRef } from "react";
import { useChatStore } from "../../store/chatStore";
import { useMemoryStore } from "../../store/memoryStore";
import ChatMessage from "./ChatMessage";
import ModelSelector from "../Common/ModelSelector";
import { useOnClickOutside, useScrollToBottom } from "../../hooks";

export default function ChatPage() {
  const [input, setInput] = useState("");
  const [showModelSelector, setShowModelSelector] = useState(false);
  const [showMemoryPopover, setShowMemoryPopover] = useState(false);
  const selectorRef = useRef<HTMLDivElement>(null);
  const memoryRef = useRef<HTMLDivElement>(null);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const coworkMode = useChatStore((s) => s.coworkMode);
  const setCoworkMode = useChatStore((s) => s.setCoworkMode);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const createChat = useChatStore((s) => s.createChat);
  const memoryEntries = useMemoryStore((s) => s.entries);
  const clearMemory = useMemoryStore((s) => s.clearMemory);

  const activeChat = chats.find((c) => c.id === activeChatId);
  const bottomRef = useScrollToBottom([
    activeChat?.messages.length,
    isResponding,
  ]);

  useOnClickOutside(selectorRef, () => setShowModelSelector(false));
  useOnClickOutside(memoryRef, () => setShowMemoryPopover(false));

  const recentMemory = memoryEntries.slice(0, 5);

  const handleSend = () => {
    if (!input.trim()) return;
    let chatId = activeChatId;
    if (!chatId) {
      createChat("chat");
      chatId = useChatStore.getState().activeChatId;
    }
    if (chatId) {
      sendMessage(chatId, input.trim());
      setInput("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex flex-1 flex-col h-full bg-surface-deep min-h-0">
      <div className="flex-1 overflow-y-auto px-margin-page py-6 min-h-0">
        <div className="max-w-content-max-width mx-auto flex flex-col gap-6">
          {(!activeChat || activeChat.messages.length === 0) && (
            <div className="flex-1 flex flex-col items-center justify-center min-h-[400px] gap-6">
              <div className="w-14 h-14 rounded-xl flex items-center justify-center overflow-hidden">
                <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
              </div>
              <h2 className="text-headline-md font-headline-md text-on-surface text-center">
                ¿En qué puedo ayudarte?
              </h2>
              <p className="text-text-muted text-body-md text-center max-w-md">
                Conversación general con selección de modelo y proveedor.
              </p>
            </div>
          )}
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
          <div ref={bottomRef} />
        </div>
      </div>

      <div className="px-margin-page pb-6 pt-2 bg-gradient-to-t from-surface-deep via-surface-deep to-transparent">
        <div className="max-w-content-max-width mx-auto">
          <div className="bg-surface-elevated rounded-xl border border-border-subtle shadow-lg flex flex-col focus-within:border-outline focus-within:ring-1 focus-within:ring-outline transition-all">
            <textarea
              className="w-full bg-transparent border-none text-on-surface font-body-lg text-body-lg p-4 resize-none focus:ring-0 placeholder:text-text-muted min-h-[90px] outline-none"
              placeholder={coworkMode ? "¿Qué tarea querés que realice?" : "Escribe tu mensaje..."}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              rows={3}
            />
            <div className="flex items-center justify-between p-3 border-t border-border-subtle/50">
              <div className="flex items-center gap-2">
                <button className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors flex items-center justify-center">
                  <span className="material-symbols-outlined text-[20px]">add</span>
                </button>
                <div className="flex items-center bg-surface-variant rounded-md overflow-hidden">
                  <button
                    className={`px-3 py-1.5 text-body-md flex items-center gap-1.5 transition-colors ${
                      !coworkMode
                        ? "text-on-surface bg-surface-bright"
                        : "text-text-muted hover:text-on-surface hover:bg-surface-bright"
                    }`}
                    onClick={() => setCoworkMode(false)}
                  >
                    <span className="material-symbols-outlined text-[16px]">chat</span>
                    Chat
                  </button>
                  <button
                    className={`px-3 py-1.5 text-body-md transition-colors ${
                      coworkMode
                        ? "text-on-surface bg-surface-bright"
                        : "text-text-muted hover:text-on-surface hover:bg-surface-bright"
                    }`}
                    onClick={() => setCoworkMode(true)}
                  >
                    Cowork
                  </button>
                </div>

                {recentMemory.length > 0 && (
                  <div className="relative" ref={memoryRef}>
                    <button
                      className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-body-md"
                      onClick={() => setShowMemoryPopover(!showMemoryPopover)}
                      title="Memoria de esta sesión"
                    >
                      <span className="material-symbols-outlined text-[16px] text-primary-container">
                        memory
                      </span>
                      <span className="text-label-caps text-text-muted">{recentMemory.length}</span>
                    </button>
                    {showMemoryPopover && (
                      <div className="absolute bottom-full left-0 mb-2 w-80 bg-surface-elevated border border-border-subtle rounded-xl shadow-xl p-3 z-50">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-label-caps text-text-muted">
                            Memoria reciente
                          </span>
                          <button
                            className="text-text-muted hover:text-on-surface text-label-caps transition-colors"
                            onClick={() => { clearMemory(); setShowMemoryPopover(false); }}
                          >
                            Limpiar
                          </button>
                        </div>
                        <div className="flex flex-col gap-2 max-h-64 overflow-y-auto">
                          {recentMemory.map((entry) => (
                            <div
                              key={entry.id}
                              className="bg-surface-variant rounded-lg p-2.5 text-[13px]"
                            >
                              <div className="text-on-surface font-medium truncate mb-1">
                                {entry.topic}
                              </div>
                              <div className="text-text-muted line-clamp-3 whitespace-pre-wrap">
                                {entry.content.slice(0, 200)}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-3 relative" ref={selectorRef}>
                <button
                  className="flex items-center gap-1 text-text-muted hover:text-on-surface transition-colors text-body-md group"
                  onClick={() => setShowModelSelector(!showModelSelector)}
                >
                  {activeChat?.model || "DeepSeek V4 Pro"}
                  <span className="text-text-muted/70 group-hover:text-text-muted transition-colors ml-1">
                    {activeChat?.provider === "openai" ? "Alto" : "Medio"}
                  </span>
                  <span className="material-symbols-outlined text-[16px] ml-0.5">expand_more</span>
                </button>
                {showModelSelector && (
                  <ModelSelector onClose={() => setShowModelSelector(false)} />
                )}
                <div className="w-px h-4 bg-border-subtle mx-1" />
                <button className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors flex items-center justify-center">
                  <span className="material-symbols-outlined text-[20px]">mic</span>
                </button>
                <button
                  className="p-1.5 rounded-lg bg-primary-container text-on-primary-fixed hover:bg-primary-fixed transition-colors flex items-center justify-center ml-1"
                  onClick={handleSend}
                >
                  <span className="material-symbols-outlined text-[20px] fill">arrow_upward</span>
                </button>
              </div>
            </div>
          </div>
          <div className="text-center mt-3 text-label-caps text-text-muted opacity-60">
            Ozy puede cometer errores. Verifica la información importante.
          </div>
        </div>
      </div>
    </div>
  );
}
