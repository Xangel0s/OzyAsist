import { useState, useEffect } from "react";
import { api } from "../../services/api";

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

  useEffect(() => {
    api.models.list()
      .then((data) => {
        setGroups(data.map((g) => ({
          provider: g.provider,
          models: g.models.map((id) => ({ id, name: id })),
        })));
      })
      .catch(() => setGroups([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="absolute bottom-full right-0 mb-2 w-72 bg-[#2a2a2a] border border-white/10 rounded-xl shadow-2xl shadow-black/50 z-50 overflow-hidden">
      <div className="p-3 border-b border-white/10 flex items-center justify-between">
        <span className="text-[11px] font-medium text-white/40 uppercase tracking-wider">
          Seleccionar modelo
        </span>
        <button
          className="text-white/40 hover:text-white transition-colors"
          onClick={onClose}
        >
          <span className="material-symbols-outlined text-[16px]">close</span>
        </button>
      </div>
      <div className="p-2 max-h-80 overflow-y-auto">
        {loading && (
          <div className="px-3 py-4 text-center text-white/40 text-[13px]">
            Cargando modelos...
          </div>
        )}
        {!loading && groups.length === 0 && (
          <div className="px-3 py-4 text-center text-white/40 text-[13px]">
            No hay providers disponibles. Configura una API key.
          </div>
        )}
        {groups.map((group) => (
          <div key={group.provider} className="mb-2">
            <div className="px-3 py-1.5 text-[11px] font-medium text-white/40 uppercase tracking-wider">
              {group.provider}
            </div>
            {group.models.map((model) => (
              <button
                key={model.id}
                className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] transition-colors text-left ${
                  currentModel === model.id
                    ? "text-white bg-white/10"
                    : "text-white/60 hover:text-white hover:bg-white/5"
                }`}
                onClick={() => { onSelect(model.id, group.provider); onClose(); }}
              >
                <span className="material-symbols-outlined text-[16px] text-white/40">
                  smart_toy
                </span>
                <span className="flex-1">{model.name}</span>
                {currentModel === model.id && (
                  <span className="text-[#c8e64a] material-symbols-outlined text-[16px]">check</span>
                )}
              </button>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
