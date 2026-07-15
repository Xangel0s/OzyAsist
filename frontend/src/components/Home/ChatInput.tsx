import { useState, useRef } from "react";
import { useChatStore } from "../../store/chatStore";
import ModelSelector from "../Common/ModelSelector";
import { useOnClickOutside } from "../../hooks";
import { api } from "../../services/api";
import { useToastStore } from "../../store/toastStore";

function shortModel(model?: string): string {
  if (!model) return "Sin modelo";
  const name = model.split("/").pop() ?? model;
  return name.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

interface ChatInputProps {
  onSend: (message: string) => void;
  placeholder?: string;
  modelOverride?: string;
  onModelChange?: (modelId: string, provider: string) => void;
  compact?: boolean;
}

export default function ChatInput({
  onSend,
  placeholder,
  modelOverride,
  onModelChange,
  compact = false,
}: ChatInputProps) {
  const [value, setValue] = useState("");
  const [showModelSelector, setShowModelSelector] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const selectorRef = useRef<HTMLDivElement>(null);
  const defaultModel = useChatStore((s) => s.defaultModel);

  useOnClickOutside(selectorRef, () => setShowModelSelector(false));

  const currentModel = modelOverride || defaultModel;
  const ph = placeholder ?? "¿Cómo puedo ayudarte hoy?";

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (value.trim()) {
        onSend(value.trim());
        setValue("");
      }
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await api.files.upload(file);
      useToastStore.getState().show(`Archivo subido: ${file.name}`, "success");
    } catch {
      useToastStore.getState().show("Error al subir archivo", "error");
    }
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleSend = () => {
    if (value.trim()) {
      onSend(value.trim());
      setValue("");
    }
  };

  const handleModelSelect = (modelId: string, provider: string) => {
    if (onModelChange) {
      onModelChange(modelId, provider);
    } else {
      useChatStore.setState({ defaultModel: modelId, defaultProvider: provider });
    }
    setShowModelSelector(false);
  };

  return (
    <div className={`w-full bg-[#1e1e1e] rounded-2xl border border-white/10 flex flex-col shadow-lg shadow-black/30 focus-within:border-white/15 transition-colors overflow-hidden ${compact ? "rounded-xl" : "rounded-2xl"}`}>
      <textarea
        ref={textareaRef}
        className={`w-full bg-transparent border-none text-white placeholder:text-white/30 resize-none focus:ring-0 outline-none px-4 ${compact ? "text-[14px] min-h-[60px] pt-4 pb-2" : "text-[15px] min-h-[100px] pt-5 pb-3 px-5"}`}
        placeholder={ph}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={compact ? 2 : 4}
      />

      <div className={`flex items-center justify-between ${compact ? "px-3 pb-3 pt-1" : "px-4 pb-4 pt-1"}`}>
        <div className="flex items-center gap-1">
          <input type="file" ref={fileInputRef} className="hidden" onChange={handleFileUpload} accept="image/*,.pdf,.txt,.md,.csv" />
          <button
            className={`${compact ? "w-8 h-8" : "w-9 h-9"} rounded-full flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors`}
            aria-label="Adjuntar archivo"
            onClick={() => fileInputRef.current?.click()}
          >
            <span className={`material-symbols-outlined ${compact ? "text-[20px]" : "text-[22px]"}`}>add</span>
          </button>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative" ref={selectorRef}>
            <button
              className="flex items-center gap-1.5 text-white/50 hover:text-white text-[13px] font-medium transition-colors px-3 py-1.5 rounded-lg hover:bg-white/5"
              onClick={() => setShowModelSelector(!showModelSelector)}
            >
              <span className="text-white">{shortModel(currentModel)}</span>
              <span className="text-white/30">Medio</span>
              <span className="material-symbols-outlined text-[16px]">expand_more</span>
            </button>

            {showModelSelector && (
              <ModelSelector
                currentModel={currentModel}
                onSelect={handleModelSelect}
                onClose={() => setShowModelSelector(false)}
              />
            )}
          </div>

          <button
            className={`${compact ? "w-8 h-8" : "w-9 h-9"} rounded-full bg-[#c8e64a] flex items-center justify-center text-[#1a1a1a] hover:bg-[#b8d63a] transition-colors disabled:opacity-30 disabled:cursor-not-allowed shadow-sm shadow-[#c8e64a]/20`}
            onClick={handleSend}
            disabled={!value.trim()}
            aria-label="Enviar mensaje"
          >
            <span className={`material-symbols-outlined ${compact ? "text-[18px]" : "text-[20px]"}`}>arrow_upward</span>
          </button>
        </div>
      </div>
    </div>
  );
}
