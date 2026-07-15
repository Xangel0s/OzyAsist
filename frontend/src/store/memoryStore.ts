import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "../services/api";

export interface MemoryEntry {
  id: string;
  topic: string;
  content: string;
  source: "import" | "chat" | "manual";
  sourceChatId?: string;
  createdAt: string;
}

interface MemoryState {
  entries: MemoryEntry[];
  rawMd: string | null;
  addEntry: (entry: Omit<MemoryEntry, "id" | "createdAt">) => void;
  removeEntry: (id: string) => void;
  importFromMd: (md: string) => Promise<{ chunks: number; feedback: string }>;
  clearMemory: () => void;
}

function parseMdSections(md: string): { topic: string; content: string }[] {
  const sections: { topic: string; content: string }[] = [];
  const lines = md.split("\n");
  let currentTopic = "General";
  let currentLines: string[] = [];

  for (const line of lines) {
    const headingMatch = line.match(/^##\s+(.+)/);
    if (headingMatch) {
      if (currentLines.length > 0) {
        sections.push({ topic: currentTopic, content: currentLines.join("\n").trim() });
      }
      currentTopic = headingMatch[1].trim();
      currentLines = [];
    } else {
      currentLines.push(line);
    }
  }
  if (currentLines.length > 0) {
    sections.push({ topic: currentTopic, content: currentLines.join("\n").trim() });
  }
  return sections;
}

export const useMemoryStore = create<MemoryState>()(
  persist(
    (set) => ({
      entries: [],
      rawMd: null,

      addEntry: (entry) => {
        const newEntry: MemoryEntry = {
          ...entry,
          id: crypto.randomUUID(),
          createdAt: new Date().toISOString(),
        };
        set((s) => ({ entries: [newEntry, ...s.entries] }));
      },

      removeEntry: (id) => {
        set((s) => ({ entries: s.entries.filter((e) => e.id !== id) }));
      },

      importFromMd: async (md) => {
        const sections = parseMdSections(md);
        const entries: MemoryEntry[] = sections.map((s) => ({
          id: crypto.randomUUID(),
          topic: s.topic,
          content: s.content,
          source: "import" as const,
          createdAt: new Date().toISOString(),
        }));
        set((s) => ({ entries: [...entries, ...s.entries], rawMd: md }));

        try {
          const result = await api.memory.import(md);
          return result;
        } catch {
          return { chunks: sections.length, feedback: `Importados ${sections.length} fragmentos (solo local, servidor no disponible)` };
        }
      },

      clearMemory: () => set({ entries: [], rawMd: null }),
    }),
    { name: "ozy-memory" },
  ),
);
