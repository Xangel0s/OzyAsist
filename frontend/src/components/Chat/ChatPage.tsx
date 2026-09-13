import { useState } from "react";
import { useChatStore } from "../../store/chatStore";
import ChatMessage from "./ChatMessage";
import { OzyChatConsole } from "./OzyChatConsole";
import { OzyLogo } from "../Brand/OzyLogo";
import { useScrollToBottom } from "../../hooks";

export default function ChatPage() {
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const createChat = useChatStore((s) => s.createChat);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const stopStreaming = useChatStore((s) => s.stopStreaming);

  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState("Todo");
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);

  const activeChat = chats.find((c) => c.id === activeChatId);
  const bottomRef = useScrollToBottom([
    activeChat?.messages.length,
    isResponding,
  ]);

  const handleSend = async (message: string, mode: "chat" | "cowork" = "chat", model?: string) => {
    let chatId = activeChatId;
    if (!chatId) {
      const newId = await createChat(mode === "cowork" ? "code" : "chat", undefined, undefined, model);
      if (!newId) return;
      chatId = newId;
    }
    sendMessage(chatId, message);
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return "13 jul";
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString("es-ES", { day: "numeric", month: "short" });
    } catch {
      return "13 jul";
    }
  };

  const filteredChats = chats.filter((c) => {
    const matchesSearch =
      !searchQuery ||
      c.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      c.messages.some((m) => m.content.toLowerCase().includes(searchQuery.toLowerCase()));

    return matchesSearch && (c.messages.length > 0 || c.id === activeChatId);
  });

  // If there is an active chat with messages, show the active chat conversation view
  if (activeChat && activeChat.messages.length > 0) {
    return (
      <div className="flex flex-1 flex-col h-full bg-[#131313] min-h-0">
        <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0">
          <div className="max-w-3xl mx-auto flex flex-col gap-6">
            {activeChat.messages.map((msg) => (
              <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
            ))}
            {isResponding && (
              <div className="flex justify-start my-2">
                <div className="flex gap-3.5 max-w-[90%] items-center">
                  <OzyLogo size={24} isThinking />
                  <div className="flex gap-1.5 py-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#d1f107] animate-bounce" style={{ animationDelay: "0ms" }} />
                    <span className="w-1.5 h-1.5 rounded-full bg-[#d1f107] animate-bounce" style={{ animationDelay: "150ms" }} />
                    <span className="w-1.5 h-1.5 rounded-full bg-[#d1f107] animate-bounce" style={{ animationDelay: "300ms" }} />
                  </div>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
        </div>

        <div className="px-4 pb-6 pt-2 bg-gradient-to-t from-[#131313] via-[#131313] to-transparent">
          <div className="max-w-3xl mx-auto">
            <OzyChatConsole
              onSendMessage={handleSend}
              onStopStreaming={stopStreaming}
              isStreaming={isResponding}
            />
            <div className="text-center mt-2 text-[11px] text-neutral-500 font-mono">
              OzyAssist es un agente autónomo de sistema operativo. Verifica acciones críticas.
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Dedicated "Chats y tareas" Page View matching screenshots
  return (
    <div className="flex flex-1 flex-col h-full bg-[#131313] p-8 overflow-y-auto min-h-0 text-white">
      <div className="max-w-[850px] mx-auto w-full flex flex-col gap-6">
        {/* Header Title & Actions Bar */}
        <div className="flex items-center justify-between">
          <h1 className="text-[26px] font-bold tracking-tight text-white font-sans">Chats y tareas</h1>

          <div className="flex items-center gap-2">
            <button className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors">
              Seleccionar
            </button>

            {/* Filter Dropdown: Todo, Chat, Code, Cowork, Archivado */}
            <div className="relative">
              <button
                className="flex items-center gap-1.5 px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors"
                onClick={() => setShowFilterDropdown(!showFilterDropdown)}
              >
                <span className="text-white/60">Filtrar por</span>
                <span className="font-semibold">{filterType}</span>
                <span className="material-symbols-outlined text-[16px] text-white/60">expand_more</span>
              </button>

              {showFilterDropdown && (
                <div className="absolute right-0 mt-2 w-40 bg-[#222222] border border-white/10 rounded-xl shadow-xl z-20 py-1 overflow-hidden">
                  {["Todo", "Chat", "Code", "Cowork", "Archivado"].map((f) => (
                    <button
                      key={f}
                      className={`w-full text-left px-3 py-2 text-xs transition-colors hover:bg-white/10 ${
                        filterType === f ? "text-[#d1f107] font-semibold bg-white/5" : "text-white/80"
                      }`}
                      onClick={() => {
                        setFilterType(f);
                        setShowFilterDropdown(false);
                      }}
                    >
                      {f}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Search Bar */}
        <div className="relative w-full">
          <span className="material-symbols-outlined absolute left-3.5 top-1/2 -translate-y-1/2 text-white/40 text-[18px]">
            search
          </span>
          <input
            type="text"
            placeholder="Buscar en tus chats..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-white/5 border border-white/10 rounded-xl text-sm text-white placeholder-white/40 focus:outline-none focus:border-[#d1f107]/50 transition-colors"
          />
        </div>

        {/* List of Chats */}
        <div className="flex flex-col gap-2">
          {filteredChats.length === 0 ? (
            <div className="text-center py-12 text-white/40 text-sm">
              No se encontraron chats o tareas.
            </div>
          ) : (
            filteredChats.map((c) => (
              <div
                key={c.id}
                onClick={() => setActiveChat(c.id)}
                className={`p-4 rounded-xl border transition-all cursor-pointer flex items-center justify-between ${
                  activeChatId === c.id
                    ? "bg-white/10 border-[#d1f107]/30 shadow-sm"
                    : "bg-white/[0.02] border-white/5 hover:bg-white/[0.06] hover:border-white/10"
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-white/5 flex items-center justify-center text-white/60">
                    <span className="material-symbols-outlined text-[18px]">
                      {c.mode === "code" ? "terminal" : "chat_bubble"}
                    </span>
                  </div>
                  <div>
                    <h3 className="text-sm font-medium text-white line-clamp-1">{c.title || "Nuevo chat"}</h3>
                    <p className="text-xs text-white/40 line-clamp-1">
                      {c.messages.length > 0
                        ? c.messages[c.messages.length - 1].content
                        : "Sin mensajes aún"}
                    </p>
                  </div>
                </div>
                <div className="text-xs text-white/40 font-mono">
                  {formatDate(c.createdAt)}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
