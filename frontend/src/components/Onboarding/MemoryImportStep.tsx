import { useState, useRef } from "react";
import { useMemoryStore } from "../../store/memoryStore";
import { useAuthStore } from "../../store/authStore";

interface MemoryImportStepProps {
  onNext: () => void;
}

export default function MemoryImportStep({ onNext }: MemoryImportStepProps) {
  const [file, setFile] = useState<File | null>(null);
  const [content, setContent] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);
  const [showHelper, setShowHelper] = useState(false);
  const importFromMd = useMemoryStore((s) => s.importFromMd);
  const entryCount = useMemoryStore((s) => s.entries.length);
  const updateProfileMd = useAuthStore((s) => s.updateProfileMd);

  const handleFile = (f: File) => {
    setFile(f);
    const reader = new FileReader();
    reader.onload = (e) => {
      setContent(e.target?.result as string);
    };
    reader.readAsText(f);
  };

  const handleContinue = () => {
    if (content) {
      importFromMd(content);
      updateProfileMd(content);
    }
    onNext();
  };

  return (
    <div className="max-w-lg mx-auto w-full">
      <h2 className="text-body-lg font-medium text-on-surface mb-2 text-center">
        Importa tu memoria
      </h2>
      <p className="text-text-muted text-body-md mb-4 text-center">
        Sube un archivo .md con tu perfil, instrucciones o memoria anterior
        para que Ozy te conozca mejor.
      </p>

      <div className="text-center mb-6">
        <button
          className="text-label-caps text-primary-container hover:underline"
          onClick={() => setShowHelper(!showHelper)}
        >
          {showHelper ? "Ocultar formato" : "¿Qué formato usa Claude?"}
        </button>
      </div>

      {showHelper && (
        <div className="bg-surface-container rounded-xl border border-border-subtle p-4 mb-6 text-code-sm text-text-muted">
          <p className="text-label-caps text-on-surface mb-2">Formato MEMORY.md de Claude:</p>
          <pre className="whitespace-pre-wrap leading-relaxed">
{`# MEMORY.md

## Arquitectura
- 2026-07-10: Migramos de REST a WebSocket para streaming — latencia reducida 40%
- 2026-07-08: Elegimos Qdrant sobre Pinecone por ser self-hosted

## Stack
- Frontend: React 19 + TypeScript + TailwindCSS
- Backend: Go 1.26 + Gin
- DB: MongoDB + Qdrant

## Reglas
- Usar 2 espacios de indentación en TypeScript
- Tests antes de implementar
- Preferir componentes funcionales

## Decisiones
- 2026-07-05: Adoptar clean architecture en Go
- 2026-07-03: Usar pnpm en vez de npm`}
          </pre>
        </div>
      )}

      <div
        className="border-2 border-dashed border-border-subtle rounded-2xl p-12 flex flex-col items-center justify-center cursor-pointer hover:border-primary-container/50 transition-colors bg-surface-container"
        onClick={() => fileRef.current?.click()}
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          const f = e.dataTransfer.files[0];
          if (f) handleFile(f);
        }}
      >
        <input
          ref={fileRef}
          type="file"
          accept=".md,.txt"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleFile(f);
          }}
        />
        <span className="material-symbols-outlined text-[48px] text-text-muted mb-4">
          upload_file
        </span>
        <span className="text-body-md text-text-muted">
          {file
            ? file.name
            : "Arrastra tu .md aquí o haz clic para seleccionar"}
        </span>
      </div>

      {content && (
        <div className="mt-6 bg-surface-container rounded-xl border border-border-subtle p-4 max-h-48 overflow-y-auto">
          <pre className="text-code-sm text-text-muted whitespace-pre-wrap">
            {content.slice(0, 500)}
            {content.length > 500 ? "..." : ""}
          </pre>
        </div>
      )}

      <button
        className="w-full mt-8 px-4 py-2.5 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium disabled:opacity-50 disabled:cursor-not-allowed"
        disabled={!content}
        onClick={handleContinue}
      >
        {content ? `Importar (${entryCount} secciones detectadas)` : "Continuar"}
      </button>
      <button
        className="w-full mt-2 px-4 py-2 text-text-muted hover:text-on-surface text-body-md transition-colors"
        onClick={onNext}
      >
        Omitir este paso
      </button>
    </div>
  );
}
