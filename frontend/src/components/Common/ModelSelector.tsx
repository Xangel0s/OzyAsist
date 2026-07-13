interface ModelOption {
  provider: string;
  models: { id: string; name: string; tier?: string }[];
}

const models: ModelOption[] = [
  {
    provider: "OpenCode Go",
    models: [
      { id: "deepseek-v4-pro", name: "DeepSeek V4 Pro", tier: "go" },
      { id: "deepseek-v4-flash", name: "DeepSeek V4 Flash", tier: "go" },
    ],
  },
  {
    provider: "Anthropic",
    models: [
      { id: "claude-sonnet-5", name: "Sonnet 5" },
      { id: "claude-opus-4", name: "Opus 4" },
      { id: "claude-haiku-3.5", name: "Haiku 3.5" },
    ],
  },
  {
    provider: "OpenAI",
    models: [
      { id: "gpt-4o", name: "GPT-4o" },
      { id: "gpt-4o-mini", name: "GPT-4o Mini" },
    ],
  },
  {
    provider: "OpenRouter",
    models: [
      { id: "mistral-3b", name: "Mistral 3B" },
      { id: "llama-3.1-8b", name: "Llama 3.1 8B" },
    ],
  },
  {
    provider: "Local (LM Studio)",
    models: [{ id: "local-model", name: "Modelo local" }],
  },
];

interface ModelSelectorProps {
  onClose: () => void;
}

export default function ModelSelector({ onClose }: ModelSelectorProps) {
  return (
    <div className="absolute bottom-full right-0 mb-2 w-72 bg-surface-elevated border border-border-subtle rounded-xl shadow-xl z-50 overflow-hidden">
      <div className="p-3 border-b border-border-subtle flex items-center justify-between">
        <span className="text-label-caps text-text-muted tracking-wider">
          Seleccionar modelo
        </span>
        <button
          className="text-text-muted hover:text-on-surface transition-colors"
          onClick={onClose}
        >
          <span className="material-symbols-outlined text-[16px]">close</span>
        </button>
      </div>
      <div className="p-2 max-h-80 overflow-y-auto">
        {models.map((group) => (
          <div key={group.provider} className="mb-2">
            <div className="px-3 py-1.5 text-label-caps text-text-muted tracking-wider">
              {group.provider}
            </div>
            {group.models.map((model) => (
              <button
                key={model.id}
                className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-left"
                onClick={onClose}
              >
                <span className="material-symbols-outlined text-[16px]">
                  smart_toy
                </span>
                <span className="flex-1">{model.name}</span>
                {model.tier === "go" && (
                  <span className="text-accent-lime text-label-caps">
                    GO
                  </span>
                )}
              </button>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
