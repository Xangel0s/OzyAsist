import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type ChatDTO, type MessageDTO } from "../services/api";
import { wsClient } from "../services/ws";
import { useMemoryStore } from "./memoryStore";
import { useToastStore } from "./toastStore";
import type { ToolCallBlockData } from "../components/Code/ToolCallBlock";

export interface Message {
  id: string;
  role: "user" | "assistant" | "tool";
  content: string;
  timestamp: string;
  feedback?: "like" | "dislike" | null;
  toolCalls?: ToolCallBlockData[];
}

export interface Chat {
  id: string;
  title: string;
  projectId?: string;
  mode: "chat" | "code";
  provider: string;
  model: string;
  messages: Message[];
  createdAt: string;
  _messagesLoaded?: boolean;
}

export type AgentState = "idle" | "thinking" | "executing" | "awaiting";

interface ChatState {
  chats: Chat[];
  activeChatId: string | null;
  isResponding: boolean;
  agentState: AgentState;
  activeSessionId: string | null;
  consentPending: { chatId: string; intent: string } | null;
  defaultProvider: string;
  defaultModel: string;
  setActiveChat: (id: string) => void;
  addChat: (chat: Chat) => void;
  addMessage: (chatId: string, message: Message) => void;
  loadChats: () => Promise<void>;
  createChat: (mode: "chat" | "code", projectId?: string, provider?: string, model?: string) => Promise<string | null>;
  sendMessage: (chatId: string, content: string) => Promise<void>;
  deleteChat: (chatId: string) => Promise<void>;
  updateChatTitle: (chatId: string, title: string) => void;
  updateMessage: (chatId: string, messageId: string, updates: Partial<Message>) => void;
  setMessageFeedback: (chatId: string, messageId: string, feedback: "like" | "dislike" | null) => Promise<void>;
  resolveConsent: (decision: "always" | "once" | "no") => void;
  dismissConsent: () => void;
  regenerateMessage: (chatId: string) => Promise<void>;
  loadMessages: (chatId: string) => Promise<void>;
  clearChatMessages: (chatId: string) => void;
  updateChatProvider: (chatId: string, provider: string, model: string) => void;
  setDefaultModel: (model: string, provider: string) => void;
  cancelResponse: () => void;
  respondToolApproval: (toolId: string, approved: boolean) => void;
  loadProviders: () => Promise<void>;
}

