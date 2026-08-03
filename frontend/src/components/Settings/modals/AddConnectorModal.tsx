import { useState } from "react";
import { useConnectorsStore } from "../../../store/connectorsStore";
import { useToastStore } from "../../../store/toastStore";

interface AddConnectorModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function AddConnectorModal({ isOpen, onClose }: AddConnectorModalProps) {
  const addConnector = useConnectorsStore((s) => s.addConnector);
  const toast = useToastStore((s) => s.show);

  const [connName, setConnName] = useState("");
  const [connTransport, setConnTransport] = useState<"http" | "stdio">("http");
  const [connUrl, setConnUrl] = useState("");
  const [connCmd, setConnCmd] = useState("");
  const [connArgs, setConnArgs] = useState("");
  const [connEnv, setConnEnv] = useState("");
  const [connOauthId, setConnOauthId] = useState("");
  const [connOauthSecret, setConnOauthSecret] = useState("");
  const [showAdvancedConn, setShowAdvancedConn] = useState(false);

  if (!isOpen) return null;

  const handleSave = async () => {
    const nameToSave = connName.trim() || "mcp-connector-" + Date.now().toString().slice(-4);
    let endpointToSave = "";
    if (connTransport === "stdio") {
      const cmdStr = connCmd.trim() || "npx";
      const argsStr = connArgs.trim();
      endpointToSave = argsStr ? `${cmdStr} ${argsStr}` : cmdStr;
    } else {
      endpointToSave = connUrl.trim() || "http://localhost:9000/mcp";
    }

    const id = await addConnector({
      id: "",
      name: nameToSave,
      type: "mcp",
      endpoint: endpointToSave,
      authConfig: { clientId: connOauthId, secret: connOauthSecret, env: connEnv },
      status: "connected",
    });

    if (id) {
      toast(`Conector MCP "${nameToSave}" (${connTransport.toUpperCase()}) agregado exitosamente`, "success");
      setConnName("");
      setConnUrl("");
      setConnCmd("");
      setConnArgs("");
      setConnEnv("");
      setConnOauthId("");
      setConnOauthSecret("");
      onClose();
    } else {
      toast("Error al agregar el conector", "error");
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h3 className="text-[18px] font-semibold">Agregar conector personalizado</h3>
          <button
            className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined text-[20px]">close</span>
          </button>
        </div>

        <div className="text-[13px] text-white/60 leading-relaxed">
          Conecta Ozy a tus datos y herramientas mediante servidores MCP remotos (HTTP/SSE) o comandos locales (STDIO).
        </div>

        {/* Transport Mode Switcher */}
        <div className="flex items-center gap-2 bg-[#1c1c1c] p-1 rounded-xl border border-white/10 text-[13px]">
          <button
            className={`flex-1 py-1.5 rounded-lg font-medium transition-all ${
              connTransport === "http" ? "bg-white/15 text-white shadow-sm" : "text-white/50 hover:text-white"
            }`}
            onClick={() => setConnTransport("http")}
          >
            HTTP / SSE
          </button>
          <button
            className={`flex-1 py-1.5 rounded-lg font-medium transition-all ${
              connTransport === "stdio" ? "bg-white/15 text-white shadow-sm" : "text-white/50 hover:text-white"
            }`}
            onClick={() => setConnTransport("stdio")}
          >
            STDIO (Comando Local)
          </button>
        </div>

        <div className="flex flex-col gap-3">
          <input
            type="text"
            placeholder="Nombre del conector (ej. sqlite-mcp)"
            value={connName}
            onChange={(e) => setConnName(e.target.value)}
            className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all"
          />

          {connTransport === "http" ? (
            <input
              type="text"
              placeholder="URL del servidor MCP remoto (ej. http://localhost:8000/sse)"
              value={connUrl}
              onChange={(e) => setConnUrl(e.target.value)}
              className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all font-mono"
            />
          ) : (
            <>
              <input
                type="text"
                placeholder="Comando ejecutable (ej. npx, python, node, uvx)"
                value={connCmd}
                onChange={(e) => setConnCmd(e.target.value)}
                className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all font-mono"
              />
              <input
                type="text"
                placeholder="Argumentos (ej. -y @modelcontextprotocol/server-sqlite C:/db.sqlite)"
                value={connArgs}
                onChange={(e) => setConnArgs(e.target.value)}
                className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all font-mono"
              />
            </>
          )}

          {/* Collapsible Advanced Config */}
          <div className="flex flex-col gap-2 pt-1">
            <button
              className="flex items-center gap-1 text-[13px] font-medium text-white/70 hover:text-white"
              onClick={() => setShowAdvancedConn(!showAdvancedConn)}
            >
              <span className="material-symbols-outlined text-[16px]">
                {showAdvancedConn ? "expand_less" : "expand_more"}
              </span>
              <span>Configuración avanzada</span>
            </button>
            {showAdvancedConn && (
              <div className="flex flex-col gap-2.5 pl-2 pt-1">
                {connTransport === "http" ? (
                  <>
                    <input
                      type="text"
                      placeholder="OAuth Client ID (opcional)"
                      value={connOauthId}
                      onChange={(e) => setConnOauthId(e.target.value)}
                      className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30"
                    />
                    <input
                      type="password"
                      placeholder="Secreto del cliente OAuth (opcional)"
                      value={connOauthSecret}
                      onChange={(e) => setConnOauthSecret(e.target.value)}
                      className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30"
                    />
                  </>
                ) : (
                  <input
                    type="text"
                    placeholder="Variables de entorno (ej. API_KEY=secret DB_PATH=/var/db)"
                    value={connEnv}
                    onChange={(e) => setConnEnv(e.target.value)}
                    className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 font-mono"
                  />
                )}
              </div>
            )}
          </div>
        </div>

        <div className="text-[11px] text-white/40 leading-relaxed pt-1">
          Solo usa conectores de desarrolladores en los que confíes. Ozy no controla qué herramientas ponen a disposición los desarrolladores.
        </div>

        <div className="flex justify-end gap-2.5 pt-2">
          <button
            className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-xl transition-colors"
            onClick={onClose}
          >
            Cancelar
          </button>
          <button
            className="px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-colors"
            onClick={handleSave}
          >
            Agregar
          </button>
        </div>
      </div>
    </div>
  );
}
