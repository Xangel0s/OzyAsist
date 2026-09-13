import { useState, useCallback } from "react";
import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import type { Message } from "../../store/chatStore";
import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import CodeArtifact from "../Code/CodeArtifact";
import ToolCallBlock from "../Code/ToolCallBlock";
import { OzyLogo } from "../Brand/OzyLogo";
import { Sparkles, ChevronDown, ChevronRight } from "lucide-react";

import MermaidDiagram from "../Common/MermaidDiagram";

interface ChatMessageProps {
  message: Message;
  chatId: string;
}

const markdownComponents: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || "");
    const codeStr = String(children).replace(/\n$/, "");
    if (match) {
      const lang = match[1].toLowerCase();
      if (lang === "mermaid") {
        return <MermaidDiagram code={codeStr} />;
      }
      const filename =
        lang === "go"
          ? "main.go"
          : lang === "ts"
          ? "index.ts"
          : lang === "tsx"
          ? "Component.tsx"
          : lang === "js"
          ? "script.js"
          : `file.${lang}`;
      return <CodeArtifact filename={filename} code={codeStr} />;
    }
    return (
      <code className="bg-[#d1f107]/10 text-[#d1f107] border border-[#d1f107]/20 px-1.5 py-0.5 rounded text-[12px] font-mono" {...props}>
        {children}
      </code>
    );
  },
  pre({ children }) {
    return <>{children}</>;
  },
  p({ children }) {
    return <p className="mb-3 leading-relaxed last:mb-0">{children}</p>;
  },
  h1({ children }) {
    return <h1 className="text-lg font-bold text-white mt-4 mb-2 first:mt-0">{children}</h1>;
  },
  h2({ children }) {
    return <h2 className="text-base font-bold text-white mt-4 mb-2 first:mt-0">{children}</h2>;
  },
  h3({ children }) {
    return <h3 className="text-sm font-bold text-neutral-100 mt-3 mb-1.5 first:mt-0">{children}</h3>;
  },
  ul({ children }) {
    return <ul className="list-disc list-outside mb-3 space-y-1.5 pl-4 text-neutral-200">{children}</ul>;
  },
  ol({ children }) {
    return <ol className="list-decimal list-outside mb-3 space-y-1.5 pl-4 text-neutral-200">{children}</ol>;
  },
  li({ children }) {
    return <li className="leading-relaxed">{children}</li>;
  },
  blockquote({ children }) {
    return <blockquote className="border-l-2 border-[#d1f107]/60 pl-3 my-2 text-neutral-300 italic">{children}</blockquote>;
  },
  hr() {
    return <hr className="border-white/10 my-4" />;
  },
};

