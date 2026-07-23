import { useState, useRef } from "react";
import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import ModelSelector from "../Common/ModelSelector";
import { useOnClickOutside } from "../../hooks";
import { api } from "../../services/api";
import { useToastStore } from "../../store/toastStore";

function shortModel(model?: string): string {
  if (!model || model === "openrouter/auto") return "Sin proveedor";
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
  const [showAddMenu, setShowAddMenu] = useState(false);
  const [activeSubmenu, setActiveSubmenu] = useState<"proyecto" | "habilidades" | "conectores" | "plugins" | null>(null);
  const [activePluginProfile, setActivePluginProfile] = useState<string | null>(null);
  const [webSearchActive, setWebSearchActive] = useState(true);
  const [selectedSkillPill, setSelectedSkillPill] = useState<string | null>(null);
  const [modePill, setModePill] = useState<"Chat" | "Cowork">("Chat");

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const selectorRef = useRef<HTMLDivElement>(null);
  const addMenuRef = useRef<HTMLDivElement>(null);
  const defaultModel = useChatStore((s) => s.defaultModel);

  useOnClickOutside(selectorRef, () => setShowModelSelector(false));
  useOnClickOutside(addMenuRef, () => {
    setShowAddMenu(false);
    setActiveSubmenu(null);
    setActivePluginProfile(null);
  });

  const currentModel = modelOverride || defaultModel;
  const ph = placeholder ?? "Escribe / para habilidades";

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
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
    const trimmed = value.trim();
    if (trimmed || selectedSkillPill) {
      const fullMsg = selectedSkillPill ? `${selectedSkillPill} ${trimmed}`.trim() : trimmed;
      onSend(fullMsg);
      setValue("");
      setSelectedSkillPill(null);
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

  const handleSelectSkill = (skillName: string) => {
    setSelectedSkillPill(`/${skillName}`);
    setShowAddMenu(false);
    setActiveSubmenu(null);
    useToastStore.getState().show(`Habilidad activa: /${skillName}`, "info");
  };

  const openSettings = (category: string) => {
    useUIStore.getState().openSettings(category);
    setShowAddMenu(false);
    setActiveSubmenu(null);
  };

  const salesProfileSkills = [
    "account-research",
    "call-prep",
    "call-summary",
    "competitive-intelligence",
    "create-an-asset",
    "daily-briefing",
    "draft-outreach",
    "forecast",
    "pipeline-review",
  ];

  return (
    <div
      className={`w-full bg-[#1e1e1e] rounded-2xl border border-white/10 flex flex-col shadow-lg shadow-black/30 focus-within:border-white/15 transition-colors relative z-20 ${
        compact ? "rounded-xl" : "rounded-2xl"
      }`}
    >
      {selectedSkillPill && (
        <div className="px-4 pt-3 flex items-center gap-1">
          <span className="bg-[#1d3557] text-[#90e0ef] px-2.5 py-1 rounded-lg text-[13px] font-mono flex items-center gap-1.5 border border-[#457b9d]/30 shadow-sm">
            <span>{selectedSkillPill}</span>
            <button
              onClick={() => setSelectedSkillPill(null)}
              className="hover:text-white transition-colors"
              title="Quitar habilidad"
            >
              <span className="material-symbols-outlined text-[14px]">close</span>
            </button>
          </span>
        </div>
      )}

      <textarea
        ref={textareaRef}
        className={`w-full bg-transparent border-none text-white placeholder:text-white/30 resize-none focus:ring-0 outline-none px-4 ${
          compact ? "text-[14px] min-h-[60px] pt-4 pb-2" : "text-[15px] min-h-[100px] pt-5 pb-3 px-5"
        }`}
        placeholder={ph}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={compact ? 2 : 4}
      />

      <div className={`flex items-center justify-between ${compact ? "px-3 pb-3 pt-1" : "px-4 pb-4 pt-1"}`}>
        <div className="flex items-center gap-2">
          <input type="file" ref={fileInputRef} className="hidden" onChange={handleFileUpload} accept="image/*,.pdf,.txt,.md,.csv" />

          <div className="relative" ref={addMenuRef}>
            <button
              className={`${
                compact ? "w-8 h-8" : "w-9 h-9"
              } rounded-xl flex items-center justify-center text-white/60 hover:text-white bg-white/5 hover:bg-white/10 border border-white/5 transition-all`}
              aria-label="Acciones y adjuntos"
              onClick={() => setShowAddMenu(!showAddMenu)}
            >
              <span className={`material-symbols-outlined ${compact ? "text-[18px]" : "text-[20px]"}`}>add</span>
            </button>

            {showAddMenu && (
              <div
                className="absolute left-0 bottom-full mb-2 w-64 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50 text-[13px] text-white select-none"
                onMouseLeave={() => {
                  setActiveSubmenu(null);
                  setActivePluginProfile(null);
                }}
              >
                <button
                  className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                  onClick={() => {
                    fileInputRef.current?.click();
                    setShowAddMenu(false);
                  }}
                >
                  <div className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-[18px] text-white/60">attach_file</span>
                    <span>Agregar archivos o fotos</span>
                  </div>
                  <span className="text-[11px] text-white/30 font-mono">Ctrl+U</span>
                </button>

                <div className="relative" onMouseEnter={() => setActiveSubmenu("proyecto")}>
                  <button className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] text-white/60">folder</span>
                      <span>Agregar al proyecto</span>
                    </div>
                    <span className="material-symbols-outlined text-[14px] text-white/40">chevron_right</span>
                  </button>
                  {activeSubmenu === "proyecto" && (
                    <div className="absolute left-full top-0 ml-1 w-56 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50">
                      {["OzyAsist (Actual)", "crm-camp", "coolify-vm"].map((p) => (
                        <button
                          key={p}
                          className="w-full px-3.5 py-2 hover:bg-white/10 transition-colors text-left truncate"
                          onClick={() => {
                            useToastStore.getState().show(`Proyecto vinculado: ${p}`, "info");
                            setShowAddMenu(false);
                          }}
                        >
                          {p}
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                <div className="h-px bg-white/10 my-1" />

                <div className="relative" onMouseEnter={() => setActiveSubmenu("habilidades")}>
                  <button className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] text-white/60">settings_suggest</span>
                      <span>Habilidades</span>
                    </div>
                    <span className="material-symbols-outlined text-[14px] text-white/40">chevron_right</span>
                  </button>
                  {activeSubmenu === "habilidades" && (
                    <div className="absolute left-full top-0 ml-1 w-64 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50">
                      {["mcp-builder", "morning", "skill-creator", "web-artifacts-builder"].map((sk) => (
                        <button
                          key={sk}
                          className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                          onClick={() => handleSelectSkill(sk)}
                        >
                          <span className="material-symbols-outlined text-[16px] text-white/40">settings_suggest</span>
                          <span className="font-mono text-[12px]">{sk}</span>
                        </button>
                      ))}
                      <div className="h-px bg-white/10 my-1" />
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => openSettings("habilidades")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">work</span>
                        <span>Administrar habilidades</span>
                      </button>
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => {
                          useUIStore.getState().setActiveView("skills");
                          setShowAddMenu(false);
                        }}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">add</span>
                        <span>Explorar habilidades</span>
                      </button>
                    </div>
                  )}
                </div>

                <div className="relative" onMouseEnter={() => setActiveSubmenu("conectores")}>
                  <button className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] text-white/60">power</span>
                      <span>Conectores</span>
                    </div>
                    <span className="material-symbols-outlined text-[14px] text-white/40">chevron_right</span>
                  </button>
                  {activeSubmenu === "conectores" && (
                    <div className="absolute left-full top-0 ml-1 w-64 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50">
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => openSettings("conectores")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">add</span>
                        <span>Agregar conector</span>
                      </button>
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => openSettings("conectores")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">work</span>
                        <span>Administrar conectores</span>
                      </button>
                      <div className="h-px bg-white/10 my-1" />
                      <div className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors">
                        <div className="flex items-center gap-2.5">
                          <span className="material-symbols-outlined text-[16px] text-white/40">radio_button_checked</span>
                          <span className="font-medium">opencode</span>
                        </div>
                        <span className="w-8 h-4.5 bg-[#3b82f6] rounded-full flex items-center justify-end px-0.5 shadow-sm">
                          <span className="w-3.5 h-3.5 bg-white rounded-full" />
                        </span>
                      </div>
                      <button className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left text-white/70">
                        <span className="material-symbols-outlined text-[16px] text-white/40">radio_button_unchecked</span>
                        <span>Agregar desde opencode</span>
                      </button>
                      <div className="h-px bg-white/10 my-1" />
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left text-white/70"
                        onClick={() => openSettings("conectores")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">search</span>
                        <span>Acceso a herramientas</span>
                      </button>
                    </div>
                  )}
                </div>

                <div className="relative" onMouseEnter={() => setActiveSubmenu("plugins")}>
                  <button className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left">
                    <div className="flex items-center gap-2.5">
                      <span className="material-symbols-outlined text-[18px] text-white/60">extension</span>
                      <span>Plugins</span>
                    </div>
                    <span className="material-symbols-outlined text-[14px] text-white/40">chevron_right</span>
                  </button>
                  {activeSubmenu === "plugins" && (
                    <div className="absolute left-full top-0 ml-1 w-56 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50">
                      {["Productivity", "Engineering", "Sales", "Design"].map((cat) => (
                        <div
                          key={cat}
                          className="relative"
                          onMouseEnter={() => setActivePluginProfile(cat)}
                        >
                          <button className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left">
                            <span>{cat}</span>
                            <span className="material-symbols-outlined text-[14px] text-white/40">chevron_right</span>
                          </button>
                          {activePluginProfile === cat && cat === "Sales" && (
                            <div className="absolute left-full top-0 ml-1 w-60 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50 max-h-72 overflow-y-auto">
                              {salesProfileSkills.map((sk) => (
                                <button
                                  key={sk}
                                  className="w-full flex items-center gap-2.5 px-3.5 py-1.5 hover:bg-white/10 transition-colors text-left"
                                  onClick={() => handleSelectSkill(sk)}
                                >
                                  <span className="material-symbols-outlined text-[14px] text-white/40">settings_suggest</span>
                                  <span className="font-mono text-[11px] truncate">{sk}</span>
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      ))}
                      <div className="h-px bg-white/10 my-1" />
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => openSettings("plugins")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">work</span>
                        <span>Administrar plugins</span>
                      </button>
                      <button
                        className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                        onClick={() => openSettings("plugins")}
                      >
                        <span className="material-symbols-outlined text-[16px] text-white/60">add</span>
                        <span>Explorar plugins</span>
                      </button>
                    </div>
                  )}
                </div>

                <div className="h-px bg-white/10 my-1" />

                <button
                  className="w-full flex items-center justify-between px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                  onClick={() => setWebSearchActive(!webSearchActive)}
                >
                  <div className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-[18px] text-white/60">language</span>
                    <span>Búsqueda web</span>
                  </div>
                  {webSearchActive && (
                    <span className="material-symbols-outlined text-[16px] text-[#3b82f6]">check</span>
                  )}
                </button>
              </div>
            )}
          </div>

          <div className="flex items-center gap-0.5 bg-white/5 p-1 rounded-xl text-[12px] text-white/50 border border-white/5">
            <button
              className={`px-2.5 py-1 rounded-lg transition-all ${
                modePill === "Chat" ? "bg-white/15 text-white font-medium shadow-sm" : "hover:text-white"
              }`}
              onClick={() => setModePill("Chat")}
            >
              Chat
            </button>
            <button
              className={`px-2.5 py-1 rounded-lg transition-all ${
                modePill === "Cowork" ? "bg-white/15 text-white font-medium shadow-sm" : "hover:text-white"
              }`}
              onClick={() => setModePill("Cowork")}
            >
              Cowork
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative" ref={selectorRef}>
            <button
              className="flex items-center gap-1.5 text-white/50 hover:text-white text-[13px] font-medium transition-colors px-3 py-1.5 rounded-lg hover:bg-white/5 border border-white/5"
              onClick={() => setShowModelSelector(!showModelSelector)}
            >
              <span className="text-white">{shortModel(currentModel)}</span>
              <span className="text-white/40 text-[11px] font-normal">
                {currentModel && currentModel !== "openrouter/auto" ? "Medio" : "Configurar"}
              </span>
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