export const useChatStore = create<ChatState>()(
  persist(
    (set, get) => ({
      chats: [],
      activeChatId: null,
      isResponding: false,
      agentState: "idle",
      activeSessionId: null,
      consentPending: null,
      defaultProvider: "",
      defaultModel: "",

      setActiveChat: (id) => {
        set({ activeChatId: id });
        get().loadMessages(id);
      },

      addChat: (chat) => {
        set((s) => {
          const exists = s.chats.some((c) => c.id === chat.id);
          if (exists) {
            return {
              chats: s.chats.map((c) => (c.id === chat.id ? { ...c, ...chat } : c)),
              activeChatId: chat.id,
            };
          }
          return {
            chats: [chat, ...s.chats],
            activeChatId: chat.id,
          };
        });
      },

      addMessage: (chatId, message) => {
        set((s) => ({
          chats: s.chats.map((c) =>
            c.id === chatId ? { ...c, messages: [...c.messages, message] } : c,
          ),
        }));
      },

      loadChats: async () => {
        try {
          const serverChats = await api.chats.list();
          set((s) => {
            const serverMap = new Map(serverChats.map((c: ChatDTO) => [c.id, c]));
            const merged: Chat[] = serverChats.map((sc: ChatDTO) => {
              const local = s.chats.find((c) => c.id === sc.id);
              return {
                id: sc.id,
                title: sc.name || "Nuevo Chat",
                projectId: sc.projectId || undefined,
                mode: (sc.mode as "chat" | "code") || "chat",
                provider: sc.provider || s.defaultProvider || "",
                model: sc.model || s.defaultModel || "",
                messages: local?._messagesLoaded ? local.messages : local?.messages || [],
                createdAt: sc.createdAt || new Date().toISOString(),
                _messagesLoaded: local?._messagesLoaded,
              };
            });
            for (const local of s.chats) {
              if (!serverMap.has(local.id)) {
                merged.push(local);
              }
            }
            return { chats: merged };
          });
        } catch {
        }
      },

      createChat: async (mode = "chat", projectId, provider, model) => {
        const id = `chat-${Date.now()}`;
        const title = mode === "code" ? "Nuevo Código" : "Nuevo Chat";
        const chosenProvider = provider || get().defaultProvider || "";
        const chosenModel = model || get().defaultModel || "";

        const newChat: Chat = {
          id,
          title,
          projectId,
          mode,
          provider: chosenProvider,
          model: chosenModel,
          messages: [],
          createdAt: new Date().toISOString(),
          _messagesLoaded: true,
        };

        set((s) => ({
          chats: [newChat, ...s.chats],
          activeChatId: id,
        }));

        try {
          const created = await api.chats.create({
            title,
            mode,
            projectId: projectId || undefined,
            provider: chosenProvider,
            model: chosenModel,
          });
          if (created?.id && created.id !== id) {
            set((s) => ({
              chats: s.chats.map((c) =>
                c.id === id ? { ...c, id: created.id } : c,
              ),
              activeChatId: created.id,
            }));
            return created.id;
          }
        } catch {
        }
        return id;
      },

      deleteChat: async (chatId) => {
        set((s) => {
          const next = s.chats.filter((c) => c.id !== chatId);
          const nextActive =
            s.activeChatId === chatId
              ? next.length > 0
                ? next[0].id
                : null
              : s.activeChatId;
          return { chats: next, activeChatId: nextActive };
        });

        try {
          await api.chats.delete(chatId);
        } catch {
        }
      },

      updateChatTitle: async (chatId, title) => {
        set((s) => ({
          chats: s.chats.map((c) => (c.id === chatId ? { ...c, title } : c)),
        }));
        try {
          await api.chats.update(chatId, { name: title });
        } catch {
        }
      },

      updateMessage: (chatId, messageId, updates) => {
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
        }));
      },

      sendMessage: async (chatId, content) => {
        const chat = get().chats.find((c) => c.id === chatId);
        if (!chat || get().isResponding || get().consentPending) return;

        const isFirstMessage = chat.messages.length === 0;
        const title = content.slice(0, 30) + (content.length > 30 ? "..." : "");

        const userMsg: Message = {
          id: `u-${Date.now()}`,
          role: "user",
          content,
          timestamp: new Date().toISOString(),
        };

        const assistantMsg: Message = {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: "",
          timestamp: new Date().toISOString(),
          toolCalls: [],
        };

        set((s) => ({
          isResponding: true,
          agentState: "thinking",
          chats: s.chats.map((c) =>
            c.id === chatId
              ? { ...c, messages: [...c.messages, userMsg, assistantMsg], title: isFirstMessage ? title : c.title }
              : c,
          ),
        }));

        if (isFirstMessage && title !== chat.title) {
          get().updateChatTitle(chatId, title);
        }

        try {
          await wsClient.connect();

          const pendingContent: string[] = [];
          const currentToolCalls: ToolCallBlockData[] = [];

          wsClient.sendMessage(chatId, content, {
            onSessionStarted: (sessionId) => {
              set({ activeSessionId: sessionId });
            },

            onStateSync: (ev) => {
              set({ agentState: ev.state });
            },

            onText: (text) => {
              pendingContent.push(text);
              get().updateMessage(chatId, assistantMsg.id, {
                content: pendingContent.join(""),
              });
            },

            onToolCall: (ev) => {
              const existingIdx = currentToolCalls.findIndex((t) => t.toolId === ev.toolId);
              const toolData: ToolCallBlockData = {
                toolId: ev.toolId,
                toolName: ev.toolName,
                toolInput: ev.toolInput,
                status: "calling",
              };
              if (existingIdx >= 0) {
                currentToolCalls[existingIdx] = { ...currentToolCalls[existingIdx], ...toolData };
              } else {
                currentToolCalls.push(toolData);
              }
              set({ agentState: "executing" });
              get().updateMessage(chatId, assistantMsg.id, {
                toolCalls: [...currentToolCalls],
              });
            },

            onToolApprovalRequest: (ev) => {
              const existingIdx = currentToolCalls.findIndex((t) => t.toolId === ev.toolId);
              const toolData: ToolCallBlockData = {
                toolId: ev.toolId,
                toolName: ev.toolName,
                toolInput: ev.toolInput,
                status: "awaiting_approval",
              };
              if (existingIdx >= 0) {
                currentToolCalls[existingIdx] = toolData;
              } else {
                currentToolCalls.push(toolData);
              }
              set({ agentState: "awaiting" });
              get().updateMessage(chatId, assistantMsg.id, {
                toolCalls: [...currentToolCalls],
              });
            },

            onToolResult: (ev) => {
              const existingIdx = currentToolCalls.findIndex((t) => t.toolId === ev.toolId);
              if (existingIdx >= 0) {
                currentToolCalls[existingIdx] = {
                  ...currentToolCalls[existingIdx],
                  status: ev.toolSuccess ? "success" : "error",
                  output: ev.toolOutput,
                  durationMs: ev.durationMs,
                };
              }
              get().updateMessage(chatId, assistantMsg.id, {
                toolCalls: [...currentToolCalls],
              });
            },

            onAgentCompleted: (_ev) => {
              const fullContent = pendingContent.join("");
              const updated: Partial<Message> = {
                content: fullContent,
                timestamp: new Date().toISOString(),
                toolCalls: [...currentToolCalls],
              };
              get().updateMessage(chatId, assistantMsg.id, updated);

              if (fullContent.trim()) {
                const firstLine = fullContent.split("\n")[0].slice(0, 60);
                useMemoryStore.getState().addEntry({
                  topic: `Agente: ${firstLine.length < 60 ? firstLine : firstLine + "..."}`,
                  content: fullContent,
                  source: "chat",
                });
              }

              set({ isResponding: false, agentState: "idle", activeSessionId: null });
            },

            onWarn: (warning) => {
              useToastStore.getState().show(warning, "warning");
            },

            onConsentRequired: (intent) => {
              const currentChat = get().chats.find((c) => c.id === chatId);
              const hasMsgs = (currentChat?.messages.length ?? 0) > 2;
              if (!hasMsgs) {
                set((s) => ({
                  isResponding: false,
                  agentState: "idle",
                  consentPending: { chatId, intent },
                  chats: s.chats.map((c) =>
                    c.id === chatId
                      ? { ...c, messages: c.messages.filter((m) => m.id !== assistantMsg.id) }
                      : c
                  ),
                }));
              } else {
                set({ consentPending: { chatId, intent }, agentState: "idle" });
              }
            },

            onDone: (_messageId) => {
              const fullContent = pendingContent.join("");
              const updated: Partial<Message> = {
                content: fullContent,
                timestamp: new Date().toISOString(),
                toolCalls: [...currentToolCalls],
              };
              get().updateMessage(chatId, assistantMsg.id, updated);

              if (fullContent.trim()) {
                const firstLine = fullContent.split("\n")[0].slice(0, 60);
                useMemoryStore.getState().addEntry({
                  topic: `Asistente: ${firstLine.length < 60 ? firstLine : firstLine + "..."}`,
                  content: fullContent,
                  source: "chat",
                });
              }

              set({ isResponding: false, agentState: "idle", activeSessionId: null });
            },

            onError: (error) => {
              get().updateMessage(chatId, assistantMsg.id, {
                content: `[Error: ${error}]`,
                toolCalls: [...currentToolCalls],
              });
              set({ isResponding: false, agentState: "idle", activeSessionId: null });
            },
          });
        } catch {
          get().updateMessage(chatId, assistantMsg.id, {
            content: "[Error: no se pudo conectar con el servidor]",
          });
          set({ isResponding: false, agentState: "idle", activeSessionId: null });
        }
      },

      cancelResponse: () => {
        const activeChatId = get().activeChatId;
        if (activeChatId) {
          wsClient.cancelStream(activeChatId);
        }
        set({ isResponding: false, agentState: "idle", activeSessionId: null });
      },

      respondToolApproval: (toolId: string, approved: boolean) => {
        wsClient.respondToolApproval(toolId, approved);
        set({ agentState: approved ? "executing" : "idle" });

        const activeChatId = get().activeChatId;
        if (activeChatId) {
          set((s) => ({
            chats: s.chats.map((c) => {
              if (c.id !== activeChatId) return c;
              const msgs = [...c.messages];
              if (msgs.length === 0) return c;
              const last = { ...msgs[msgs.length - 1] };
              if (last.toolCalls) {
                last.toolCalls = last.toolCalls.map((t) =>
                  t.toolId === toolId
                    ? {
                        ...t,
                        status: approved ? "calling" : "error",
                        output: approved ? undefined : "Herramienta denegada por el usuario",
                      }
                    : t
                );
                msgs[msgs.length - 1] = last;
              }
              return { ...c, messages: msgs };
            }),
          }));
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
        if (!chat || get().isResponding || get().consentPending) return;

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
        set({ consentPending: null, isResponding: true, agentState: "thinking" });

        const assistantMsg: Message = {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: "",
          timestamp: new Date().toISOString(),
          toolCalls: [],
        };
        set((s) => ({
          chats: s.chats.map((c) =>
            c.id === pending.chatId
              ? { ...c, messages: [...c.messages, assistantMsg] }
              : c
          ),
        }));

        const pendingContent: string[] = [];

        const sent = wsClient.sendConsentResponse(pending.chatId, decision, {
          onText: (text) => {
            pendingContent.push(text);
            get().updateMessage(pending.chatId, assistantMsg.id, { content: pendingContent.join("") });
          },
          onDone: () => {
            const fullContent = pendingContent.join("");
            if (!fullContent.trim()) {
              get().updateMessage(pending.chatId, assistantMsg.id, { content: "[Acción rechazada]" });
            }
            set({ isResponding: false, agentState: "idle" });
          },
          onError: (error) => {
            get().updateMessage(pending.chatId, assistantMsg.id, { content: `[Error: ${error}]` });
            set({ isResponding: false, agentState: "idle" });
          },
        });
        if (!sent) {
          get().updateMessage(pending.chatId, assistantMsg.id, { content: "[Error: No hay conexión con el servidor]" });
          set({ isResponding: false, agentState: "idle" });
        }
      },

      dismissConsent: () => {
        const pending = get().consentPending;
        if (pending) {
          set({ consentPending: null });
        }
      },

      updateChatProvider: (chatId, provider, model) =>
        set((s) => ({
          chats: s.chats.map((c) =>
            c.id === chatId ? { ...c, provider, model } : c,
          ),
        })),

      setDefaultModel: (model, provider) =>
        set({ defaultModel: model, defaultProvider: provider }),

      loadProviders: async () => {
        try {
          const opencodeKey = localStorage.getItem("opencode_key") || "";
          const openaiKey = localStorage.getItem("openai_key") || "";
          const openrouterKey = localStorage.getItem("openrouter_key") || "";
          const anthropicKey = localStorage.getItem("anthropic_key") || "";

          if (opencodeKey || openaiKey || openrouterKey || anthropicKey) {
            await api.settings.update({
              opencode_key: opencodeKey,
              openai_key: openaiKey,
              openrouter_key: openrouterKey,
              anthropic_key: anthropicKey,
            }).catch(() => {});
          }

          const providers = await api.models.list();
          if (providers.length > 0) {
            const first = providers[0];
            const modelId = first.models[0] || "";
            set({ defaultProvider: first.provider, defaultModel: modelId });
          } else {
            set({ defaultProvider: "", defaultModel: "" });
          }
        } catch {
          set({ defaultProvider: "", defaultModel: "" });
        }
      },

      loadMessages: async (chatId) => {
        try {
          const data = await api.chats.getById(chatId);
          const messages: Message[] = (data.messages || []).map((m: MessageDTO) => ({
            id: m.id,
            role: m.role,
            content: m.content,
            timestamp: m.createdAt || new Date().toISOString(),
            feedback: (m.feedback as "like" | "dislike" | null) || null,
          }));
          set((s) => ({
            chats: s.chats.map((c) =>
              c.id === chatId ? { ...c, messages, _messagesLoaded: true } : c,
            ),
          }));
        } catch {
        }
      },

      clearChatMessages: (chatId: string) => {
        set((s) => ({
          chats: s.chats.map((c) => (c.id === chatId ? { ...c, messages: [] } : c)),
        }));
      },
    }),
    {
      name: "ozy-chats",
      partialize: (state) => ({
        chats: state.chats,
        activeChatId: state.activeChatId,
        defaultProvider: state.defaultProvider,
        defaultModel: state.defaultModel,
      }),
    },
  ),
);

// Load chats and providers from backend on init
useChatStore.getState().loadProviders();
useChatStore.getState().loadChats();
