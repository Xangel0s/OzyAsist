import { useState, useRef } from "react";
import { useChatStore } from "../../store/chatStore";

interface ChatInputProps {
  onSend: (message: string) => void;
  placeholder?: string;
}

export default function ChatInput({
  onSend,
  placeholder,
}: ChatInputProps) {
  const [value, setValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const coworkMode = useChatStore((s) => s.coworkMode);
  const setCoworkMode = useChatStore((s) => s.setCoworkMode);

  const ph = placeholder ?? (coworkMode ? "¿Qué tarea querés que realice?" : "¿Cómo puedo ayudarte hoy?");

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (value.trim()) {
        onSend(value.trim());
        setValue("");
      }
    }
  };

  return (
    <div className="w-full bg-surface-elevated rounded-[1.25rem] border border-border-subtle p-4 flex flex-col shadow-lg shadow-black/20 focus-within:border-outline-variant transition-colors">
      <textarea
        ref={textareaRef}
        className="w-full bg-transparent border-none text-on-surface placeholder:text-text-muted resize-none focus:ring-0 text-body-lg font-body-lg min-h-[80px] outline-none"
        placeholder={ph}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={3}
      />
      <div className="flex items-center justify-between mt-4">
        <div className="flex items-center gap-2">
          <button
            className="w-8 h-8 rounded-full bg-surface-variant flex items-center justify-center text-on-surface hover:bg-surface-bright transition-colors border border-border-subtle"
            aria-label="Adjuntar archivo"
          >
            <span className="material-symbols-outlined text-[20px]">add</span>
          </button>
          <div className="flex bg-surface-variant rounded-full border border-border-subtle p-0.5">
            <button
              className={`px-3 py-1 rounded-full text-[12px] font-medium transition-colors ${
                !coworkMode
                  ? "bg-surface-container-highest text-on-surface"
                  : "text-text-muted hover:text-on-surface"
              }`}
              onClick={() => setCoworkMode(false)}
            >
              Chat
            </button>
            <button
              className={`px-3 py-1 rounded-full text-[12px] font-medium transition-colors ${
                coworkMode
                  ? "bg-surface-container-highest text-on-surface"
                  : "text-text-muted hover:text-on-surface"
              }`}
              onClick={() => setCoworkMode(true)}
            >
              Cowork
            </button>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button className="flex items-center gap-1 text-text-muted hover:text-on-surface text-[13px] font-medium transition-colors">
            <span className="text-on-surface">DeepSeek V4 Pro</span> Go{" "}
            <span className="material-symbols-outlined text-[16px]">
              expand_more
            </span>
          </button>
          <button
            className="text-text-muted hover:text-on-surface transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-variant"
            aria-label="Micrófono"
          >
            <span className="material-symbols-outlined text-[20px]">mic</span>
          </button>
          <button
            className="text-text-muted hover:text-on-surface transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-variant"
            aria-label="Eco"
          >
            <span className="material-symbols-outlined text-[20px]">
              graphic_eq
            </span>
          </button>
        </div>
      </div>
    </div>
  );
}
