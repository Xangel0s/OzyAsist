import React, { useEffect, useState } from "react";
import { api, MCPConnectorDTO } from "../../services/api";

export const MCPConnectorsList: React.FC = () => {
  const [connectors, setConnectors] = useState<MCPConnectorDTO[]>([]);
  const [loading, setLoading] = useState(true);

  // Form state
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [env, setEnv] = useState("");

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.mcp.list();
      setConnectors(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.mcp.create({
        name,
        command,
        args: args.split(",").map((s) => s.trim()).filter(Boolean),
        env: env.split(",").map((s) => s.trim()).filter(Boolean),
      });
      setName("");
      setCommand("");
      setArgs("");
      setEnv("");
      load();
    } catch (err) {
      console.error(err);
      alert("Error agregando el conector MCP");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar este conector MCP?")) return;
    try {
      await api.mcp.delete(id);
      load();
    } catch (e) {
      console.error(e);
    }
  };

  const handleReconnect = async (id: string) => {
    try {
      await api.mcp.reconnect(id);
      load();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-bold text-[#d1f107]">Conectores MCP</h3>
        <p className="text-sm text-gray-400">
          Añade servidores Model Context Protocol externos (ej. Github, Postgres, SQLite) para que Ozy pueda interactuar con ellos de forma nativa a través de herramientas.
        </p>
      </div>

      <div className="space-y-3">
        {loading ? (
          <div className="text-sm text-gray-400">Cargando conectores...</div>
        ) : connectors.length === 0 ? (
          <div className="text-sm text-gray-400">No hay conectores MCP instalados.</div>
        ) : (
          connectors.map((c) => (
            <div
              key={c.id}
              className="p-3 bg-[#1e1e1e] border border-gray-700 rounded flex justify-between items-start"
            >
              <div>
                <div className="flex items-center space-x-2">
                  <h4 className="font-semibold text-gray-100">{c.name}</h4>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${
                      c.status === "connected"
                        ? "bg-green-900/30 text-green-400 border border-green-700/50"
                        : c.status === "error"
                        ? "bg-red-900/30 text-red-400 border border-red-700/50"
                        : "bg-gray-800 text-gray-400"
                    }`}
                  >
                    {c.status}
                  </span>
                </div>
                <div className="text-xs text-gray-400 font-mono mt-1">
                  $ {c.command} {c.args && JSON.parse(c.args).join(" ")}
                </div>
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={() => handleReconnect(c.id)}
                  className="px-2 py-1 bg-[#2c2c2c] hover:bg-[#333] text-gray-300 text-xs rounded transition-colors"
                >
                  Reconectar
                </button>
                <button
                  onClick={() => handleDelete(c.id)}
                  className="px-2 py-1 bg-red-900/20 hover:bg-red-900/40 text-red-400 text-xs rounded transition-colors"
                >
                  Borrar
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      <form onSubmit={handleAdd} className="bg-[#1e1e1e] p-4 rounded border border-gray-700 space-y-3">
        <h4 className="text-sm font-semibold text-gray-200">Añadir Nuevo Servidor MCP</h4>
        
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-400 mb-1">Nombre</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-[#131313] border border-gray-700 rounded px-3 py-1.5 text-sm focus:border-[#d1f107] focus:outline-none transition-colors"
              placeholder="Ej: SQLite MCP"
              required
            />
          </div>
          <div>
            <label className="block text-xs text-gray-400 mb-1">Comando Exec</label>
            <input
              type="text"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              className="w-full bg-[#131313] border border-gray-700 rounded px-3 py-1.5 text-sm focus:border-[#d1f107] focus:outline-none transition-colors"
              placeholder="Ej: npx"
              required
            />
          </div>
        </div>

        <div>
          <label className="block text-xs text-gray-400 mb-1">Argumentos (separados por coma)</label>
          <input
            type="text"
            value={args}
            onChange={(e) => setArgs(e.target.value)}
            className="w-full bg-[#131313] border border-gray-700 rounded px-3 py-1.5 text-sm focus:border-[#d1f107] focus:outline-none transition-colors font-mono"
            placeholder="Ej: -y, @modelcontextprotocol/server-sqlite, /path/to/db.sqlite"
          />
        </div>

        <div>
          <label className="block text-xs text-gray-400 mb-1">Variables de Entorno (separadas por coma)</label>
          <input
            type="text"
            value={env}
            onChange={(e) => setEnv(e.target.value)}
            className="w-full bg-[#131313] border border-gray-700 rounded px-3 py-1.5 text-sm focus:border-[#d1f107] focus:outline-none transition-colors font-mono"
            placeholder="Ej: GITHUB_TOKEN=xxx, KEY=value"
          />
        </div>

        <button
          type="submit"
          className="bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-semibold px-4 py-2 rounded text-sm transition-colors w-full"
        >
          Guardar y Conectar
        </button>
      </form>
    </div>
  );
};
