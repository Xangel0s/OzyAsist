const WS_URL = "ws://localhost:8080/ws";

export interface StreamCallbacks {
  onText: (text: string) => void;
  onToolCall: (toolCall: { toolId: string; toolName: string; toolInput: string }) => void;
  onDone: (messageId: string) => void;
  onError: (error: string) => void;
  onWarn?: (warning: string) => void;
  onConsentRequired?: (intent: string) => void;
  onAgentStep?: (result: string) => void;
  onAgentDone?: (taskId: string, result: string) => void;
}

interface StreamSession {
  chatId: string;
  callbacks: StreamCallbacks;
}

class WsClient {
  private ws: WebSocket | null = null;
  private session: StreamSession | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldReconnect = false;

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.shouldReconnect = true;
      this.ws = new WebSocket(WS_URL);

      this.ws.onopen = () => resolve();
      this.ws.onclose = () => {
        if (this.shouldReconnect) {
          this.scheduleReconnect();
        }
      };
      this.ws.onerror = () => reject(new Error("WS connection failed"));
      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.handleMessage(data);
        } catch {
          // ignore malformed messages
        }
      };
    });
  }

  private handleMessage(data: any) {
    if (!this.session) return;

    switch (data.type) {
      case "text":
        this.session.callbacks.onText(data.content ?? "");
        break;
      case "tool_call":
        this.session.callbacks.onToolCall({
          toolId: data.tool_id ?? "",
          toolName: data.tool_name ?? "",
          toolInput: data.tool_input ?? "",
        });
        break;
      case "done":
        this.session.callbacks.onDone(data.message_id ?? "");
        this.session = null;
        break;
      case "error":
        this.session.callbacks.onError(data.content ?? "Error desconocido");
        this.session = null;
        break;
      case "warn":
        if (this.session.callbacks.onWarn) {
          this.session.callbacks.onWarn(data.warning ?? "");
        }
        break;
      case "consent_required":
        if (this.session.callbacks.onConsentRequired) {
          this.session.callbacks.onConsentRequired(data.content ?? "");
        }
        break;
      case "agent_start":
        break;
      case "agent_step":
        if (this.session.callbacks.onAgentStep) {
          this.session.callbacks.onAgentStep(data.content ?? "");
        }
        break;
      case "agent_done":
        if (this.session.callbacks.onAgentDone) {
          this.session.callbacks.onAgentDone(data.message_id ?? "", data.content ?? "");
        }
        this.session = null;
        break;
    }
  }

  sendMessage(chatId: string, content: string, callbacks: StreamCallbacks, attachments?: { id: string; type: string }[]) {
    if (this.session) {
      callbacks.onError("Ya hay un streaming en curso");
      return;
    }

    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      callbacks.onError("No hay conexión WebSocket");
      return;
    }

    this.session = { chatId, callbacks };
    const msg: any = { type: "message", chat_id: chatId, content };
    if (attachments && attachments.length > 0) {
      msg.attachments = attachments;
    }
    this.ws.send(JSON.stringify(msg));
  }

  sendConsentResponse(chatId: string, decision: string, callbacks: StreamCallbacks) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.session = { chatId, callbacks };
    this.ws.send(JSON.stringify({ type: "consent_response", chat_id: chatId, content: decision }));
  }

  cancelStream(chatId: string) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "cancel", chat_id: chatId }));
    }
    this.session = null;
  }

  disconnect() {
    this.shouldReconnect = false;
    this.session = null;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.session = null;
      this.connect().catch(() => {});
    }, 3000);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export const wsClient = new WsClient();
