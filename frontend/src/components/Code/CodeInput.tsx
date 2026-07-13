import { useState } from "react";
import { useChatStore } from "../../store/chatStore";

interface CodeInputProps {
  onSend: (message: string) => void;
}

export default function CodeInput({ onSend }: CodeInputProps) {
  const [value, setValue] = useState("");
  const activeChatId = useChatStore((s) => s.activeChatId);
  const coworkMode = useChatStore((s) => s.coworkMode);
  const setCoworkMode = useChatStore((s) => s.setCoworkMode);

  const handleSubmit = () => {
    if (value.trim() && activeChatId) {
      onSend(value.trim());
      setValue("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="px-margin-page pb-6 pt-2 bg-gradient-to-t from-surface-deep via-surface-deep to-transparent z-10">
      <div className="max-w-content-max-width mx-auto">
        <div className="bg-surface-elevated rounded-xl border border-border-subtle shadow-lg flex flex-col focus-within:border-outline focus-within:ring-1 focus-within:ring-outline transition-all">
          <textarea
            className="w-full bg-transparent border-none text-on-surface font-body-lg text-body-lg p-4 resize-none focus:ring-0 placeholder:text-text-muted min-h-[90px] outline-none"
            placeholder={coworkMode ? "¿Qué tarea querés que realice?" : "Responder a Ozy..."}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={3}
          />
          <div className="flex items-center justify-between p-3 border-t border-border-subtle/50">
            <div className="flex items-center gap-2">
              <button
                className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors flex items-center justify-center"
                title="Adjuntar archivo"
              >
                <span className="material-symbols-outlined text-[20px]">
                  add
                </span>
              </button>
              <div className="flex items-center bg-surface-variant rounded-md overflow-hidden">
                <button
                  className={`px-3 py-1.5 text-body-md font-body-md flex items-center gap-1.5 transition-colors ${
                    !coworkMode
                      ? "text-on-surface bg-surface-bright"
                      : "text-text-muted hover:text-on-surface hover:bg-surface-bright"
                  }`}
                  onClick={() => setCoworkMode(false)}
                >
                  <span className="material-symbols-outlined text-[16px]">
                    code
                  </span>
                  Code
                </button>
                <button
                  className={`px-3 py-1.5 text-body-md font-body-md transition-colors ${
                    coworkMode
                      ? "text-on-surface bg-surface-bright"
                      : "text-text-muted hover:text-on-surface hover:bg-surface-bright"
                  }`}
                  onClick={() => setCoworkMode(true)}
                >
                  Cowork
                </button>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <button className="flex items-center gap-1 text-text-muted hover:text-on-surface transition-colors text-body-md font-body-md group">
                DeepSeek V4 Pro
                <span className="text-text-muted/70 group-hover:text-text-muted transition-colors">
                  Go
                </span>
                <span className="material-symbols-outlined text-[16px] ml-0.5">
                  expand_more
                </span>
              </button>
              <div className="w-px h-4 bg-border-subtle mx-1" />
              <button
                className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors flex items-center justify-center"
                aria-label="Micrófono"
              >
                <span className="material-symbols-outlined text-[20px]">
                  mic
                </span>
              </button>
              <button
                className="p-1.5 rounded-lg bg-primary-container text-on-primary-fixed hover:bg-primary-fixed transition-colors flex items-center justify-center ml-1"
                onClick={handleSubmit}
                aria-label="Enviar"
              >
                <span className="material-symbols-outlined text-[20px] fill">
                  arrow_upward
                </span>
              </button>
            </div>
          </div>
        </div>
        <div className="text-center mt-3 text-label-caps text-text-muted opacity-60">
          Ozy puede cometer errores. Considera verificar el código crítico.
        </div>
      </div>
    </div>
  );
}
