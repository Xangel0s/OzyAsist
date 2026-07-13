import { useState, useCallback } from "react";
import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import type { Message } from "../../store/chatStore";
import { useChatStore } from "../../store/chatStore";
import CodeArtifact from "../Code/CodeArtifact";

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
      <code className="bg-white/8 px-1.5 py-0.5 rounded text-[13px] font-mono text-on-surface" {...props}>
        {children}
      </code>
    );
  },
  pre({ children }) {
    return <>{children}</>;
  },
};

function ActionBar({
  message,
  chatId,
}: {
  message: Message;
  chatId: string;
}) {
  const setFeedback = useChatStore((s) => s.setMessageFeedback);
  const regenerate = useChatStore((s) => s.regenerateMessage);
  const isResponding = useChatStore((s) => s.isResponding);
  const [copied, setCopied] = useState(false);

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

  const handleFeedback = (fb: "like" | "dislike") => {
    const next = message.feedback === fb ? null : fb;
    setFeedback(chatId, message.id, next);
  };

  return (
    <div className="flex items-center gap-1 mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
      <button
        className="p-1 rounded-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
        onClick={handleCopy}
        title="Copiar"
      >
        <span className="material-symbols-outlined text-[16px]">
          {copied ? "check" : "content_copy"}
        </span>
      </button>

      <button
        className={`p-1 rounded-md transition-colors ${
          message.feedback === "like"
            ? "text-green-500 bg-green-500/10"
            : "text-text-muted hover:text-green-500 hover:bg-surface-variant"
        }`}
        onClick={() => handleFeedback("like")}
        title="Me gusta"
      >
        <span className="material-symbols-outlined text-[16px]">
          thumb_up
        </span>
      </button>

      <button
        className={`p-1 rounded-md transition-colors ${
          message.feedback === "dislike"
            ? "text-red-500 bg-red-500/10"
            : "text-text-muted hover:text-red-500 hover:bg-surface-variant"
        }`}
        onClick={() => handleFeedback("dislike")}
        title="No me gusta"
      >
        <span className="material-symbols-outlined text-[16px]">
          thumb_down
        </span>
      </button>

      <button
        className="p-1 rounded-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors disabled:opacity-30"
        onClick={() => regenerate(chatId)}
        disabled={isResponding}
        title="Regenerar"
      >
        <span className="material-symbols-outlined text-[16px]">
          autorenew
        </span>
      </button>
    </div>
  );
}

export default function ChatMessage({ message, chatId }: ChatMessageProps) {
  const isUser = message.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div className="bg-chat-bubble-user text-on-surface rounded-2xl rounded-tr-sm px-5 py-3 max-w-[80%] text-body-lg whitespace-pre-wrap">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-start group">
      <div className="flex gap-4 max-w-[90%]">
        <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 mt-1 overflow-hidden">
          <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
        </div>
        <div className="flex flex-col gap-1 min-w-0">
          <div className="text-body-lg text-on-surface prose-custom">
            <ReactMarkdown components={markdownComponents}>
              {message.content}
            </ReactMarkdown>
          </div>
          <ActionBar message={message} chatId={chatId} />
        </div>
      </div>
    </div>
  );
}
