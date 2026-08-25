import { useState, useEffect } from "react";
import { api } from "../../services/api";
import { useUIStore } from "../../store/uiStore";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";

interface ModelGroup {
  provider: string;
  models: { id: string; name: string }[];
}

interface ModelSelectorProps {
  currentModel?: string;
  onSelect: (modelId: string, provider: string) => void;
  onClose: () => void;
}

export default function ModelSelector({ currentModel, onSelect, onClose }: ModelSelectorProps) {
  const [groups, setGroups] = useState<ModelGroup[]>([]);
  const [loading, setLoading] = useState(true);

  const handleChoose = (modelId: string, provider: string) => {
    // 1. Trigger parent callback
    onSelect(modelId, provider);

    // 2. Sync to chat store default & active chat
    const { activeChatId, updateChatProvider, setDefaultModel } = useChatStore.getState();
    setDefaultModel(modelId, provider);
    if (activeChatId) {
      updateChatProvider(activeChatId, provider, modelId);
    }

    // 3. Sync to backend API
    api.models.select({ provider, model: modelId }).catch(() => {});
    useToastStore.getState().show(`Modelo activo: ${modelId}`, "info");

    onClose();
  };

  useEffect(() => {
    api.models.list()
      .then((data) => {
        if (data && data.length > 0) {
          setGroups(data.map((g) => ({
            provider: g.provider,
            models: g.models.map((id) => ({ id, name: id })),
          })));
        } else {
          setGroups([]);
        }
      })
      .catch(() => setGroups([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="absolute bottom-full right-0 mb-2 w-80 bg-[#222222] border border-white/10 rounded-2xl shadow-2xl shadow-black/80 z-50 overflow-hidden font-sans select-none animate-slideUp">
      {/* Header */}
      <div className="p-3 border-b border-white/10 flex items-center justify-between bg-[#1e1e1e]">
        <span className="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">
          Seleccionar Modelo
        </span>
        <button
          className="text-zinc-500 hover:text-white transition-colors p-1 rounded"
          onClick={onClose}
        >
          <span className="material-symbols-outlined text-[16px]">close</span>
        </button>
      </div>

      <div className="p-2 max-h-80 overflow-y-auto scrollbar-thin">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-6 text-zinc-500 text-[12px] gap-2">
            <span className="material-symbols-outlined text-[20px] animate-spin text-[#d1f107]">sync</span>
            <span>Comprobando proveedores y modelos activos...</span>
          </div>
        ) : groups.length === 0 ? (
          /* Empty State: NO fake models, prompt to connect API */
          <div className="flex flex-col items-center justify-center p-5 text-center">
            <div className="w-11 h-11 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center text-zinc-400 mb-3 shadow-inner">
              <span className="material-symbols-outlined text-[22px]">cloud_off</span>
            </div>
            <span className="font-semibold text-white text-[13px] mb-1">
              Sin proveedores conectados
            </span>
            <p className="text-[11px] text-zinc-400 leading-relaxed mb-4 max-w-[240px]">
              No hay ninguna clave de API configurada ni host local activo. Conecta un proveedor para seleccionar sus modelos.
            </p>

            <button
              className="w-full py-2 px-3 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[12px] transition-all shadow-lg shadow-[#d1f107]/10 flex items-center justify-center gap-1.5"
              onClick={() => {
                useUIStore.getState().openSettings("proveedores");
                onClose();
              }}
            >
              <span className="material-symbols-outlined text-[16px]">key</span>
              <span>Configurar Proveedores LLM</span>
            </button>
          </div>
        ) : (
          /* Real Connected Groups & Models from Backend Fetch */
          <div className="flex flex-col gap-2">
            {groups.map((group) => (
              <div key={group.provider} className="flex flex-col">
                <div className="px-2.5 py-1 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider font-mono">
                  {group.provider}
                </div>
                <div className="flex flex-col gap-0.5">
                  {group.models.map((model) => (
                    <button
                      key={model.id}
                      className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-[12px] transition-colors text-left group ${
                        currentModel === model.id
                          ? "text-white bg-white/10 font-semibold"
                          : "text-zinc-300 hover:text-white hover:bg-white/5"
                      }`}
                      onClick={() => handleChoose(model.id, group.provider)}
                    >
                      <div className="flex items-center gap-2 truncate">
                        <span className="material-symbols-outlined text-[16px] text-zinc-500 group-hover:text-zinc-300 shrink-0">
                          psychology
                        </span>
                        <span className="truncate">{model.name}</span>
                      </div>
                      {currentModel === model.id && (
                        <span className="text-[#d1f107] material-symbols-outlined text-[16px] shrink-0 ml-1">
                          check
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </div>
            ))}

            <div className="px-3 py-2 mt-1 border-t border-white/5 flex items-center justify-between">
              <span className="text-[11px] text-zinc-400">¿Agregar más claves?</span>
              <button
                className="text-[11px] text-[#d1f107] hover:underline font-semibold"
                onClick={() => {
                  useUIStore.getState().openSettings("proveedores");
                  onClose();
                }}
              >
                Personalizar
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
