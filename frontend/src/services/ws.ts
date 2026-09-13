const WS_URL = "ws://localhost:8080/ws";

// ============================================================
// Tipos del protocolo de eventos del ReAct Loop
// ============================================================

export interface ToolCallEvent {
  toolId: string;
  toolName: string;
  toolInput: string;
}

export interface ToolResultEvent {
  toolId: string;
  toolOutput: string;
  toolSuccess: boolean;
  durationMs: number;
}

export interface ToolApprovalRequestEvent {
  toolId: string;
  toolName: string;
  toolInput: string;
}

export interface AgentStateEvent {
  state: "thinking" | "executing" | "awaiting" | "idle";
}

export interface AgentCompletedEvent {
  taskId: string;
  messageId: string;
  turns: number;
}

// StreamCallbacks — unificados para Modo Chat y Modo Code (ReAct Loop)
export interface StreamCallbacks {
  // Modo Chat + Modo Code (texto)
  onText: (text: string) => void;
  onDone: (messageId: string) => void;
  onError: (error: string) => void;
  onWarn?: (warning: string) => void;
  onConsentRequired?: (intent: string) => void;

  // ReAct Loop — Modo Code
  onSessionStarted?: (sessionId: string) => void;
  onStateSync?: (ev: AgentStateEvent) => void;
  onToolCall?: (ev: ToolCallEvent) => void;
  onToolApprovalRequest?: (ev: ToolApprovalRequestEvent) => void;
  onToolResult?: (ev: ToolResultEvent) => void;
  onAgentCompleted?: (ev: AgentCompletedEvent) => void;
}

interface StreamSession {
  chatId: string;
  callbacks: StreamCallbacks;
  sessionId?: string; // ID del loop agéntico activo (para cancelar/aprobar)
}

class WsClient {
  private ws: WebSocket | null = null;
  private session: StreamSession | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private shouldReconnect = false;

  private statusListeners: Array<(connected: boolean) => void> = [];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private eventListeners: Map<string, Set<(data: any) => void>> = new Map();

