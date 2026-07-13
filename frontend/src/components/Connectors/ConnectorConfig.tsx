import { useState } from "react";
import {
  useConnectorsStore,
  type Connector,
  type ConnectorType,
} from "../../store/connectorsStore";

interface ConnectorConfigProps {
  connector: Connector | null;
  onClose: () => void;
}

export default function ConnectorConfig({
  connector,
  onClose,
}: ConnectorConfigProps) {
  const addConnector = useConnectorsStore((s) => s.addConnector);
  const [name, setName] = useState(connector?.name || "");
  const [type, setType] = useState<ConnectorType>(connector?.type || "mcp");
  const [endpoint, setEndpoint] = useState(connector?.endpoint || "");
  const [token, setToken] = useState(connector?.authConfig?.token || "");

  const handleSave = () => {
    addConnector({
      id: connector?.id || Date.now().toString(),
      name,
      type,
      endpoint,
      authConfig: { token },
      status: "disconnected",
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-surface-elevated border border-border-subtle rounded-2xl w-full max-w-lg mx-4 overflow-hidden">
        <div className="flex items-center justify-between p-5 border-b border-border-subtle">
          <h2 className="text-headline-md font-headline-md text-on-surface">
            {connector ? "Editar conector" : "Nuevo conector"}
          </h2>
          <button
            className="text-text-muted hover:text-on-surface transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>
        <div className="p-5 flex flex-col gap-4">
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Nombre
            </label>
            <input
              className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ej: GitHub MCP"
            />
          </div>
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Tipo
            </label>
            <div className="flex gap-2">
              {(["mcp", "custom"] as const).map((t) => (
                <button
                  key={t}
                  className={`px-3 py-1.5 rounded-lg text-body-md transition-colors ${
                    type === t
                      ? "bg-primary-container text-on-primary"
                      : "bg-surface-container text-text-muted hover:text-on-surface border border-border-subtle"
                  }`}
                  onClick={() => setType(t)}
                >
                  {t === "mcp" ? "MCP" : "Custom"}
                </button>
              ))}
            </div>
          </div>
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Endpoint
            </label>
            <input
              className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder="http://localhost:8080/mcp/..."
            />
          </div>
          {type === "mcp" && (
            <div>
              <label className="text-label-caps text-text-muted mb-1.5 block">
                Token de autenticación
              </label>
              <input
                className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Bearer token..."
              />
            </div>
          )}
        </div>
        <div className="flex justify-end gap-3 p-5 border-t border-border-subtle">
          <button
            className="px-4 py-2 text-body-md text-text-muted hover:text-on-surface transition-colors"
            onClick={onClose}
          >
            Cancelar
          </button>
          <button
            className="px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={handleSave}
          >
            Guardar
          </button>
        </div>
      </div>
    </div>
  );
}
