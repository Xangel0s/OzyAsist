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

export interface UserDTO {
  id: string;
  name: string;
  email?: string;
  avatarColor?: string;
  hasPin: boolean;
  role?: string;
  plan?: string;
  profileMd?: string;
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
  auth: {
    listProfiles: () => request<UserDTO[]>("/auth/profiles"),
    createProfile: (data: { name: string; email?: string; avatarColor?: string; pin?: string; role?: string }) =>
      request<UserDTO>("/auth/profiles", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    login: (data: { userId: string; pin?: string }) =>
      request<{ status: string; user: UserDTO }>("/auth/login", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    verifyPin: (data: { userId: string; pin: string }) =>
      request<{ valid: boolean; user: UserDTO }>("/auth/pin/verify", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    updatePin: (data: { userId: string; oldPin?: string; newPin: string }) =>
      request<{ status: string; hasPin: boolean }>("/auth/pin/update", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    deleteProfile: (userId: string) =>
      request<{ status: string }>(`/auth/profiles/${userId}`, { method: "DELETE" }),
  },

  chats: {
    list: () => request<ChatDTO[]>("/chats"),
    create: (data: { title: string; mode: string; provider?: string; model?: string; projectId?: string }) =>
      request<ChatDTO>("/chats", {
        method: "POST",
        body: JSON.stringify({ name: data.title, mode: data.mode, provider: data.provider, model: data.model, project_id: data.projectId }),
      }),
    getById: (id: string) => {
      if (!id) throw new Error("ID de chat vacío");
      return request<{ chat: ChatDTO; messages: MessageDTO[] }>(`/chats/${id}`);
    },
    update: (id: string, data: { name?: string; provider?: string; model?: string }) => {
      if (!id) throw new Error("ID de chat vacío");
      return request<ChatDTO>(`/chats/${id}`, {
        method: "PATCH",
        body: JSON.stringify(data),
      });
    },
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
    readFile: (id: string, path: string) =>
      request<{ path: string; content: string; size: number }>(
        `/projects/${id}/file?path=${encodeURIComponent(path)}`,
      ),
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

  mcp: {
    list: () => request<MCPConnectorDTO[]>("/mcp"),
    create: (d: {name: string, command: string, args: string[], env: string[]}) => request<MCPConnectorDTO>("/mcp", { method: "POST", body: JSON.stringify(d) }),
    delete: (id: string) => request<{ status: string }>(`/mcp/${id}`, { method: "DELETE" }),
    reconnect: (id: string) => request<{ status: string }>(`/mcp/${id}/reconnect`, { method: "POST" }),
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
    available: () => request<any[]>("/models/available"),
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

  settings: {
    update: (keys: {
      opencode_key?: string;
      openai_key?: string;
      openrouter_key?: string;
      anthropic_key?: string;
      deepseek_key?: string;
      local_host_url?: string;
      ollama_url?: string;
    }) =>
      request<{ status: string; providers: any[] }>("/settings", {
        method: "PUT",
        body: JSON.stringify(keys),
      }),
  },

  git: {
    status: (projectId: string) =>
      request<GitStatusDTO>(`/projects/${projectId}/git/status`),
    init: (projectId: string) =>
      request<{ message: string; output: string }>(`/projects/${projectId}/git/init`, { method: "POST" }),
    stage: (projectId: string, paths: string[], all = false) =>
      request<{ message: string; output: string }>(`/projects/${projectId}/git/stage`, {
        method: "POST",
        body: JSON.stringify({ paths, all }),
      }),
    unstage: (projectId: string, paths: string[], all = false) =>
      request<{ message: string; output: string }>(`/projects/${projectId}/git/unstage`, {
        method: "POST",
        body: JSON.stringify({ paths, all }),
      }),
    commit: (projectId: string, message: string, sync = false) =>
      request<{ message: string; output: string; syncOutput?: string }>(`/projects/${projectId}/git/commit`, {
        method: "POST",
        body: JSON.stringify({ message, sync }),
      }),
    sync: (projectId: string) =>
      request<{ message: string; pull: string; push?: string; pushErr?: string }>(`/projects/${projectId}/git/sync`, {
        method: "POST",
      }),
    generateMsg: (projectId: string) =>
      request<{ message: string; highlights: string[] }>(`/projects/${projectId}/git/generate-msg`, {
        method: "POST",
      }),
  },

  terminal: {
    exec: (projectId: string, command: string, timeoutSecs = 30) =>
      request<TerminalExecResult>(`/projects/${projectId}/terminal/exec`, {
        method: "POST",
        body: JSON.stringify({ command, timeoutSecs }),
      }),
  },

  security: {
    getAuditTrail: () =>
      request<{ is_valid: boolean; total_verified: number; logs: AuditEntryDTO[] }>("/security/audit"),
    getTaskSnapshots: (taskId: string) =>
      request<{ task_id: string; snapshots: ShadowSnapshotDTO[] }>(`/security/snapshots/${taskId}`),
    undoTask: (taskId: string) =>
      request<{ success: boolean; task_id: string; restored_files: string[]; message: string }>(`/security/undo/${taskId}`, {
        method: "POST",
      }),
    killSuspectProcess: (pid: number) =>
      request<{ success: boolean; pid: number; message: string }>("/security/watchdog/kill", {
        method: "POST",
        body: JSON.stringify({ pid }),
      }),
  },

  audio: {
    getDevices: () =>
      request<{ devices: { id: number; name: string; type: string; channels: number; is_default: boolean }[]; total: number }>("/audio/devices"),
    getStatus: () =>
      request<{ status: string; hardware_access: boolean; engine: string; input_devices: number; vad_enabled: boolean }>("/audio/status"),
    testWakeWord: (text: string) =>
      request<{ match: { matched: boolean; keyword: string; confidence: number; clean_phrase: string } }>("/audio/wakeword/test", {
        method: "POST",
        body: JSON.stringify({ text }),
      }),
  },

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  post: <T = any>(path: string, body?: any) =>
    request<T>(path.startsWith("/api") ? path.slice(4) : path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  get: <T = any>(path: string) =>
    request<T>(path.startsWith("/api") ? path.slice(4) : path),
};

export interface AuditEntryDTO {
  id: number;
  timestamp: string;
  agent: string;
  action: string;
  details: string;
  prev_hash: string;
  record_hash: string;
}

export interface ShadowSnapshotDTO {
  id: string;
  task_id: string;
  file_path: string;
  created_at: string;
}

export interface GitFileItem {
  path: string;
  status: "M" | "A" | "D" | "U" | "R";
  added: number;
  deleted: number;
  staged: boolean;
}

export interface GitStatusDTO {
  initialized: boolean;
  branch: string;
  ahead: number;
  behind: number;
  files: GitFileItem[];
  totalAdded: number;
  totalDeleted: number;
  error?: string;
}

export interface TerminalExecResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  duration: string;
  error?: string;
}

export interface MCPConnectorDTO {
  id: string;
  userId: string;
  name: string;
  command: string;
  args: string;
  env: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}
