import { useState, useEffect } from "react";
import { useMemoryStore } from "../../store/memoryStore";
import { api } from "../../services/api";

interface FeedbackStepProps {
  onNext: () => void;
  onBack: () => void;
}

export default function FeedbackStep({ onNext, onBack }: FeedbackStepProps) {
  const [loading, setLoading] = useState(true);
  const [feedback, setFeedback] = useState("");
  const [error, setError] = useState("");
  const entries = useMemoryStore((s) => s.entries);

  useEffect(() => {
    const content = entries.length > 0
      ? entries.map((e) => `## ${e.topic}\n${e.content}`).join("\n\n")
      : "";

    if (!content) {
      setFeedback("No se encontró memoria importada. Volvé al paso anterior para subir tu archivo.");
      setLoading(false);
      return;
    }

    api.onboarding.analyzeMemory(content)
      .then((res) => {
        setFeedback(res.feedback);
        setLoading(false);
      })
      .catch((e) => {
        setError(e?.message || "Error al analizar la memoria");
        setLoading(false);
      });
  }, []);

  return (
    <div className="max-w-lg mx-auto w-full">
      <h2 className="text-body-lg font-medium text-on-surface mb-2 text-center">
        Feedback de la IA
      </h2>
      <p className="text-text-muted text-body-md mb-8 text-center">
        Ozy ha analizado tu memoria y tiene algunas sugerencias.
      </p>

      {loading ? (
        <div className="bg-surface-container rounded-xl border border-border-subtle p-6 flex items-center justify-center">
          <div className="flex items-center gap-3 text-text-muted">
            <span className="material-symbols-outlined text-[24px] animate-spin">
              progress_activity
            </span>
            <span>Analizando tu memoria...</span>
          </div>
        </div>
      ) : error ? (
        <div className="bg-surface-container rounded-xl border border-border-subtle p-6">
          <p className="text-text-muted text-body-md">{error}</p>
        </div>
      ) : (
        <div className="bg-surface-container rounded-xl border border-border-subtle p-6">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 overflow-hidden">
              <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
            </div>
            <div className="text-body-md text-on-surface whitespace-pre-wrap">
              {feedback}
            </div>
          </div>
        </div>
      )}

      <div className="flex gap-3 mt-8">
        <button
          className="flex-1 px-4 py-2.5 border border-border-subtle rounded-lg text-body-md text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
          onClick={onBack}
        >
          Volver
        </button>
        <button
          className="flex-1 px-4 py-2.5 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
          onClick={onNext}
        >
          Continuar
        </button>
      </div>
    </div>
  );
}
