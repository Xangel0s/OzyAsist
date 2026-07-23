import { useState } from "react";
import { useChatStore } from "../../store/chatStore";
import ChatMessage from "./ChatMessage";
import ChatInput from "../Home/ChatInput";
import { useScrollToBottom } from "../../hooks";

export default function ChatPage() {
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const createChat = useChatStore((s) => s.createChat);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const updateChatProvider = useChatStore((s) => s.updateChatProvider);

  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState("Todo");
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);

  const activeChat = chats.find((c) => c.id === activeChatId);
  const bottomRef = useScrollToBottom([
    activeChat?.messages.length,
    isResponding,
  ]);

  const handleSend = async (message: string) => {
    let chatId = activeChatId;
    if (!chatId) {
      const newId = await createChat("chat");
      if (!newId) return;
      chatId = newId;
    }
    sendMessage(chatId, message);
  };

  const handleModelChange = (modelId: string, provider: string) => {
    if (activeChatId) {
      updateChatProvider(activeChatId, provider, modelId);
    } else {
      useChatStore.setState({ defaultModel: modelId, defaultProvider: provider });
    }
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

    const matchesFilter =
      filterType === "Todo" ||
      (filterType === "Chat" && c.mode === "chat") ||
      (filterType === "Code" && c.mode === "code") ||
      (filterType === "Cowork" && (c.mode as string) === "cowork");

    return matchesSearch && matchesFilter && (c.messages.length > 0 || c.id === activeChatId);
  });

  // If there is an active chat with messages, show the active chat conversation view
  if (activeChat && activeChat.messages.length > 0) {
    return (
      <div className="flex flex-1 flex-col h-full bg-[#1a1a1a] min-h-0">
        <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0">
          <div className="max-w-[800px] mx-auto flex flex-col gap-6">
            {activeChat.messages.map((msg) => (
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
            <div ref={bottomRef} />
          </div>
        </div>

        <div className="px-4 pb-6 pt-2 bg-gradient-to-t from-[#1a1a1a] via-[#1a1a1a] to-transparent">
          <div className="max-w-[800px] mx-auto">
            <ChatInput
              onSend={handleSend}
              placeholder="Escribe / para habilidades..."
              modelOverride={activeChat?.model}
              onModelChange={handleModelChange}
              compact
            />
            <div className="text-center mt-3 text-[11px] text-white/30">
              Ozy puede cometer errores. Verifica la información importante.
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Dedicated "Chats y tareas" Page View matching screenshots
  return (
    <div className="flex flex-1 flex-col h-full bg-[#1a1a1a] p-8 overflow-y-auto min-h-0 text-white">
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
                <div className="absolute right-0 top-full mt-1.5 w-44 bg-[#262626] border border-white/10 rounded-xl shadow-2xl py-1 z-50 text-[13px] text-white">
                  {["Todo", "Chat", "Code", "Cowork", "Archivado"].map((type) => (
                    <button
                      key={type}
                      className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                      onClick={() => {
                        setFilterType(type);
                        setShowFilterDropdown(false);
                      }}
                    >
                      <span>{type}</span>
                      {filterType === type && (
                        <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <button
              className="px-4 py-1.5 bg-white text-black hover:bg-white/90 text-[13px] font-semibold rounded-lg transition-colors shadow-sm"
              onClick={() => createChat("chat")}
            >
              Nuevo
            </button>
          </div>
        </div>

        {/* Search Bar Input */}
        <div className="relative w-full">
          <span className="material-symbols-outlined absolute left-3.5 top-1/2 -translate-y-1/2 text-[18px] text-white/40">
            search
          </span>
          <input
            type="text"
            placeholder="Buscar chats y tareas..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-[#242424] border border-white/10 rounded-xl pl-10 pr-4 py-2.5 text-[14px] text-white placeholder:text-white/40 focus:outline-none focus:border-white/30 transition-all"
          />
        </div>

        {/* Chats List Table */}
        <div className="flex flex-col border-t border-white/10 pt-2">
          {filteredChats.length === 0 ? (
            <div className="py-12 text-center text-white/40 text-[14px]">
              No se encontraron chats o tareas.
            </div>
          ) : (
            filteredChats.map((chat) => (
              <button
                key={chat.id}
                className="flex items-center justify-between py-3.5 px-3 hover:bg-white/5 rounded-xl transition-all text-left group"
                onClick={() => setActiveChat(chat.id)}
              >
                <div className="flex items-center gap-3 min-w-0 pr-4">
                  <span className="material-symbols-outlined text-[18px] text-white/40 group-hover:text-white/70 transition-colors">
                    chat_bubble
                  </span>
                  <span className="text-[14px] font-medium text-white/90 group-hover:text-white truncate">
                    {chat.title || "Nueva conversación"}
                  </span>
                </div>
                <span className="text-[12px] text-white/40 font-mono whitespace-nowrap">
                  {formatDate(chat.createdAt)}
                </span>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
