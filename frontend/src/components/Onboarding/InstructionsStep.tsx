import { useState } from "react";
import { useMemoryStore } from "../../store/memoryStore";
import { useAuthStore } from "../../store/authStore";

interface InstructionsStepProps {
  onComplete: () => void;
  onBack: () => void;
}

export default function InstructionsStep({ onComplete, onBack }: InstructionsStepProps) {
  const [rules, setRules] = useState("");
  const addEntry = useMemoryStore((s) => s.addEntry);
  const updateProfileMd = useAuthStore((s) => s.updateProfileMd);
  const profileMd = useAuthStore((s) => s.user?.profileMd);

  const handleSave = () => {
    if (rules.trim()) {
      addEntry({
        topic: "Instrucciones",
        content: rules.trim(),
        source: "manual",
      });
      const existing = profileMd || "";
      updateProfileMd(existing + (existing ? "\n\n" : "") + "# Instrucciones\n" + rules.trim());
    }
    onComplete();
  };

  return (
    <div className="max-w-lg mx-auto w-full">
      <h2 className="text-body-lg font-medium text-on-surface mb-2 text-center">
        Instrucciones para Ozy
      </h2>
      <p className="text-text-muted text-body-md mb-8 text-center">
        Define reglas y preferencias que Ozy debe tomar en cuenta al responder.
      </p>

      <div className="bg-surface-elevated rounded-xl border border-border-subtle p-4">
        <textarea
          className="w-full bg-transparent border-none text-on-surface placeholder:text-text-muted resize-none focus:ring-0 outline-none text-body-lg min-h-[160px]"
          placeholder={
            "Ej:\n- Responder siempre en español\n- Usar 2 espacios de indentación\n- Preferir clean architecture en Go\n- Escribir tests antes de implementar\n- Explicar decisiones técnicas con contexto"
          }
          value={rules}
          onChange={(e) => setRules(e.target.value)}
          rows={8}
        />
      </div>

      <p className="text-text-muted text-label-caps mt-3 text-center">
        Estas instrucciones se guardarán y Ozy las tomará como referencia.
      </p>

      <div className="flex gap-3 mt-6">
        <button
          className="flex-1 px-4 py-2.5 border border-border-subtle rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
          onClick={onBack}
        >
          Volver
        </button>
        <button
          className="flex-1 px-4 py-2.5 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
          onClick={handleSave}
        >
          Guardar y finalizar
        </button>
      </div>
    </div>
  );
}
