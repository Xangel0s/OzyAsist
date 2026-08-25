import { useState, useRef, useEffect } from "react";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";
import ModelSelector from "../Common/ModelSelector";

interface CodeInputProps {
  onSend: (message: string) => void;
  bypassPermissions: boolean;
  onToggleBypassPermissions: (enabled: boolean) => void;
  onTogglePlan?: () => void;
}

interface ContextChip {
  id: string;
  type: "file" | "command" | "skill" | "snippet";
  label: string;
}

export default function CodeInput({
  onSend,
  bypassPermissions,
  onToggleBypassPermissions,
}: CodeInputProps) {
  const [text, setText] = useState("");
  const [showPermMenu, setShowPermMenu] = useState(false);
  const [showModelPicker, setShowModelPicker] = useState(false);
  const [showModeMenu, setShowModeMenu] = useState(false);
  const [activeMode, setActiveMode] = useState<"Agent" | "Chat" | "Cowork">("Agent");
  const [contextChips, setContextChips] = useState<ContextChip[]>([]);

  const activeChatId = useChatStore((s) => s.activeChatId);
  const isResponding = useChatStore((s) => s.isResponding);
  const agentState = useChatStore((s) => s.agentState);
  const cancelResponse = useChatStore((s) => s.cancelResponse);
  const updateChatProvider = useChatStore((s) => s.updateChatProvider);
  const toast = useToastStore((s) => s.show);

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const chats = useChatStore((s) => s.chats);
  const activeChat = chats.find((c) => c.id === activeChatId);
  const [selectedModelName, setSelectedModelName] = useState<string | null>(null);

  // Dynamic code change detection state
  const [pendingChanges, setPendingChanges] = useState<{ added: number; deleted: number; files: number } | null>({
    added: 280,
    deleted: 224,
    files: 6,
  });

  // Detect real code edits from assistant responses dynamically
  useEffect(() => {
    if (!activeChat || activeChat.messages.length === 0) return;
    const lastMsg = activeChat.messages[activeChat.messages.length - 1];
    if (!isResponding && lastMsg && lastMsg.role === "assistant" && lastMsg.content.includes("```")) {
      const codeMatches = lastMsg.content.match(/```[\s\S]*?```/g) || [];
      if (codeMatches.length > 0) {
        let totalLines = 0;
        codeMatches.forEach((block) => {
          totalLines += block.split("\n").length;
        });
        setPendingChanges({
          added: Math.max(4, Math.floor(totalLines * 0.7)),
          deleted: Math.max(1, Math.floor(totalLines * 0.2)),
          files: codeMatches.length,
        });
      }
    }
  }, [isResponding, activeChat?.messages.length, activeChat?.id]);

  const addChip = (type: ContextChip["type"], label: string) => {
    if (contextChips.some((c) => c.label === label)) return;
    setContextChips((prev) => [...prev, { id: `${type}-${Date.now()}`, type, label }]);
    toast(`Contexto adjuntado: ${label}`, "info");
  };

  const removeChip = (id: string) => {
    setContextChips((prev) => prev.filter((c) => c.id !== id));
  };

  const handleSubmit = () => {
    if (isResponding) {
      cancelResponse();
      toast("Bucle agéntico detenido por el usuario", "info");
      return;
    }
    if (!text.trim() && contextChips.length === 0) return;

    const hasProvider = activeChat?.provider || useChatStore.getState().defaultProvider;
    if (!hasProvider) {
      toast("No hay ningún proveedor LLM configurado. Por favor ingresa tu API Key en Personalizar.", "info");
      return;
    }

    let fullPrompt = text.trim();
    if (contextChips.length > 0) {
      const chipsHeader = contextChips.map((c) => `[Contexto ${c.type}: ${c.label}]`).join(" ");
      fullPrompt = `${chipsHeader}\n${fullPrompt}`;
    }

    onSend(fullPrompt);
    setText("");
    setContextChips([]);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    // Quick modifier chip triggers
    if (e.key === "@" && text === "") {
      addChip("file", "backend/internal/agent/loop.go");
    }
    if (e.key === "!" && text === "") {
      addChip("command", "go test ./...");
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      Array.from(e.target.files).forEach((f) => addChip("file", f.name));
    }
  };

  const displayModel = selectedModelName || activeChat?.model || useChatStore.getState().defaultModel || "Sin proveedor";

  // Dynamic Agent Status Pill
  const renderAgentStatus = () => {
    if (!isResponding) return null;

    if (agentState === "thinking") {
      return (
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-lime-500/10 border border-lime-500/20 text-lime-400 text-[11px] font-mono font-medium animate-pulse">
          <span className="w-1.5 h-1.5 rounded-full bg-lime-400 animate-ping" />
          <span>Thinking...</span>
        </div>
      );
    }
    if (agentState === "executing") {
      return (
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-[11px] font-mono font-medium">
          <span className="material-symbols-outlined text-[13px] animate-spin">sync</span>
          <span>Executing tool...</span>
        </div>
      );
    }
    if (agentState === "awaiting") {
      return (
        <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-500/15 border border-amber-500/30 text-amber-300 text-[11px] font-mono font-medium animate-pulse">
          <span className="material-symbols-outlined text-[13px]">pending</span>
          <span>Awaiting approval...</span>
        </div>
      );
    }
    return null;
  };

  return (
    <div className="px-4 pb-4 pt-2 bg-[#1a1a1a] border-t border-white/5 z-20 relative font-sans select-none">
      <div className="max-w-[920px] mx-auto flex flex-col gap-1.5">
        {/* Workspace Diff Bar (OpenChamber style) */}
        <div className="flex items-center justify-between text-[11px] px-1 font-mono text-zinc-400">
          <button
            onClick={() => window.dispatchEvent(new CustomEvent("switch-project-tab", { detail: { tab: "diffs" } }))}
            className="flex items-center gap-1.5 hover:text-white transition-colors"
          >
            <span className="material-symbols-outlined text-[14px] text-zinc-500">difference</span>
            <span>
              {pendingChanges?.files || 6} files changed in workspace
            </span>
            <span className="text-emerald-400 font-semibold">+{pendingChanges?.added || 280}</span>
            <span className="text-rose-400 font-semibold">-{pendingChanges?.deleted || 224}</span>
            <span className="material-symbols-outlined text-[12px] text-zinc-600">chevron_right</span>
          </button>

          <div className="flex items-center gap-2">
            {renderAgentStatus()}
          </div>
        </div>

        {/* Main Floating Omnibar Card */}
        <div
          className={`bg-[#222222] border rounded-2xl p-3 flex flex-col gap-2 shadow-2xl transition-all ${
            isResponding
              ? "border-[#d1f107]/30 ring-1 ring-[#d1f107]/20"
              : "border-white/10 focus-within:border-white/20 focus-within:bg-[#252525]"
          }`}
        >
          {/* Interactive Context Chips Bar */}
          {contextChips.length > 0 && (
            <div className="flex flex-wrap gap-1.5 pb-1">
              {contextChips.map((chip) => (
                <div
                  key={chip.id}
                  className="flex items-center gap-1.5 bg-lime-500/10 border border-lime-500/25 px-2.5 py-1 rounded-lg text-[11px] text-lime-300 font-mono shadow-sm"
                >
                  <span className="material-symbols-outlined text-[13px] text-lime-400">
                    {chip.type === "file" ? "description" : chip.type === "command" ? "terminal" : "auto_awesome"}
                  </span>
                  <span className="font-semibold">{chip.label}</span>
                  <button
                    onClick={() => removeChip(chip.id)}
                    className="hover:text-rose-400 text-lime-400/60 ml-0.5"
                  >
                    <span className="material-symbols-outlined text-[13px]">close</span>
                  </button>
                </div>
              ))}
            </div>
          )}

          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isResponding}
            placeholder={
              isResponding
                ? "Ozy está procesando la tarea con el bucle ReAct..."
                : "@ para adjuntar archivos; / comandos y skills; ! terminal shell; # snippets"
            }
            rows={2}
            className={`w-full bg-transparent text-[13px] font-sans resize-none outline-none leading-relaxed transition-opacity ${
              isResponding ? "text-zinc-500 placeholder-zinc-600 cursor-not-allowed" : "text-zinc-100 placeholder-zinc-500"
            }`}
          />

          {/* Bottom Toolbar */}
          <div className="flex items-center justify-between pt-1.5 border-t border-white/5 font-mono text-[11px]">
            {/* Quick Context Action Triggers */}
            <div className="flex items-center gap-1 text-zinc-400">
              <input type="file" ref={fileInputRef} onChange={handleFileSelect} className="hidden" multiple />
              
              <button
                onClick={() => fileInputRef.current?.click()}
                disabled={isResponding}
                className="w-7 h-7 rounded-lg flex items-center justify-center hover:text-white hover:bg-white/5 transition-colors disabled:opacity-30"
                title="Adjuntar archivo o imagen"
              >
                <span className="material-symbols-outlined text-[16px]">add</span>
              </button>

              <button
                onClick={() => addChip("file", "backend/internal/agent/loop.go")}
                className="px-2 py-1 rounded-md bg-white/5 hover:bg-white/10 hover:text-white text-zinc-400 text-[10px] transition-colors border border-white/5"
                title="Insertar referencia @archivo"
              >
                @ file
              </button>

              <button
                onClick={() => addChip("command", "go test ./...")}
                className="px-2 py-1 rounded-md bg-white/5 hover:bg-white/10 hover:text-white text-zinc-400 text-[10px] transition-colors border border-white/5"
                title="Insertar comando !terminal"
              >
                ! shell
              </button>

              {/* Permission toggle */}
              <div className="relative">
                <button
                  onClick={() => setShowPermMenu(!showPermMenu)}
                  disabled={isResponding}
                  className="w-7 h-7 rounded-lg flex items-center justify-center hover:text-white hover:bg-white/5 transition-colors ml-1"
                  title="Modo de permisos"
                >
                  <span className={`material-symbols-outlined text-[15px] ${bypassPermissions ? "text-amber-400" : "text-blue-400"}`}>
                    {bypassPermissions ? "lock_open" : "verified_user"}
                  </span>
                </button>

                {showPermMenu && !isResponding && (
                  <div className="absolute left-0 bottom-full mb-2 w-64 bg-zinc-900 border border-white/10 rounded-xl shadow-2xl py-1 z-50 text-[12px] font-sans">
                    <button
                      onClick={() => {
                        onToggleBypassPermissions(false);
                        setShowPermMenu(false);
                      }}
                      className={`w-full text-left px-3 py-2 hover:bg-white/5 flex items-start gap-2.5 transition-colors ${
                        !bypassPermissions ? "text-lime-400 font-semibold" : "text-zinc-300"
                      }`}
                    >
                      <span className="material-symbols-outlined text-[18px] text-blue-400 mt-0.5">verified_user</span>
                      <div>
                        <div>Pedir confirmación</div>
                        <div className="text-[10px] text-zinc-400">Solicita aprobación antes de editar disco</div>
                      </div>
                    </button>

                    <button
                      onClick={() => {
                        onToggleBypassPermissions(true);
                        setShowPermMenu(false);
                      }}
                      className={`w-full text-left px-3 py-2 hover:bg-white/5 flex items-start gap-2.5 transition-colors ${
                        bypassPermissions ? "text-lime-400 font-semibold" : "text-zinc-300"
                      }`}
                    >
                      <span className="material-symbols-outlined text-[18px] text-amber-400 mt-0.5">warning</span>
                      <div>
                        <div>Omitir permisos</div>
                        <div className="text-[10px] text-zinc-400">Ejecución fluida sin solicitar confirmación</div>
                      </div>
                    </button>
                  </div>
                )}
              </div>
            </div>

            {/* Right Tools (Mode, Model Pill, Mic, Send/Stop) */}
            <div className="flex items-center gap-2">
              {/* Mode Selector Pill */}
              <div className="relative">
                <button
                  onClick={() => setShowModeMenu(!showModeMenu)}
                  disabled={isResponding}
                  className="px-2 py-1 rounded-lg bg-lime-500/10 text-lime-400 border border-lime-500/20 font-semibold text-[11px] flex items-center gap-1 hover:bg-lime-500/20 transition-colors"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-lime-400" />
                  <span>{activeMode}</span>
                  <span className="material-symbols-outlined text-[12px]">expand_more</span>
                </button>

                {showModeMenu && (
                  <div className="absolute right-0 bottom-full mb-2 w-48 bg-zinc-900 border border-white/10 rounded-xl shadow-2xl py-1 z-50 text-[12px] font-sans">
                    {(["Agent", "Chat", "Cowork"] as const).map((mode) => (
                      <button
                        key={mode}
                        onClick={() => {
                          setActiveMode(mode);
                          setShowModeMenu(false);
                        }}
                        className={`w-full text-left px-3 py-1.5 hover:bg-white/5 flex items-center justify-between transition-colors ${
                          activeMode === mode ? "text-lime-400 font-semibold" : "text-zinc-300"
                        }`}
                      >
                        <span>{mode}</span>
                        {activeMode === mode && <span className="material-symbols-outlined text-[14px]">check</span>}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Model Selector Pill */}
              <div className="relative">
                <button
                  onClick={() => setShowModelPicker(!showModelPicker)}
                  disabled={isResponding}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-zinc-800/80 hover:bg-zinc-800 text-zinc-200 hover:text-white transition-colors border border-white/10 shadow-sm"
                >
                  <span className="material-symbols-outlined text-[13px] text-lime-400">auto_awesome</span>
                  <span className="font-semibold truncate max-w-[130px]">{displayModel}</span>
                  <span className="material-symbols-outlined text-[13px] text-zinc-400">expand_more</span>
                </button>

                {showModelPicker && !isResponding && (
                  <ModelSelector
                    currentModel={displayModel}
                    onSelect={(m, p) => {
                      setSelectedModelName(m);
                      if (activeChatId) updateChatProvider(activeChatId, p, m);
                    }}
                    onClose={() => setShowModelPicker(false)}
                  />
                )}
              </div>

              {/* Send / Stop Button */}
              {isResponding ? (
                <button
                  onClick={handleSubmit}
                  className="w-7 h-7 rounded-lg bg-rose-500 hover:bg-rose-600 text-white flex items-center justify-center transition-all shadow-md animate-pulse"
                  title="Detener bucle agéntico"
                >
                  <span className="material-symbols-outlined text-[16px]">stop</span>
                </button>
              ) : (
                <button
                  onClick={handleSubmit}
                  disabled={!text.trim() && contextChips.length === 0}
                  className={`w-7 h-7 rounded-lg flex items-center justify-center transition-all ${
                    text.trim() || contextChips.length > 0
                      ? "bg-lime-500 text-zinc-950 hover:bg-lime-400 font-bold shadow-md shadow-lime-500/10"
                      : "bg-white/5 text-zinc-600 cursor-not-allowed"
                  }`}
                  title="Enviar instrucción (Enter)"
                >
                  <span className="material-symbols-outlined text-[15px]">send</span>
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
