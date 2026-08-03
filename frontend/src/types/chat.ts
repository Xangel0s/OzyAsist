export interface Message {
  id: string;
  chatId: string;
  role: "user" | "assistant" | "system";
  content: string;
  createdAt: string;
}

export interface Chat {
  id: string;
  title: string;
  type: "chat" | "task";
  status: "active" | "archived" | "completed";
  createdAt: string;
  updatedAt: string;
  messages?: Message[];
}

export interface Provider {
  id: string;
  name: string;
  provider: string;
  model: string;
  active: boolean;
}
