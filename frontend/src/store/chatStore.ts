import { create } from "zustand";
import { api } from "../services/api";
import { wsClient } from "../services/ws";
import { useMemoryStore } from "./memoryStore";
import { useToastStore } from "./toastStore";

export interface Message {
  id: string;
  role: "user" | "assistant" | "tool";
  content: string;
  timestamp: string;
  feedback?: "like" | "dislike" | null;
}

export interface Chat {
  id: string;
  title: string;
  mode: "chat" | "code";
  provider: string;
  model: string;
  messages: Message[];
  createdAt: string;
  _messagesLoaded?: boolean;
}

interface ChatState {
  chats: Chat[];
  activeChatId: string | null;
  isResponding: boolean;
  coworkMode: boolean;
  consentPending: { chatId: string; intent: string } | null;
  setActiveChat: (id: string) => void;
  addChat: (chat: Chat) => void;
  addMessage: (chatId: string, message: Message) => void;
  loadChats: () => Promise<void>;
  createChat: (mode: "chat" | "code", projectId?: string) => Promise<string | null>;
  sendMessage: (chatId: string, content: string) => Promise<void>;
  deleteChat: (chatId: string) => Promise<void>;
  updateChatTitle: (chatId: string, title: string) => void;
  updateMessage: (chatId: string, messageId: string, updates: Partial<Message>) => void;
  setMessageFeedback: (chatId: string, messageId: string, feedback: "like" | "dislike" | null) => Promise<void>;
  resolveConsent: (decision: "always" | "once" | "no") => void;
  dismissConsent: () => void;
  regenerateMessage: (chatId: string) => Promise<void>;
  toggleCowork: () => void;
  setCoworkMode: (mode: boolean) => void;
  loadMessages: (chatId: string) => Promise<void>;
}

