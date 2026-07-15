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
    const tokens: { text: string; className?: string }[] = [];
    let remaining = line;

    const patterns: [RegExp, string][] = [
      [/(\/\/.*)/, "token comment"],
      [/(".*?")/, "token string"],
      [/\b(package|import|func|if|else|for|return|nil|true|false|struct|interface|map|chan|go|defer|select|case|switch|type|var|const)\b/, "token keyword"],
      [/\b(func|main|New\w+)\b(?=\s*\()/, "token function"],
      [/(:=|==|!=|<=|>=|=|\+|-|\*|\/|&|\||\^|<|>)/, "token operator"],
    ];

    while (remaining.length > 0) {
      let earliestIdx = remaining.length;
      let earliestPattern: [RegExp, string] | null = null;
      let earliestMatch: RegExpExecArray | null = null;

      for (const [re, cls] of patterns) {
        re.lastIndex = 0;
        const m = re.exec(remaining);
        if (m && m.index < earliestIdx) {
          earliestIdx = m.index;
          earliestPattern = [re, cls];
          earliestMatch = m;
        }
      }

      if (earliestPattern && earliestMatch) {
        if (earliestMatch.index > 0) {
          tokens.push({ text: remaining.slice(0, earliestMatch.index) });
        }
        tokens.push({ text: earliestMatch[0], className: earliestPattern[1] });
        remaining = remaining.slice(earliestMatch.index + earliestMatch[0].length);
      } else {
        tokens.push({ text: remaining });
        remaining = "";
      }
    }

    return tokens;
  };

  const renderLine = (line: string, i: number) => {
    const tokens = highlight(line);
    return (
      <code key={i}>
        {tokens.map((t, j) =>
          t.className ? (
            <span key={j} className={t.className}>{t.text}</span>
          ) : (
            <span key={j}>{t.text}</span>
          ),
        )}
      </code>
    );
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
          {lines.map(renderLine)}
        </pre>
      </div>
    </div>
  );
}
