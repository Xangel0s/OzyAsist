import { create } from "zustand";

export interface Toast {
  id: string;
  message: string;
  type?: "success" | "error" | "warning" | "info" | string;
  timestamp: number;
}

interface ToastState {
  toasts: Toast[];
  show: (message: string, type?: "success" | "error" | "warning" | "info" | string) => void;
  dismiss: (id: string) => void;
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  show: (message, type = "success") => {
    const now = Date.now();
    const existing = get().toasts.find(
      (t) => t.message === message && now - t.timestamp < 2000
    );
    // Prevent duplicate spamming of the exact same toast within 2 seconds
    if (existing) return;

    const id = crypto.randomUUID();
    set((s) => ({
      toasts: [...s.toasts.slice(-3), { id, message, type, timestamp: now }],
    }));

    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    }, 3200);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));
