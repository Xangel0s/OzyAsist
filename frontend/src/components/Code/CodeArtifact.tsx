interface CodeArtifactProps {
  filename: string;
  code: string;
  language?: string;
}

export default function CodeArtifact({ filename, code }: CodeArtifactProps) {
  const handleCopy = () => {
    navigator.clipboard.writeText(code);
  };

  const lines = code.split("\n");

  /* Simple syntax highlighting by keywords — matches the CSS token classes */
  const highlight = (line: string) => {
    let h = line
      .replace(
        /\b(package|import|func|if|else|for|return|nil|true|false|struct|interface|map|chan|go|defer|select|case|switch|type|var|const)\b/g,
        '<span class="token keyword">$1</span>',
      )
      .replace(
        /\b(func|main|New\w+)\b(?=\s*\()/g,
        '<span class="token function">$1</span>',
      )
      .replace(/"([^"]*)"/g, '<span class="token string">"$1"</span>')
      .replace(/\/\/.*/g, '<span class="token comment">$&</span>')
      .replace(
        /(\s)(:=|==|!=|<=|>=|=|\+|-|\*|\/|&|\||\^|<|>)(\s)/g,
        '$1<span class="token operator">$2</span>$3',
      );
    return h;
  };

  return (
    <div className="rounded-lg overflow-hidden border border-border-subtle bg-[#0D0D0D] flex flex-col">
      <div className="flex items-center justify-between px-4 py-2 bg-surface-container border-b border-border-subtle">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[16px] text-text-muted">
            code
          </span>
          <span className="text-label-caps text-text-muted tracking-wider">
            {filename}
          </span>
        </div>
        <button
          className="flex items-center gap-1.5 text-text-muted hover:text-on-surface transition-colors"
          onClick={handleCopy}
        >
          <span className="material-symbols-outlined text-[14px]">
            content_copy
          </span>
          <span className="text-label-caps">Copy code</span>
        </button>
      </div>
      <div className="p-4 overflow-x-auto text-code-sm font-code-sm leading-relaxed text-on-surface">
        <pre>
          {lines.map((line, i) => (
            <code key={i} dangerouslySetInnerHTML={{ __html: highlight(line) }} />
          ))}
        </pre>
      </div>
    </div>
  );
}
