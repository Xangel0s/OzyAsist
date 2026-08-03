import { useState, useRef } from "react";
import { useChatStore } from "../../store/chatStore";
import { useProjectsStore } from "../../store/projectsStore";
import { useToastStore } from "../../store/toastStore";
import ModelSelector from "../Common/ModelSelector";

interface CodeInputProps {
  onSend: (message: string) => void;
  bypassPermissions: boolean;
  onToggleBypassPermissions: (enabled: boolean) => void;
  onTogglePlan?: () => void;
}

export default function CodeInput({
  onSend,
  bypassPermissions,
  onToggleBypassPermissions,
  onTogglePlan,
}: CodeInputProps) {
  const [text, setText] = useState("");
  const [showPermMenu, setShowPermMenu] = useState(false);
  const [showModelPicker, setShowModelPicker] = useState(false);
  const activeChatId = useChatStore((s) => s.activeChatId);
  const isResponding = useChatStore((s) => s.isResponding);
  const updateChatProvider = useChatStore((s) => s.updateChatProvider);
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const projects = useProjectsStore((s) => s.projects);
  const activeProject = projects.find((p) => p.id === activeProjectId);
  const toast = useToastStore((s) => s.show);

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [attachedFiles, setAttachedFiles] = useState<string[]>([]);

  const handleSubmit = () => {
    if (!text.trim() && attachedFiles.length === 0) return;
    if (isResponding) return;

    let fullPrompt = text.trim();
    if (attachedFiles.length > 0) {
      fullPrompt = `[Archivos adjuntos: ${attachedFiles.join(", ")}]\n${fullPrompt}`;
    }
    onSend(fullPrompt);
    setText("");
    setAttachedFiles([]);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleConfirmChanges = () => {
    toast("Cambios de código confirmados y sincronizados con Git", "success");
  };

  const handleOpenVSCode = () => {
    const cleanPath = projectPath.replace(/\\/g, "/");
    window.open(`vscode://file/${cleanPath}`, "_blank");
    toast(`Abriendo ${cleanPath} en VS Code...`, "info");
  };

  const handleCopyPath = () => {
    navigator.clipboard.writeText(projectPath);
    toast("Ruta del proyecto copiada al portapapeles", "success");
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const names = Array.from(e.target.files).map((f) => f.name);
      setAttachedFiles((prev) => [...prev, ...names]);
      toast(`Adjuntado: ${names.join(", ")}`, "success");
    }
  };

  const projectPath = activeProject?.rootPath || `C:\\Users\\User\\Documents\\ozyAsis`;

  return (
    <div className="px-4 pb-3 pt-2 bg-[#171717] border-t border-white/10 z-20 relative font-sans">
      <div className="max-w-[900px] mx-auto flex flex-col gap-2">
        {/* Top Control Bar of Input Container */}
        <div className="flex items-center justify-between text-[12px] px-1 font-mono text-white/60">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[16px] text-white/40">fork_right</span>
            <span className="text-white/40">master</span>
            <span className="text-white/20">←</span>
            <span className="text-white/80 font-medium">feat/ozy-assistant</span>
          </div>

          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 font-mono text-[12px]">
              <span className="text-emerald-400 font-semibold">+1423</span>
              <span className="text-rose-400 font-semibold">-18867</span>
            </div>

            <button
              onClick={handleConfirmChanges}
              className="px-3 py-1 rounded-lg border border-white/20 hover:border-white/40 bg-white/5 hover:bg-white/10 text-white font-medium text-[12px] transition-all flex items-center gap-1.5"
            >
              Confirmar cambios
            </button>
          </div>
        </div>

        {/* Main Card Surface */}
        <div className="bg-[#212121] border border-white/10 rounded-xl p-3 flex flex-col gap-2.5 shadow-lg focus-within:border-white/25 transition-all">
          {attachedFiles.length > 0 && (
            <div className="flex flex-wrap gap-1.5 pb-1">
              {attachedFiles.map((file, idx) => (
                <div key={idx} className="flex items-center gap-1.5 bg-white/10 px-2 py-1 rounded-md text-[11px] text-white/80 font-mono">
                  <span className="material-symbols-outlined text-[14px] text-[#d1f107]">description</span>
                  <span>{file}</span>
                  <button
                    onClick={() => setAttachedFiles((prev) => prev.filter((_, i) => i !== idx))}
                    className="hover:text-rose-400 text-white/40 ml-1"
                  >
                    <span className="material-symbols-outlined text-[14px]">close</span>
                  </button>
                </div>
              ))}
            </div>
          )}

          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Responder..."
            rows={2}
            className="w-full bg-transparent text-white placeholder-white/40 text-[14px] resize-none outline-none leading-relaxed"
          />

          {/* Bottom Bar Inside Input Box */}
          <div className="flex items-center justify-between pt-1 border-t border-white/5">
            <div className="flex items-center gap-2 relative">
              <input type="file" ref={fileInputRef} onChange={handleFileSelect} className="hidden" multiple />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="w-8 h-8 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
                title="Adjuntar archivo o imagen"
              >
                <span className="material-symbols-outlined text-[20px]">add</span>
              </button>

              {/* Permission Selector Dropdown */}
              <div className="relative">
                <button
                  onClick={() => setShowPermMenu(!showPermMenu)}
                  className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-medium transition-colors ${
                    bypassPermissions
                      ? "bg-amber-500/20 text-amber-300 border border-amber-500/30"
                      : "bg-white/5 text-white/60 hover:text-white"
                  }`}
                >
                  <span className="material-symbols-outlined text-[16px] text-amber-400">warning</span>
                  <span>{bypassPermissions ? "Omitir permisos" : "Pedir confirmación"}</span>
                  <span className="material-symbols-outlined text-[14px]">expand_more</span>
                </button>

                {showPermMenu && (
                  <div className="absolute left-0 bottom-full mb-2 w-64 bg-[#2a2a2a] border border-white/10 rounded-xl shadow-2xl py-1 z-50 text-[12px]">
                    <button
                      onClick={() => {
                        onToggleBypassPermissions(false);
                        setShowPermMenu(false);
                      }}
                      className={`w-full text-left px-3 py-2 hover:bg-white/10 flex items-start gap-2.5 transition-colors ${
                        !bypassPermissions ? "text-[#d1f107] font-semibold" : "text-white/80"
                      }`}
                    >
                      <span className="material-symbols-outlined text-[18px] text-blue-400 mt-0.5">verified_user</span>
                      <div>
                        <div>Pedir confirmación</div>
                        <div className="text-[10px] text-white/40">Solicita aprobación antes de editar disco</div>
                      </div>
                    </button>

                    <button
                      onClick={() => {
                        onToggleBypassPermissions(true);
                        setShowPermMenu(false);
                      }}
                      className={`w-full text-left px-3 py-2 hover:bg-white/10 flex items-start gap-2.5 transition-colors ${
                        bypassPermissions ? "text-[#d1f107] font-semibold" : "text-white/80"
                      }`}
                    >
                      <span className="material-symbols-outlined text-[18px] text-amber-400 mt-0.5">warning</span>
                      <div>
                        <div>Omitir permisos</div>
                        <div className="text-[10px] text-white/40">Ejecución fluida sin solicitar confirmación</div>
                      </div>
                    </button>
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-center gap-3">
              <div className="relative">
                <button
                  onClick={() => setShowModelPicker(!showModelPicker)}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-medium bg-white/5 hover:bg-white/10 text-white/70 hover:text-white transition-colors"
                >
                  <span className="material-symbols-outlined text-[15px] text-[#d1f107]">auto_awesome</span>
                  <span>Claude 3.7 (contexto 1M)</span>
                  <span className="material-symbols-outlined text-[14px]">expand_more</span>
                </button>

                {showModelPicker && (
                  <ModelSelector
                    onSelect={(m, p) => {
                      if (activeChatId) updateChatProvider(activeChatId, p, m);
                    }}
                    onClose={() => setShowModelPicker(false)}
                  />
                )}
              </div>

              <button
                onClick={handleSubmit}
                disabled={!text.trim() || isResponding}
                className={`w-8 h-8 rounded-full flex items-center justify-center transition-all ${
                  text.trim() && !isResponding
                    ? "bg-[#d1f107] text-black hover:bg-[#b8d63a] font-bold"
                    : "bg-white/10 text-white/20 cursor-not-allowed"
                }`}
              >
                <span className="material-symbols-outlined text-[18px]">
                  {isResponding ? "stop" : "arrow_upward"}
                </span>
              </button>
            </div>
          </div>
        </div>

        {/* Footer Bar Below Input Container */}
        <div className="flex items-center justify-between text-[11px] text-white/40 px-1 font-mono pt-0.5">
          <button
            onClick={handleCopyPath}
            className="flex items-center gap-1.5 truncate hover:text-white transition-colors cursor-pointer text-left"
            title="Hacer clic para copiar ruta al portapapeles"
          >
            <span className="material-symbols-outlined text-[14px]">folder</span>
            <span className="truncate">{projectPath}</span>
          </button>

          <div className="flex items-center gap-3 shrink-0">
            <button
              onClick={onTogglePlan}
              className="flex items-center gap-1 hover:text-white transition-colors"
            >
              <span className="material-symbols-outlined text-[15px]">list_alt</span>
              Plan
            </button>

            <button
              onClick={handleOpenVSCode}
              className="flex items-center gap-1 hover:text-white transition-colors"
              title="Abrir en editor externo"
            >
              <span className="material-symbols-outlined text-[15px]">code</span>
              VS Code
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
