import { useState } from "react";
import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import { useToastStore } from "../../store/toastStore";

export default function ChatsTasksPage() {
  const chats = useChatStore((s) => s.chats);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const createChat = useChatStore((s) => s.createChat);
  const deleteChat = useChatStore((s) => s.deleteChat);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const toast = useToastStore((s) => s.show);

  const [searchQuery, setSearchQuery] = useState("");
  const [filterMode, setFilterMode] = useState<"all" | "chat" | "code">("all");
  const [isSelectMode, setIsSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // Filter chats by search query and mode
  const filteredChats = chats.filter((c) => {
    const matchesSearch = c.title.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesMode =
      filterMode === "all" ? true : filterMode === "chat" ? c.mode === "chat" : c.mode === "code";
    return matchesSearch && matchesMode;
  });

  const handleOpenChat = (chatId: string, mode: "chat" | "code") => {
    if (isSelectMode) {
      toggleSelect(chatId);
      return;
    }
    setActiveChat(chatId);
    setActiveView(mode === "code" ? "code" : "chat");
  };

  const handleCreateNew = async () => {
    const chatId = await createChat("chat");
    if (chatId) {
      setActiveView("chat");
    }
  };

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    );
  };

  const toggleSelectAll = () => {
    if (selectedIds.length === filteredChats.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filteredChats.map((c) => c.id));
    }
  };

  const handleDeleteSingle = async (e: React.MouseEvent, chatId: string) => {
    e.stopPropagation();
    await deleteChat(chatId);
    toast("Chat eliminado", "success");
  };

  const handleDeleteSelected = async () => {
    if (selectedIds.length === 0) return;
    const count = selectedIds.length;
    const idsToDelete = [...selectedIds];
    setSelectedIds([]);
    setIsSelectMode(false);
    for (const id of idsToDelete) {
      await deleteChat(id);
    }
    toast(`${count} ${count === 1 ? "chat eliminado" : "chats eliminados"}`, "success");
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return "";
    try {
      const d = new Date(dateStr);
      const months = ["ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"];
      return `${d.getDate()} ${months[d.getMonth()]}`;
    } catch {
      return "";
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-background overflow-hidden p-8 max-w-5xl mx-auto w-full">
      {/* Header section */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-on-surface">Chats y tareas</h1>
        <div className="flex items-center gap-3">
          <button
            className={`px-4 py-1.5 rounded-lg text-[13px] font-medium transition-all ${
              isSelectMode
                ? "bg-surface-bright text-on-surface border border-border-subtle"
                : "bg-surface-elevated text-text-muted hover:text-on-surface border border-border-subtle"
            }`}
            onClick={() => {
              setIsSelectMode(!isSelectMode);
              setSelectedIds([]);
            }}
          >
            {isSelectMode ? "Cancelar" : "Seleccionar"}
          </button>

          <div className="relative flex items-center">
            <span className="text-[13px] text-text-muted mr-2">Filtrar por</span>
            <select
              className="bg-surface-elevated text-on-surface border border-border-subtle rounded-lg px-3 py-1.5 text-[13px] font-medium outline-none cursor-pointer"
              value={filterMode}
              onChange={(e) => setFilterMode(e.target.value as "all" | "chat" | "code")}
            >
              <option value="all">Todo</option>
              <option value="chat">Chats</option>
              <option value="code">Code</option>
            </select>
          </div>

          <button
            className="px-4 py-1.5 bg-on-surface text-background hover:opacity-90 font-medium text-[13px] rounded-lg transition-all shadow-sm flex items-center gap-1.5"
            onClick={handleCreateNew}
          >
            <span className="material-symbols-outlined text-[16px]">add</span>
            <span>Nuevo</span>
          </button>
        </div>
      </div>

      {/* Search Input */}
      <div className="relative mb-6">
        <span className="material-symbols-outlined text-[18px] text-text-muted absolute left-3.5 top-1/2 -translate-y-1/2">
          search
        </span>
        <input
          type="text"
          className="w-full bg-surface-container-low border border-border-subtle rounded-xl pl-10 pr-4 py-2.5 text-[14px] text-on-surface placeholder:text-text-muted/50 outline-none focus:border-border-subtle/80 transition-all"
          placeholder="Buscar chats y tareas..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {/* Selection Action Bar */}
      {isSelectMode && (
        <div className="flex items-center justify-between bg-surface-elevated border border-border-subtle p-3 rounded-xl mb-4 animate-fadeIn">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              className="w-4 h-4 accent-[#d1f107] cursor-pointer"
              checked={filteredChats.length > 0 && selectedIds.length === filteredChats.length}
              onChange={toggleSelectAll}
            />
            <span className="text-[13px] text-on-surface font-medium">
              {selectedIds.length} seleccionados
            </span>
          </div>
          <button
            className="px-3 py-1.5 bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20 rounded-lg text-[13px] font-medium transition-all flex items-center gap-1.5 disabled:opacity-40"
            disabled={selectedIds.length === 0}
            onClick={handleDeleteSelected}
          >
            <span className="material-symbols-outlined text-[16px]">delete</span>
            <span>Eliminar seleccionados</span>
          </button>
        </div>
      )}

      {/* List view */}
      <div className="flex-1 overflow-y-auto custom-scrollbar pr-1 flex flex-col gap-1 border-t border-border-subtle/40 pt-4">
        {filteredChats.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-text-muted">
            <span className="material-symbols-outlined text-[36px] opacity-40 mb-2">chat_bubble_outline</span>
            <p className="text-[14px]">No se encontraron chats o tareas</p>
          </div>
        ) : (
          filteredChats.map((chat) => {
            const isSelected = selectedIds.includes(chat.id);
            return (
              <div
                key={chat.id}
                className={`group flex items-center justify-between px-4 py-3 rounded-xl transition-all cursor-pointer ${
                  isSelected
                    ? "bg-surface-variant border border-[#d1f107]/40"
                    : "hover:bg-surface-container-high/60 border border-transparent"
                }`}
                onClick={() => handleOpenChat(chat.id, chat.mode)}
              >
                <div className="flex items-center gap-3.5 min-w-0 flex-1">
                  {isSelectMode && (
                    <input
                      type="checkbox"
                      className="w-4 h-4 accent-[#d1f107] cursor-pointer shrink-0"
                      checked={isSelected}
                      onClick={(e) => e.stopPropagation()}
                      onChange={(e) => {
                        e.stopPropagation();
                        toggleSelect(chat.id);
                      }}
                    />
                  )}
                  <span className="material-symbols-outlined text-[20px] text-text-muted shrink-0 group-hover:text-on-surface transition-colors">
                    {chat.mode === "code" ? "code" : "chat_bubble_outline"}
                  </span>
                  <span className="text-[14px] text-on-surface font-medium truncate">
                    {chat.title}
                  </span>
                </div>

                <div className="flex items-center gap-4 shrink-0 ml-4">
                  {chat.createdAt && (
                    <span className="text-[12px] text-text-muted font-mono">
                      {formatDate(chat.createdAt)}
                    </span>
                  )}
                  {!isSelectMode && (
                    <button
                      className="opacity-0 group-hover:opacity-100 p-1.5 hover:bg-surface-elevated rounded-lg text-text-muted hover:text-red-400 transition-all"
                      title="Eliminar chat"
                      onClick={(e) => handleDeleteSingle(e, chat.id)}
                    >
                      <span className="material-symbols-outlined text-[16px]">delete</span>
                    </button>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
