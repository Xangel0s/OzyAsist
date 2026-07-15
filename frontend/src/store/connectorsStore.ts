import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type ConnectorDTO } from "../services/api";

export type ConnectorType = "mcp" | "custom";

export interface Connector {
  id: string;
  name: string;
  type: ConnectorType;
  endpoint: string;
  authConfig: Record<string, string>;
  status: "connected" | "disconnected" | "error";
}

interface ConnectorsState {
  connectors: Connector[];
  loading: boolean;
  addConnector: (connector: Connector) => Promise<string | null>;
  removeConnector: (id: string) => Promise<void>;
  updateConnectorStatus: (id: string, status: Connector["status"]) => void;
  loadConnectors: () => Promise<void>;
}

const dtoToConnector = (dto: ConnectorDTO): Connector => {
  let authConfig: Record<string, string> = {};
  try { authConfig = JSON.parse(dto.authConfig || "{}"); } catch { /* ignore */ }
  return {
    id: dto.id,
    name: dto.name,
    type: dto.type as ConnectorType,
    endpoint: dto.endpoint,
    authConfig,
    status: "disconnected",
  };
};

export const useConnectorsStore = create<ConnectorsState>()(
  persist(
    (set) => ({
  connectors: [],
  loading: true,

  addConnector: async (connector) => {
    try {
      const dto = await api.connectors.create({
        name: connector.name,
        type: connector.type,
        endpoint: connector.endpoint,
        authConfig: JSON.stringify(connector.authConfig),
      });
      const mapped = dtoToConnector(dto);
      set((s) => ({ connectors: [...s.connectors, mapped] }));
      return mapped.id;
    } catch {
      return null;
    }
  },

  removeConnector: async (id) => {
    try {
      await api.connectors.delete(id);
      set((s) => ({ connectors: s.connectors.filter((c) => c.id !== id) }));
    } catch {
      // ignore
    }
  },

  updateConnectorStatus: (id, status) =>
    set((s) => ({
      connectors: s.connectors.map((c) => (c.id === id ? { ...c, status } : c)),
    })),

  loadConnectors: async () => {
    set({ loading: true });
    try {
      const dtos = await api.connectors.list();
      set({ connectors: dtos.map(dtoToConnector), loading: false });
    } catch {
      set({ loading: false });
    }
  },
}),
    {
      name: "ozy-connectors",
      partialize: (state) => ({
        connectors: state.connectors,
      }),
    },
  ),
);

useConnectorsStore.getState().loadConnectors();
