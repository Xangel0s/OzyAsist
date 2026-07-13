import { useState } from "react";
import { useConnectorsStore, type Connector } from "../../store/connectorsStore";
import ConnectorConfig from "./ConnectorConfig";

const statusColors: Record<string, string> = {
  connected: "bg-primary-container",
  disconnected: "bg-surface-variant",
  error: "bg-error",
};

const statusLabels: Record<string, string> = {
  connected: "Conectado",
  disconnected: "Desconectado",
  error: "Error",
};

export default function ConnectorsPage() {
  const connectors = useConnectorsStore((s) => s.connectors);
  const removeConnector = useConnectorsStore((s) => s.removeConnector);
  const updateConnectorStatus = useConnectorsStore(
    (s) => s.updateConnectorStatus,
  );
  const [showConfig, setShowConfig] = useState(false);
  const [editing, setEditing] = useState<Connector | null>(null);

  const handleToggle = (connector: Connector) => {
    const newStatus =
      connector.status === "connected" ? "disconnected" : "connected";
    updateConnectorStatus(connector.id, newStatus);
  };

  return (
    <div className="flex-1 flex h-full bg-surface-deep overflow-y-auto">
      <div className="max-w-container-max mx-auto w-full px-gutter py-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-headline-md font-headline-md text-on-surface">
            Connectors
          </h1>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={() => {
              setEditing(null);
              setShowConfig(true);
            }}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nuevo conector
          </button>
        </div>

        {showConfig && (
          <ConnectorConfig
            connector={editing}
            onClose={() => setShowConfig(false)}
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {connectors.map((connector) => (
            <div
              key={connector.id}
              className="bg-surface-container rounded-xl border border-border-subtle p-5"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-lg bg-primary-container/20 flex items-center justify-center">
                    <span className="material-symbols-outlined text-primary-container text-[18px]">
                      power
                    </span>
                  </div>
                  <div>
                    <h3 className="text-body-lg font-medium text-on-surface">
                      {connector.name}
                    </h3>
                    <span className="text-label-caps text-text-muted uppercase">
                      {connector.type === "mcp" ? "MCP" : "Custom"}
                    </span>
                  </div>
                </div>
                <div className="flex gap-1">
                  <button
                    className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
                    onClick={() => {
                      setEditing(connector);
                      setShowConfig(true);
                    }}
                  >
                    <span className="material-symbols-outlined text-[16px]">
                      edit
                    </span>
                  </button>
                  <button
                    className="p-1.5 rounded-lg text-text-muted hover:text-error hover:bg-surface-variant transition-colors"
                    onClick={() => removeConnector(connector.id)}
                  >
                    <span className="material-symbols-outlined text-[16px]">
                      delete
                    </span>
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span
                    className={`w-2 h-2 rounded-full ${
                      statusColors[connector.status]
                    }`}
                  />
                  <span className="text-label-caps text-text-muted">
                    {statusLabels[connector.status]}
                  </span>
                </div>
                <button
                  className={`px-3 py-1.5 rounded-lg text-body-md transition-colors ${
                    connector.status === "connected"
                      ? "bg-surface-variant text-text-muted hover:text-on-surface"
                      : "bg-primary-container text-on-primary hover:bg-primary-fixed-dim"
                  }`}
                  onClick={() => handleToggle(connector)}
                >
                  {connector.status === "connected"
                    ? "Desconectar"
                    : "Conectar"}
                </button>
              </div>
              <div className="mt-3 bg-surface-deep rounded-lg px-3 py-2 text-code-sm text-text-muted truncate">
                {connector.endpoint}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