  subscribeStatus(listener: (connected: boolean) => void): () => void {
    this.statusListeners.push(listener);
    listener(this.isConnected());
    return () => {
      this.statusListeners = this.statusListeners.filter((l) => l !== listener);
    };
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  subscribe(eventType: string, handler: (data: any) => void): () => void {
    if (!this.eventListeners.has(eventType)) {
      this.eventListeners.set(eventType, new Set());
    }
    this.eventListeners.get(eventType)!.add(handler);
    // Ensure WS is connected
    this.connect().catch(() => {});
    return () => {
      this.eventListeners.get(eventType)?.delete(handler);
    };
  }

  private notifyStatus(connected: boolean) {
    this.statusListeners.forEach((l) => l(connected));
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.notifyStatus(true);
        resolve();
        return;
      }
      if (this.ws?.readyState === WebSocket.CONNECTING) {
        this.ws.onopen = () => {
          this.notifyStatus(true);
          resolve();
        };
        this.ws.onerror = () => {
          this.notifyStatus(false);
          reject(new Error("WS connection failed"));
        };
        return;
      }

      this.shouldReconnect = true;
      this.ws = new WebSocket(WS_URL);

      this.ws.onopen = () => {
        this.notifyStatus(true);
        if (this.pingTimer) clearInterval(this.pingTimer);
        this.pingTimer = setInterval(() => {
          if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: "ping" }));
          }
        }, 20000);
        resolve();
      };
      this.ws.onclose = () => {
        this.notifyStatus(false);
        if (this.pingTimer) {
          clearInterval(this.pingTimer);
          this.pingTimer = null;
        }
        if (this.session) {
          this.session.callbacks.onError("Conexión perdida con el servidor");
          this.session = null;
        }
        if (this.shouldReconnect) {
          this.scheduleReconnect();
        }
      };
      this.ws.onerror = () => {
        this.notifyStatus(false);
        reject(new Error("WS connection failed"));
      };
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

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private handleMessage(data: Record<string, any>) {
    // 1. Dispatch to global event subscribers
    if (data?.type && this.eventListeners.has(data.type)) {
      this.eventListeners.get(data.type)!.forEach((listener) => {
        try {
          listener(data);
        } catch (err) {
          console.error("Error in WS subscriber:", err);
        }
      });
    }

    if (!this.session) return;
    const cb = this.session.callbacks;

    switch (data.type) {
      // --- Modo Chat + Modo Code (texto) ---
      case "text":
        cb.onText(data.content ?? "");
        break;

      case "done":
        cb.onDone(data.message_id ?? "");
        this.session = null;
        break;

      case "error":
        cb.onError(data.error ?? data.content ?? "Error desconocido");
        this.session = null;
        break;

      case "warn":
        cb.onWarn?.(data.warning ?? "");
        break;

      case "consent_required":
        cb.onConsentRequired?.(data.content ?? "");
        this.session = null;
        break;

      // --- ReAct Loop — Modo Code ---
      case "agent:session_started":
        // El backend nos da el sessionId para poder cancelar o responder aprobaciones
        if (this.session && data.session_id) {
          this.session.sessionId = data.session_id;
        }
        cb.onSessionStarted?.(data.session_id ?? "");
        break;

      case "message:delta":
        cb.onText(data.content ?? "");
        break;

      case "state:sync":
        cb.onStateSync?.({ state: data.state ?? "idle" });
        break;

      case "tool:call":
        cb.onToolCall?.({
          toolId: data.tool_id ?? "",
          toolName: data.tool_name ?? "",
          toolInput: data.tool_input ?? "",
        });
        break;

      case "tool:approval_request":
        cb.onToolApprovalRequest?.({
          toolId: data.tool_id ?? "",
          toolName: data.tool_name ?? "",
          toolInput: data.tool_input ?? "",
        });
        break;

      case "tool:result":
        cb.onToolResult?.({
          toolId: data.tool_id ?? "",
          toolOutput: data.tool_output ?? "",
          toolSuccess: data.tool_success ?? false,
          durationMs: data.duration_ms ?? 0,
        });
        break;

      case "agent:completed":
        cb.onAgentCompleted?.({
          taskId: data.task_id ?? "",
          messageId: data.message_id ?? "",
          turns: data.turns ?? 0,
        });
        this.session = null;
        break;

      // Legado — compatibilidad con mensajes del viejo agente
      case "agent_start":
        break;
      case "agent_step":
        break;
      case "agent_done":
        this.session = null;
        break;
    }
  }

  sendMessage(
    chatId: string,
    content: string,
    callbacks: StreamCallbacks,
    attachments?: { id: string; type: string }[],
    voiceMode?: boolean
  ) {
    if (this.session) {
      callbacks.onError("Ya hay un streaming en curso");
      return;
    }
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      callbacks.onError("No hay conexión WebSocket");
      return;
    }

    this.session = { chatId, callbacks };
    const msg: {
      type: string;
      chat_id: string;
      content: string;
      attachments?: { id: string; type: string }[];
      voice_mode?: boolean;
    } = { type: "message", chat_id: chatId, content };
    if (attachments && attachments.length > 0) {
      msg.attachments = attachments;
    }
    if (voiceMode) {
      msg.voice_mode = true;
    }
    this.ws.send(JSON.stringify(msg));
  }

  sendConsentResponse(chatId: string, decision: string, callbacks: StreamCallbacks): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false;
    this.session = { chatId, callbacks };
    this.ws.send(JSON.stringify({ type: "consent_response", chat_id: chatId, content: decision }));
    return true;
  }

  /** Responde a una solicitud de aprobación de herramienta del ReAct Loop */
  respondToolApproval(toolId: string, approved: boolean) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    if (!this.session?.sessionId) return;
    this.ws.send(
      JSON.stringify({
        type: "tool_approval",
        session_id: this.session.sessionId,
        tool_id: toolId,
        approved,
      })
    );
  }

  cancelStream(chatId: string) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "cancel", chat_id: chatId }));
      // Si hay una sesión de loop agéntico, cancelarla también
      if (this.session?.sessionId) {
        this.ws.send(
          JSON.stringify({ type: "cancel_session", session_id: this.session.sessionId })
        );
      }
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
export const wsService = wsClient;
