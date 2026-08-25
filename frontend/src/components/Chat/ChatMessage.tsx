import { useState, useCallback } from "react";
import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import type { Message } from "../../store/chatStore";
import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import CodeArtifact from "../Code/CodeArtifact";
import ToolCallBlock from "../Code/ToolCallBlock";

interface ChatMessageProps {
  message: Message;
  chatId: string;
}

const markdownComponents: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || "");
    const codeStr = String(children).replace(/\n$/, "");
    if (match) {
      const lang = match[1];
      const filename = lang === "go" ? "main.go" : `file.${lang}`;
      return <CodeArtifact filename={filename} code={codeStr} />;
    }
    return (
      <code className="bg-lime-500/10 text-lime-400 border border-lime-500/20 px-1.5 py-0.5 rounded text-[12px] font-mono" {...props}>
        {children}
      </code>
    );
  },
  pre({ children }) {
    return <>{children}</>;
  },
};

function parseThinking(content: string): { thinking: string | null; response: string } {
  const match = /<think>([\s\S]*?)<\/think>/i.exec(content);
  if (match) {
    return {
      thinking: match[1].trim(),
      response: content.replace(/<think>[\s\S]*?<\/think>/i, "").trim(),
    };
  }
  return { thinking: null, response: content };
}

export default function ChatMessage({ message, chatId }: ChatMessageProps) {
  const isUser = message.role === "user";
  const respondApproval = useChatStore((s) => s.respondToolApproval);
  const activeChat = useChatStore((s) => s.chats.find((c) => c.id === chatId));
  const setFeedback = useChatStore((s) => s.setMessageFeedback);
  const regenerate = useChatStore((s) => s.regenerateMessage);
  const isResponding = useChatStore((s) => s.isResponding);
  const [copied, setCopied] = useState(false);
  const [thinkingExpanded, setThinkingExpanded] = useState(false);

  const { thinking, response } = parseThinking(message.content);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = message.content;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [message.content]);

  if (isUser) {
    return (
      <div className="flex justify-end my-2 group">
        <div className="bg-[#262626] hover:bg-[#2a2a2a] text-zinc-100 rounded-2xl rounded-tr-sm px-4 py-2.5 max-w-[80%] text-[13px] font-sans whitespace-pre-wrap border border-white/10 shadow-md leading-relaxed transition-colors">
          {message.content}
        </div>
      </div>
    );
  }

  const hasToolCalls = message.toolCalls && message.toolCalls.length > 0;
  const modelName = activeChat?.model || "Ozy Assistant";
  const isErrorMessage = response.startsWith("[Error:") || response.includes("no se pudo conectar");

  const timeStr = message.timestamp
    ? new Date(message.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : "";

  return (
    <div className="flex justify-start my-3 group">
      <div className="flex gap-3 max-w-[95%] w-full">
        {/* Avatar */}
        <div className="w-8 h-8 rounded-xl bg-zinc-900 border border-white/10 flex items-center justify-center flex-shrink-0 mt-0.5 overflow-hidden shadow-sm">
          <img src="/ozybaselogo.png" alt="Ozy" className="w-5 h-5 object-contain" />
        </div>

        <div className="flex flex-col gap-2 min-w-0 flex-1">
          {/* Thinking Block (OpenChamber style) */}
          {thinking && (
            <div className="rounded-xl border border-white/5 bg-zinc-900/60 p-2.5 text-xs font-mono">
              <button
                onClick={() => setThinkingExpanded(!thinkingExpanded)}
                className="flex items-center gap-1.5 text-zinc-400 hover:text-zinc-200 transition-colors w-full text-left"
              >
                <span className="material-symbols-outlined text-[15px] text-lime-400">psychology</span>
                <span className="font-semibold text-zinc-300">Thinking</span>
                <span className="text-zinc-500 truncate flex-1 font-sans">
                  {thinking.slice(0, 100)}...
                </span>
                <span className="material-symbols-outlined text-[14px]">
                  {thinkingExpanded ? "expand_less" : "expand_more"}
                </span>
              </button>
              {thinkingExpanded && (
                <div className="mt-2 pt-2 border-t border-white/5 text-zinc-400 leading-relaxed whitespace-pre-wrap font-mono text-[11px]">
                  {thinking}
                </div>
              )}
            </div>
          )}

          {/* Tool Calls inline blocks */}
          {hasToolCalls && (
            <div className="flex flex-col gap-1 my-1">
              {message.toolCalls!.map((tc) => (
                <ToolCallBlock
                  key={tc.toolId}
                  block={tc}
                  onApprove={(id) => respondApproval(id, true)}
                  onDeny={(id) => respondApproval(id, false)}
                />
              ))}
            </div>
          )}

          {/* Error Card if unconfigured */}
          {isErrorMessage ? (
            <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/25 flex flex-col gap-2.5 text-[12px] font-sans">
              <div className="flex items-start gap-2.5 text-amber-200">
                <span className="material-symbols-outlined text-[18px] text-amber-400 shrink-0 mt-0.5">warning</span>
                <span className="leading-relaxed">{response.replace(/^\[Error:\s*|\s*\]$/g, "")}</span>
              </div>
              <div className="flex items-center gap-2 pt-1 border-t border-amber-500/20">
                <button
                  onClick={() => useUIStore.getState().openSettings("proveedores")}
                  className="px-3 py-1.5 rounded-lg bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[11px] transition-all shadow-md flex items-center gap-1.5"
                >
                  <span className="material-symbols-outlined text-[14px]">key</span>
                  <span>Configurar API Key en Personalizar</span>
                </button>
              </div>
            </div>
          ) : response ? (
            /* Assistant Text Response */
            <div className="text-[13px] text-zinc-200 leading-relaxed font-sans prose-custom">
              <ReactMarkdown components={markdownComponents}>
                {response}
              </ReactMarkdown>
            </div>
          ) : null}

          {/* Assistant Metadata & Action Bar */}
          <div className="flex items-center justify-between text-[11px] font-mono text-zinc-500 pt-1 border-t border-white/5 select-none">
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5 text-zinc-400">
                <span className="font-semibold">{modelName}</span>
              </div>
              <span className="px-1.5 py-0.2 rounded bg-white/5 text-[10px] text-zinc-400 border border-white/5">
                build
              </span>
              {timeStr && (
                <span className="text-zinc-600 text-[10px]">{timeStr}</span>
              )}
            </div>

            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                className="p-1 rounded hover:text-white hover:bg-white/5 transition-colors"
                onClick={handleCopy}
                title="Copiar respuesta"
              >
                <span className="material-symbols-outlined text-[14px]">
                  {copied ? "check" : "content_copy"}
                </span>
              </button>

              <button
                className={`p-1 rounded transition-colors ${
                  message.feedback === "like"
                    ? "text-emerald-400 bg-emerald-500/10"
                    : "hover:text-emerald-400 hover:bg-white/5"
                }`}
                onClick={() => setFeedback(chatId, message.id, message.feedback === "like" ? null : "like")}
                title="Me gusta"
              >
                <span className="material-symbols-outlined text-[14px]">thumb_up</span>
              </button>

              <button
                className={`p-1 rounded transition-colors ${
                  message.feedback === "dislike"
                    ? "text-rose-400 bg-rose-500/10"
                    : "hover:text-rose-400 hover:bg-white/5"
                }`}
                onClick={() => setFeedback(chatId, message.id, message.feedback === "dislike" ? null : "dislike")}
                title="No me gusta"
              >
                <span className="material-symbols-outlined text-[14px]">thumb_down</span>
              </button>

              <button
                className="p-1 rounded hover:text-white hover:bg-white/5 transition-colors disabled:opacity-30"
                onClick={() => regenerate(chatId)}
                disabled={isResponding}
                title="Regenerar respuesta"
              >
                <span className="material-symbols-outlined text-[14px]">refresh</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