export const useChatStore = create<ChatState>((set, get) => ({
  chats: [],
  activeChatId: null,
  isResponding: false,
  coworkMode: false,
  consentPending: null,

  setActiveChat: (id) => {
    set({ activeChatId: id });
    const chat = get().chats.find((c) => c.id === id);
    if (chat && chat.messages.length === 0 && !chat._messagesLoaded) {
      get().loadMessages(id);
    }
  },

  addChat: (chat) => set((s) => ({ chats: [chat, ...s.chats] })),

  addMessage: (chatId, message) =>
    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === chatId
          ? { ...c, messages: [...c.messages, message] }
          : c,
      ),
    })),

  updateMessage: (chatId, messageId, updates) =>
    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === chatId
          ? {
              ...c,
              messages: c.messages.map((m) =>
                m.id === messageId ? { ...m, ...updates } : m,
              ),
            }
          : c,
      ),
    })),

  updateChatTitle: (chatId, title) =>
    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === chatId ? { ...c, title } : c,
      ),
    })),

  loadChats: async () => {
    try {
      const dtos = await api.chats.list();
      const chats: Chat[] = dtos.map((dto: any) => ({
        id: dto.id,
        title: dto.name || "Chat",
        mode: dto.mode || "chat",
        provider: dto.provider || "opencode",
        model: dto.model || "deepseek-v4-pro",
        messages: [],
        createdAt: dto.createdAt,
      }));
      set({ chats });
    } catch {
      // server not available — keep local state
    }
  },

  createChat: async (mode, projectId?) => {
    try {
      const dto = await api.chats.create({
        title: "Nuevo chat",
        mode,
        provider: "opencode",
        model: "deepseek-v4-pro",
        projectId,
      });
      const chat: Chat = {
        id: dto.id,
        title: dto.name || "Nuevo chat",
        mode: mode,
        provider: dto.provider || "opencode",
        model: dto.model || "deepseek-v4-pro",
        messages: [],
        createdAt: dto.createdAt,
        _messagesLoaded: true,
      };
      set((s) => ({ chats: [chat, ...s.chats], activeChatId: chat.id }));
      return chat.id;
    } catch {
      return null;
    }
  },

  deleteChat: async (chatId) => {
    try {
      await api.chats.delete(chatId);
      set((s) => ({
        chats: s.chats.filter((c) => c.id !== chatId),
        activeChatId: s.activeChatId === chatId ? null : s.activeChatId,
      }));
    } catch {
      // ignore
    }
  },

  sendMessage: async (chatId, content) => {
    const chat = get().chats.find((c) => c.id === chatId);
    if (!chat || get().isResponding) return;

    const isFirstMessage = chat.messages.length === 0;
    const title = isFirstMessage
      ? content.length > 36
        ? content.slice(0, 36) + "..."
        : content
      : chat.title;

    const userMsg: Message = {
      id: `u-${Date.now()}`,
      role: "user",
      content,
      timestamp: new Date().toISOString(),
    };

    set((s) => ({
      isResponding: true,
      chats: s.chats.map((c) =>
        c.id === chatId
          ? { ...c, messages: [...c.messages, userMsg], title: isFirstMessage ? title : c.title }
          : c,
      ),
    }));

    const assistantMsg: Message = {
      id: `a-${Date.now()}`,
      role: "assistant",
      content: "",
      timestamp: new Date().toISOString(),
    };

    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === chatId
          ? { ...c, messages: [...c.messages, assistantMsg] }
          : c,
      ),
    }));

    try {
      await wsClient.connect();

      const pendingContent: string[] = [];

      wsClient.sendMessage(chatId, content, {
        onText: (text) => {
          pendingContent.push(text);
          get().updateMessage(chatId, assistantMsg.id, {
            content: pendingContent.join(""),
          });
        },
        onToolCall: () => {},
        onWarn: (warning) => {
          useToastStore.getState().show(warning, "warning");
        },
        onConsentRequired: (intent) => {
          const chat = get().chats.find((c) => c.id === chatId);
          const isFirstMsg = chat?.messages.length === 1;
          if (isFirstMsg) {
            set((s) => ({
              isResponding: false,
              consentPending: { chatId, intent },
              chats: s.chats.map((c) =>
                c.id === chatId
                  ? { ...c, messages: c.messages.filter((m) => m.id !== assistantMsg.id) }
                  : c
              ),
            }));
          } else {
            set({ consentPending: { chatId, intent } });
          }
        },
        onDone: (messageId) => {
          const fullContent = pendingContent.join("");
          assistantMsg.id = messageId;
          assistantMsg.content = fullContent;
          assistantMsg.timestamp = new Date().toISOString();

          get().updateMessage(chatId, assistantMsg.id, {
            content: fullContent,
            timestamp: assistantMsg.timestamp,
          });

          const firstLine = fullContent.split("\n")[0].slice(0, 60);
          useMemoryStore.getState().addEntry({
            topic: `Asistente: ${firstLine.length < 60 ? firstLine : firstLine + "..."}`,
            content: fullContent,
            source: "chat",
          });

          set({ isResponding: false });
        },
        onError: (error) => {
          get().updateMessage(chatId, assistantMsg.id, {
            content: `[Error: ${error}]`,
          });
          set({ isResponding: false });
        },
      });
    } catch {
      get().updateMessage(chatId, assistantMsg.id, {
        content: "[Error: no se pudo conectar con el servidor]",
      });
      set({ isResponding: false });
    }
  },

  setMessageFeedback: async (chatId, messageId, feedback) => {
    get().updateMessage(chatId, messageId, { feedback });

    try {
      await api.chats.updateFeedback(chatId, messageId, feedback ?? "");
    } catch {
    }
  },

  regenerateMessage: async (chatId) => {
    const chat = get().chats.find((c) => c.id === chatId);
    if (!chat || get().isResponding) return;

    const messages = [...chat.messages];
    let lastUserIdx = -1;
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].role === "user") {
        lastUserIdx = i;
        break;
      }
    }
    if (lastUserIdx === -1) return;

    const lastUserContent = messages[lastUserIdx].content;
    const messagesToKeep = messages.slice(0, lastUserIdx);

    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === chatId ? { ...c, messages: messagesToKeep } : c,
      ),
    }));

    await get().sendMessage(chatId, lastUserContent);
  },

  resolveConsent: (decision) => {
    const pending = get().consentPending;
    if (!pending) return;
    set({ consentPending: null, isResponding: true });

    const assistantMsg: Message = {
      id: `a-${Date.now()}`,
      role: "assistant",
      content: "",
      timestamp: new Date().toISOString(),
    };
    set((s) => ({
      chats: s.chats.map((c) =>
        c.id === pending.chatId
          ? { ...c, messages: [...c.messages, assistantMsg] }
          : c
      ),
    }));

    const pendingContent: string[] = [];

    wsClient.sendConsentResponse(pending.chatId, decision, {
      onText: (text) => {
        pendingContent.push(text);
        get().updateMessage(pending.chatId, assistantMsg.id, { content: pendingContent.join("") });
      },
      onToolCall: () => {},
      onAgentStep: (result) => {
        try {
          const step = JSON.parse(result);
          pendingContent.push(`[Paso ${step.stepId}] ${step.success ? "OK" : "FAIL"}: ${step.output || step.error}\n`);
          get().updateMessage(pending.chatId, assistantMsg.id, { content: pendingContent.join("") });
        } catch {
          pendingContent.push(result);
          get().updateMessage(pending.chatId, assistantMsg.id, { content: pendingContent.join("") });
        }
      },
      onAgentDone: (_taskId, results) => {
        let summary = "";
        try {
          const steps = JSON.parse(results);
          steps.forEach((s: any) => {
            summary += `${s.success ? "✅" : "❌"} Paso ${s.stepId}: ${s.output || s.error}\n`;
          });
          get().updateMessage(pending.chatId, assistantMsg.id, { content: summary });
        } catch {
          summary = results;
          get().updateMessage(pending.chatId, assistantMsg.id, { content: results });
        }
        const firstLine = summary.split("\n")[0].slice(0, 60);
        useMemoryStore.getState().addEntry({
          topic: `Agente: ${firstLine || "Tarea completada"}`,
          content: summary,
          source: "chat" as any,
        });
        set({ isResponding: false });
      },
      onDone: () => {
        set({ isResponding: false });
      },
      onError: (error) => {
        get().updateMessage(pending.chatId, assistantMsg.id, { content: `[Error: ${error}]` });
        set({ isResponding: false });
      },
    });
  },

  dismissConsent: () => {
    set({ consentPending: null, isResponding: false });
  },

  toggleCowork: () => set((s) => ({ coworkMode: !s.coworkMode })),
  setCoworkMode: (mode) => set({ coworkMode: mode }),

  loadMessages: async (chatId) => {
    try {
      const data = await api.chats.getById(chatId);
      const messages: Message[] = (data.messages || []).map((m: any) => ({
        id: m.id,
        role: m.role,
        content: m.content,
        timestamp: m.createdAt || new Date().toISOString(),
        feedback: (m.feedback as "like" | "dislike" | null) || null,
      }));
      set((s) => ({
        chats: s.chats.map((c) =>
          c.id === chatId ? { ...c, messages } : c,
        ),
      }));
    } catch {
    }
  },
}));