function parseThinking(content: string): { thinking: string | null; response: string } {
  // 1. Extraer bloque de pensamiento (<think> o <thought>)
  let thinking: string | null = null;
  const thinkMatch = /<(think|thought)>([\s\S]*?)<\/\1>/i.exec(content);
  if (thinkMatch) {
    thinking = thinkMatch[2].trim();
  }

  // 2. Limpiar el contenido de ruido (XML, JSON de herramientas, etc.)
  let cleanResponse = content.replace(/<(think|thought)>[\s\S]*?<\/\1>/gi, "");
  
  // Limpiar etiquetas sueltas o sin cerrar al final
  cleanResponse = cleanResponse.replace(/(?:<(think|thought)>)+\s*$/gi, "");
  cleanResponse = cleanResponse.replace(/<(think|thought)>\s*/gi, "");
  cleanResponse = cleanResponse.replace(/<\/(think|thought)>\s*/gi, "");
  
  // Limpiar posibles fugas del JSON de Ollama (ej: `,{"name": "os_list_apps", "arguments": {}}`)
  cleanResponse = cleanResponse.replace(/^,?\s*\{"name":\s*"[^"]+",\s*"arguments":\s*\{[^}]*\}\}\s*/i, "");
  
  // Limpiar etiquetas <tool> o <tool_call> residuales
  cleanResponse = cleanResponse.replace(/<tool\s+name=[^>]+>[\s\S]*?<\/tool>/gi, "");
  cleanResponse = cleanResponse.replace(/<tool_call>[\s\S]*?<\/tool_call>/gi, "");
  
  // Limpiar etiquetas <HOLDER> y <context> (alucinaciones del modelo o inyecciones de prompt)
  cleanResponse = cleanResponse.replace(/<HOLDER>[\s\S]*?<\/HOLDER>/gi, "");
  cleanResponse = cleanResponse.replace(/<context>[\s\S]*?<\/context>/gi, "");
  cleanResponse = cleanResponse.replace(/<context>[\s\S]*/gi, ""); // si quedó sin cerrar

  // Limpiar posibles JSON crudos intermedios (ej: `\n{"name": "os_launch_app"...}`)
  cleanResponse = cleanResponse.replace(/\s*\{"name":\s*"[^"]+",\s*"arguments":\s*\{[^}]*\}\}\s*/gi, "");

  // Limpiar literal '\n' que Ollama a veces escupe crudo al principio
  cleanResponse = cleanResponse.replace(/^\\n\s*/, "");
  cleanResponse = cleanResponse.replace(/\\n/g, "\n");

  return { thinking, response: cleanResponse.trim() };
}

export default function ChatMessage({ message, chatId }: ChatMessageProps) {
  const isUser = message.role === "user";
  const respondApproval = useChatStore((s) => s.respondToolApproval);
  const activeChat = useChatStore((s) => s.chats.find((c) => c.id === chatId));
  const setFeedback = useChatStore((s) => s.setMessageFeedback);
  const regenerate = useChatStore((s) => s.regenerateMessage);
  const [copied, setCopied] = useState(false);
  const [thinkingExpanded, setThinkingExpanded] = useState(true);

  const { thinking, response } = parseThinking(message.content);
  const effectiveThinking = message.thoughtTrace || thinking;

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
      <div className="flex justify-end my-3 group">
        <div className="bg-[#222222] hover:bg-[#262626] text-neutral-100 rounded-[20px] px-4 py-2.5 max-w-[85%] text-[15px] leading-relaxed font-sans whitespace-pre-wrap border border-white/5 shadow-sm transition-colors">
          {message.content}
        </div>
      </div>
    );
  }

  const hasToolCalls = message.toolCalls && message.toolCalls.length > 0;
  const modelName = activeChat?.model || "Ozy Hybrid";
  const isLocalModel =
    modelName.toLowerCase().includes("lmstudio") ||
    modelName.toLowerCase().includes("ollama") ||
    modelName.toLowerCase().includes("local") ||
    modelName.toLowerCase().includes("ministral") ||
    modelName.toLowerCase().includes("qwen");

  const isFatalErrorOnly = response.startsWith("[Error:") && !response.includes("\n\n");

  const timeStr = message.timestamp
    ? new Date(message.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : "";

  return (
    <div className="flex justify-start my-4 group">
      <div className="flex gap-3.5 max-w-[95%] w-full">
        {/* Avatar Oficial de Ozy */}
        <div className="flex-shrink-0 pt-0.5">
          <OzyLogo size={26} isThinking={message.isThinking} />
        </div>

        <div className="flex flex-col gap-2.5 min-w-0 flex-1">
          {/* Bloque de Razonamiento Colapsable (Estilo Deduciendo / Thinking con Nine) */}
          {effectiveThinking && (
            <div className="rounded-xl border border-white/5 bg-black/25 p-2.5">
              <button
                type="button"
                onClick={() => setThinkingExpanded(!thinkingExpanded)}
                className="flex items-center gap-2 text-xs font-mono text-neutral-400 hover:text-neutral-200 transition-colors w-full text-left"
              >
                <Sparkles className="h-3.5 w-3.5 text-[#d1f107]" />
                <span className="font-semibold text-neutral-300">Razonando estrategia...</span>
                <span className="text-neutral-500 truncate flex-1 font-sans text-[11px]">
                  {effectiveThinking.slice(0, 80)}...
                </span>
                {thinkingExpanded ? (
                  <ChevronDown className="h-3.5 w-3.5 text-neutral-400" />
                ) : (
                  <ChevronRight className="h-3.5 w-3.5 text-neutral-400" />
                )}
              </button>
              {thinkingExpanded && (
                <div className="mt-2 pl-4 border-l border-white/10 font-mono text-xs leading-relaxed text-neutral-400 whitespace-pre-wrap">
                  {effectiveThinking}
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

          {/* Assistant Text Response */}
          {response && !isFatalErrorOnly ? (
            <div className="text-[15px] text-neutral-200 leading-relaxed font-sans prose-custom">
              <ReactMarkdown components={markdownComponents}>
                {response}
              </ReactMarkdown>
              {message.isStreaming && (
                <span className="inline-block h-4 w-1.5 ml-1 bg-[#d1f107] animate-pulse align-middle" />
              )}
            </div>
          ) : null}

          {/* Error Card if connection failed / unconfigured */}
          {isFatalErrorOnly ? (
            <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/25 flex flex-col gap-2.5 text-[12px] font-sans">
              <div className="flex items-start gap-2.5 text-amber-200">
                <span className="material-symbols-outlined text-[18px] text-amber-400 shrink-0 mt-0.5">warning</span>
                <div className="flex flex-col gap-0.5">
                  <span className="font-semibold text-amber-300">
                    {response.includes("Conexión perdida") ? "Conexión interrumpida con el modelo" : "Aviso de conexión"}
                  </span>
                  <span className="leading-relaxed text-neutral-300 text-[11px]">
                    {response.replace(/^\[Error:\s*|\s*\]$/g, "")}
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-2 pt-1 border-t border-amber-500/20">
                <button
                  type="button"
                  onClick={() => useUIStore.getState().openSettings("proveedores")}
                  className="px-3 py-1.5 rounded-lg bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[11px] transition-all shadow-md flex items-center gap-1.5"
                >
                  <span className="material-symbols-outlined text-[14px]">
                    {isLocalModel ? "dns" : "key"}
                  </span>
                  <span>
                    {isLocalModel
                      ? "Verificar Servidor Local en Personalizar (Host / URL)"
                      : "Configurar Proveedores en Personalizar"}
                  </span>
                </button>
              </div>
            </div>
          ) : null}

          {/* Assistant Metadata & Action Bar */}
          <div className="flex items-center justify-between text-[11px] font-mono text-neutral-500 pt-1 border-t border-white/5 select-none">
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5 text-neutral-400">
                <span className="font-semibold">{modelName}</span>
              </div>
              <span className="px-1.5 py-0.2 rounded bg-white/5 text-[10px] text-neutral-400 border border-white/5">
                core
              </span>
              {message.latencyMs ? (
                <span className="text-[#d1f107]/80 text-[10px] font-medium tracking-wide">
                  {(message.latencyMs / 1000).toFixed(1)}s
                </span>
              ) : message.isStreaming ? (
                <span className="text-neutral-500 text-[10px] flex items-center gap-1">
                  <span className="inline-block w-1 h-1 bg-[#d1f107] rounded-full animate-ping" />
                  generando...
                </span>
              ) : null}
              {timeStr && (
                <span className="text-neutral-600 text-[10px]">{timeStr}</span>
              )}
            </div>

            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                type="button"
                onClick={() => {
                  if ('speechSynthesis' in window) {
                    if (window.speechSynthesis.speaking) {
                      window.speechSynthesis.cancel();
                    } else {
                      const cleanText = response.replace(/```[\s\S]*?```/g, "Bloque de código omitido.").replace(/[#*_`]/g, "");
                      const utterance = new SpeechSynthesisUtterance(cleanText);
                      utterance.lang = "es-ES";
                      utterance.rate = 1.05;
                      window.speechSynthesis.speak(utterance);
                    }
                  }
                }}
                title="Leer en voz alta"
                className="p-1 hover:bg-white/10 rounded text-neutral-400 hover:text-[#d1f107] transition-colors"
              >
                <span className="material-symbols-outlined text-[15px]">volume_up</span>
              </button>

              <button
                type="button"
                onClick={handleCopy}
                title="Copiar respuesta"
                className="p-1 hover:bg-white/10 rounded text-neutral-400 hover:text-white transition-colors"
              >
                <span className="material-symbols-outlined text-[15px]">
                  {copied ? "check" : "content_copy"}
                </span>
              </button>

              <button
                type="button"
                onClick={() => setFeedback(chatId, message.id, message.feedback === "like" ? null : "like")}
                title="Buena respuesta"
                className={`p-1 hover:bg-white/10 rounded transition-colors ${
                  message.feedback === "like" ? "text-[#d1f107]" : "text-neutral-400 hover:text-white"
                }`}
              >
                <span className="material-symbols-outlined text-[15px]">thumb_up</span>
              </button>

              <button
                type="button"
                onClick={() => setFeedback(chatId, message.id, message.feedback === "dislike" ? null : "dislike")}
                title="Mala respuesta"
                className={`p-1 hover:bg-white/10 rounded transition-colors ${
                  message.feedback === "dislike" ? "text-rose-400" : "text-neutral-400 hover:text-white"
                }`}
              >
                <span className="material-symbols-outlined text-[15px]">thumb_down</span>
              </button>

              <button
                type="button"
                onClick={() => regenerate(chatId)}
                title="Regenerar respuesta"
                className="p-1 hover:bg-white/10 rounded text-neutral-400 hover:text-white transition-colors"
              >
                <span className="material-symbols-outlined text-[15px]">refresh</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
