import { useState } from "react";
import { useProjectsStore } from "../../store/projectsStore";
import { useChatStore } from "../../store/chatStore";

interface ProjectChatsProps {
  projectId: string;
  onBack: () => void;
  onSelectChat: (chatId: string) => void;
}

export default function ProjectChats({ projectId, onBack, onSelectChat }: ProjectChatsProps) {
  const projects = useProjectsStore((s) => s.projects);
  const chats = useChatStore((s) => s.chats);
  const createChat = useChatStore((s) => s.createChat);
  const deleteChat = useChatStore((s) => s.deleteChat);
  const [search, setSearch] = useState("");

  const project = projects.find((p) => p.id === projectId);
  const projectChats = chats.filter((c) => c.projectId === projectId && c.mode === "code");
  const filtered = projectChats.filter((c) =>
    (c.title || "Nuevo chat").toLowerCase().includes(search.toLowerCase())
  );

  const handleNewChat = async () => {
    const chatId = await createChat("code", projectId);
    if (chatId) {
      onSelectChat(chatId);
    }
  };

  const handleDeleteChat = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    await deleteChat(id);
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-[#1a1a1a] overflow-y-auto">
      <div className="max-w-[800px] mx-auto w-full px-6 py-12">
        <button
          className="flex items-center gap-2 text-white/40 hover:text-white transition-colors mb-6 text-[14px]"
          onClick={onBack}
        >
          <span className="material-symbols-outlined text-[18px]">arrow_back</span>
          Volver a proyectos
        </button>

        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-[20px] font-semibold text-white">
              {project?.name || "Proyecto"}
            </h1>
            <p className="text-[13px] text-white/40 mt-1">
              {project?.rootPath || "Sin ruta definida"}
            </p>
          </div>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-colors text-[13px] font-semibold"
            onClick={handleNewChat}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nuevo chat
          </button>
        </div>

        <div className="relative mb-6">
          <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-white/30 text-[20px]">
            search
          </span>
          <input
            className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl py-2.5 pl-10 pr-4 text-white placeholder:text-white/25 outline-none focus:border-[#c8e64a]/50 transition-colors text-[14px]"
            placeholder="Buscar chats..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-2">
          {filtered.map((chat) => (
            <button
              key={chat.id}
              className="flex items-center gap-4 bg-[#1e1e1e] rounded-xl border border-white/10 p-4 text-left hover:bg-[#252525] hover:border-white/15 transition-all group"
              onClick={() => onSelectChat(chat.id)}
            >
              <div className="w-10 h-10 rounded-lg bg-[#c8e64a]/15 flex items-center justify-center shrink-0">
                <span className="material-symbols-outlined text-[#c8e64a] text-[20px]">chat</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-[14px] font-medium text-white truncate">
                  {chat.title || "Nuevo chat"}
                </div>
                <div className="text-[12px] text-white/40 truncate">
                  {chat.model || "Sin modelo"} · {new Date(chat.createdAt).toLocaleDateString()}
                </div>
              </div>
              <button
                className="p-1.5 rounded-lg text-white/20 hover:text-red-400 hover:bg-red-400/10 opacity-0 group-hover:opacity-100 transition-all"
                onClick={(e) => handleDeleteChat(chat.id, e)}
                title="Eliminar chat"
              >
                <span className="material-symbols-outlined text-[16px]">delete</span>
              </button>
            </button>
          ))}

          {filtered.length === 0 && (
            <div className="flex flex-col items-center justify-center py-16 text-white/30">
              <span className="material-symbols-outlined text-[48px] mb-4">chat_bubble_outline</span>
              <p className="text-[15px]">
                {projectChats.length === 0
                  ? "No hay chats aún. Iniciá uno nuevo."
                  : "Sin resultados para esta búsqueda"}
              </p>
              {projectChats.length === 0 && (
                <button
                  className="mt-4 px-4 py-2 bg-[#c8e64a] text-[#1a1a1a] rounded-xl hover:bg-[#b8d63a] transition-colors text-[13px] font-semibold"
                  onClick={handleNewChat}
                >
                  Nuevo chat
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
