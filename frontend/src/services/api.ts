const BASE = "http://localhost:8080/api";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export interface ChatDTO {
  id: string;
  userId: string;
  projectId?: string;
  name: string;
  mode: "chat" | "code";
  provider: string;
  model: string;
  createdAt: string;
}

export interface MessageDTO {
  id: string;
  chatId: string;
  role: "user" | "assistant" | "tool";
  content: string;
  attachments?: string;
  toolCalls?: string;
  feedback?: string;
  createdAt: string;
}

export interface ProjectDTO {
  id: string;
  userId: string;
  name: string;
  rootPath?: string;
  instructionsMd?: string;
  permissionLevel?: string;
  createdAt: string;
}

export interface SkillDTO {
  id: string;
  name: string;
  description: string;
  triggerPattern: string;
  executionType: "script" | "prompt_template" | "api_call";
  config?: string;
}

export interface ConnectorDTO {
  id: string;
  name: string;
  type: "mcp" | "custom";
  endpoint: string;
  authConfig?: string;
}

export interface TreeNode {
  name: string;
  path: string;
  type: "file" | "folder";
  children?: TreeNode[];
}

export interface SearchHit {
  id: string;
  text: string;
  score: number;
}

export interface SearchResults {
  messages: SearchHit[];
  chats: SearchHit[];
  projects: SearchHit[];
  memory: SearchHit[];
}

export interface GraphEdge {
  id: string;
  project_id: string;
  from_symbol: string;
  to_symbol: string;
  edge_type: string;
  created_at: string;
}

export interface AgentTaskResult {
  taskId: string;
  plan: string;
  results: { stepId: number; success: boolean; output?: string; error?: string }[];
}

export const api = {
  chats: {
    list: () => request<ChatDTO[]>("/chats"),
    create: (data: { title: string; mode: string; provider?: string; model?: string; projectId?: string }) =>
      request<ChatDTO>("/chats", {
        method: "POST",
        body: JSON.stringify({ name: data.title, mode: data.mode, provider: data.provider, model: data.model, project_id: data.projectId }),
      }),
    getById: (id: string) => request<{ chat: ChatDTO; messages: MessageDTO[] }>(`/chats/${id}`),
    update: (id: string, data: { name?: string; provider?: string; model?: string }) =>
      request<ChatDTO>(`/chats/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      }),
    delete: (id: string) => request<void>(`/chats/${id}`, { method: "DELETE" }),
    updateFeedback: (chatId: string, messageId: string, feedback: string) =>
      request<{ ok: boolean }>(`/chats/${chatId}/messages/${messageId}/feedback`, {
        method: "PATCH",
        body: JSON.stringify({ feedback }),
      }),
  },

  projects: {
    list: () => request<ProjectDTO[]>("/projects"),
    create: (data: { name: string; rootPath?: string; instructionsMd?: string; permissionLevel?: string }) =>
      request<ProjectDTO>("/projects", {
        method: "POST",
        body: JSON.stringify({
          name: data.name,
          root_path: data.rootPath,
          instructions_md: data.instructionsMd,
          permission_level: data.permissionLevel,
        }),
      }),
    getById: (id: string) => request<ProjectDTO>(`/projects/${id}`),
    update: (id: string, data: { name?: string; rootPath?: string; instructionsMd?: string }) =>
      request<ProjectDTO>(`/projects/${id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: data.name,
          root_path: data.rootPath,
          instructions_md: data.instructionsMd,
        }),
      }),
    delete: (id: string) => request<void>(`/projects/${id}`, { method: "DELETE" }),
    index: (id: string) =>
      request<{ files: number; imports: number }>(`/projects/${id}/index`, { method: "POST" }),
    tree: (id: string) =>
      request<{ tree: TreeNode; indexed: boolean }>(`/projects/${id}/tree`),
    graph: (id: string, filepath: string) =>
      request<{ file: string; neighbors: { file: string; relation: string }[] }>(
        `/projects/${id}/graph/${encodeURIComponent(filepath)}`,
      ),
    fullGraph: (id: string) => request<GraphEdge[]>(`/projects/${id}/fullgraph`),
    uploadFiles: (id: string, files: { path: string; content: string }[]) =>
      request<{ files: number; imports: number }>(`/projects/${id}/upload-files`, {
        method: "POST",
        body: JSON.stringify(files),
      }),
  },

  skills: {
    list: () => request<SkillDTO[]>("/skills"),
    create: (data: Omit<SkillDTO, "id">) =>
      request<SkillDTO>("/skills", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    delete: (id: string) => request<void>(`/skills/${id}`, { method: "DELETE" }),
  },

  connectors: {
    list: () => request<ConnectorDTO[]>("/connectors"),
    create: (data: Omit<ConnectorDTO, "id">) =>
      request<ConnectorDTO>("/connectors", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    delete: (id: string) => request<void>(`/connectors/${id}`, { method: "DELETE" }),
  },

  memory: {
    import: (md: string) =>
      request<{ chunks: number; feedback: string }>("/memory/import", {
        method: "POST",
        body: JSON.stringify({ content: md }),
      }),
    search: (query: string) =>
      request<{ results: { text: string; score: number }[] }>(
        `/memory/search?q=${encodeURIComponent(query)}`,
      ),
  },

  search: {
    global: (q: string) =>
      request<SearchResults>(
        `/search?q=${encodeURIComponent(q)}`,
      ),
  },

  files: {
    upload: (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return fetch(`${BASE}/files/upload`, { method: "POST", body: formData }).then((r) => {
        if (!r.ok) throw new Error(`upload failed: ${r.status}`);
        return r.json() as Promise<{ id: string; filename: string; size: number; url: string }>;
      });
    },
    getUrl: (id: string) => `${BASE}/files/${id}`,
  },

  agent: {
    createTask: (data: { chatId?: string; projectId?: string; goal: string; permissionLevel?: string }) =>
      request<AgentTaskResult>("/agent/tasks", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    getTask: (id: string) => request<any>(`/agent/tasks/${id}`),
    cancelTask: (id: string) =>
      request<void>(`/agent/tasks/${id}/cancel`, { method: "POST" }),
    confirmAction: (actionId: string) =>
      request<void>(`/agent/tasks/${actionId}/confirm`, { method: "POST" }),
  },

  sidebar: {
    observe: (data: { context?: string; chatId?: string; projectId?: string }) =>
      request<{ analysis: string; context: string }>("/sidebar/observe", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    command: (data: { command: string; chatId?: string; projectId?: string }) =>
      request<{ response: string }>("/sidebar/command", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },

  models: {
    list: () => request<{ provider: string; models: string[] }[]>("/models"),
    select: (data: { provider: string; model: string }) =>
      request<void>("/models/select", { method: "POST", body: JSON.stringify(data) }),
  },

  users: {
    updateProfile: (profileMd: string) =>
      request<{ status: string }>("/users/profile", {
        method: "PUT",
        body: JSON.stringify({ profileMd }),
      }),
  },

  onboarding: {
    analyzeMemory: (content: string) =>
      request<{ feedback: string }>("/onboarding/analyze-memory", {
        method: "POST",
        body: JSON.stringify({ content }),
      }),
  },
};
