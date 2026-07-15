import { useState, useEffect, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useProjectsStore } from "../../store/projectsStore";
import { useKeyboard } from "../../hooks";
import { api } from "../../services/api";

interface SearchHit {
  id: string;
  snippet?: string;
  title?: string;
  name?: string;
  text?: string;
  kind?: string;
}

export default function SearchModal() {
  const searchOpen = useUIStore((s) => s.searchOpen);
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const setActiveChat = useChatStore((s) => s.setActiveChat);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const chats = useChatStore((s) => s.chats);
  const projects = useProjectsStore((s) => s.projects);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<{
    messages: SearchHit[];
    chats: SearchHit[];
    projects: SearchHit[];
    memory: SearchHit[];
  } | null>(null);
  const [searching, setSearching] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<number | null>(null);

  useKeyboard("Escape", () => setSearchOpen(false));

  useEffect(() => {
    if (searchOpen) {
      setTimeout(() => inputRef.current?.focus(), 100);
    } else {
      setQuery("");
      setResults(null);
    }
  }, [searchOpen]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!query.trim()) {
      setResults(null);
      return;
    }
    debounceRef.current = window.setTimeout(async () => {
      setSearching(true);
      try {
        const data = await api.search.global(query);
        setResults(data);
      } catch {
        setResults(null);
      }
      setSearching(false);
    }, 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [query]);

  if (!searchOpen) return null;

  const filteredChats = chats.filter((c) =>
    c.title.toLowerCase().includes(query.toLowerCase()),
  );
  const filteredProjects = projects.filter((p) =>
    p.name.toLowerCase().includes(query.toLowerCase()),
  );

  const handleSelectChat = (chatId: string, mode: string) => {
    setActiveChat(chatId);
    setActiveView(mode === "code" ? "code" : "chat");
    setSearchOpen(false);
  };

  return (
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] bg-black/60"
      onClick={() => setSearchOpen(false)}
    >
      <div
        className="w-full max-w-xl mx-4 bg-surface-elevated border border-border-subtle rounded-2xl overflow-hidden shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 p-4 border-b border-border-subtle">
          <span className="material-symbols-outlined text-text-muted">
            search
          </span>
          <input
            ref={inputRef}
            className="flex-1 bg-transparent border-none text-on-surface placeholder:text-text-muted outline-none text-body-lg"
            placeholder="Buscar chats, proyectos, archivos..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <kbd className="px-2 py-0.5 bg-surface-variant rounded text-label-caps text-text-muted">
            ESC
          </kbd>
        </div>
        <div className="max-h-80 overflow-y-auto p-2">
          {searching && (
            <div className="px-3 py-8 text-center text-text-muted text-body-md">
              Buscando...
            </div>
          )}
          {!searching && results && (
            <>
              {results.messages.length > 0 && (
                <div className="mb-2">
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Mensajes</div>
                  {results.messages.map((m) => (
                    <div key={m.id} className="flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted">
                      <span className="material-symbols-outlined text-[16px]">chat_bubble</span>
                      <span className="truncate">{m.snippet}</span>
                    </div>
                  ))}
                </div>
              )}
              {results.chats.length > 0 && (
                <div className="mb-2">
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Chats</div>
                  {results.chats.map((ch) => (
                    <button
                      key={ch.id}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
                      onClick={() => handleSelectChat(ch.id, "chat")}
                    >
                      <span className="material-symbols-outlined text-[16px]">forum</span>
                      <span className="truncate">{ch.title}</span>
                    </button>
                  ))}
                </div>
              )}
              {results.projects.length > 0 && (
                <div className="mb-2">
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Proyectos</div>
                  {results.projects.map((p) => (
                    <button
                      key={p.id}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
                      onClick={() => { setActiveProject(p.id); setActiveView("projects"); setSearchOpen(false); }}
                    >
                      <span className="material-symbols-outlined text-[16px]">inventory_2</span>
                      <span className="truncate">{p.name}</span>
                    </button>
                  ))}
                </div>
              )}
              {results.memory.length > 0 && (
                <div className="mb-2">
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Memoria</div>
                  {results.memory.map((m) => (
                    <div key={m.id} className="flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted">
                      <span className="material-symbols-outlined text-[16px]">psychiatry</span>
                      <span className="truncate">{m.text}</span>
                    </div>
                  ))}
                </div>
              )}
              {results.messages.length === 0 && results.chats.length === 0 && results.projects.length === 0 && results.memory.length === 0 && (
                <div className="px-3 py-8 text-center text-text-muted text-body-md">
                  Sin resultados para "{query}"
                </div>
              )}
            </>
          )}
          {!searching && !results && query && (
            <>
              {filteredChats.length > 0 && (
                <div className="mb-2">
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Chats</div>
                  {filteredChats.map((chat) => (
                    <button
                      key={chat.id}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
                      onClick={() => handleSelectChat(chat.id, chat.mode)}
                    >
                      <span className="material-symbols-outlined text-[16px]">chat_bubble</span>
                      <span className="truncate">{chat.title}</span>
                    </button>
                  ))}
                </div>
              )}
              {filteredProjects.length > 0 && (
                <div>
                  <div className="px-3 py-1.5 text-label-caps text-text-muted">Proyectos</div>
                  {filteredProjects.map((project) => (
                    <button
                      key={project.id}
                      className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
                      onClick={() => {
                        setActiveProject(project.id);
                        setActiveView("projects");
                        setSearchOpen(false);
                      }}
                    >
                      <span className="material-symbols-outlined text-[16px]">inventory_2</span>
                      <span className="truncate">{project.name}</span>
                    </button>
                  ))}
                </div>
              )}
              {filteredChats.length === 0 && filteredProjects.length === 0 && (
                <div className="px-3 py-8 text-center text-text-muted text-body-md">
                  Sin resultados para "{query}"
                </div>
              )}
            </>
          )}
          {!query && (
            <div className="px-3 py-8 text-center text-text-muted text-body-md">
              Escribe para buscar en toda la aplicación
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
