interface ConsentModalProps {
  intent: string;
  onResolve: (decision: "always" | "once" | "no") => void;
}

export default function ConsentModal({ intent, onResolve }: ConsentModalProps) {
  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60">
      <div className="bg-surface-elevated border border-border-subtle rounded-2xl shadow-2xl max-w-md w-full mx-4 overflow-hidden">
        <div className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 overflow-hidden">
              <img src="/ozybaselogo.png" alt="Ozy" className="w-full h-full object-contain" />
            </div>
            <div>
              <h2 className="text-body-lg font-medium text-on-surface">
                Ozy detectó intención de acción
              </h2>
              <p className="text-label-caps text-text-muted mt-0.5">
                ¿Querés que ejecute esto como tarea de agente?
              </p>
            </div>
          </div>

          <div className="bg-surface-container rounded-xl border border-border-subtle p-4 mb-6">
            <p className="text-body-md text-on-surface whitespace-pre-wrap break-words">
              {intent}
            </p>
          </div>

          <div className="flex flex-col gap-2">
            <button
              className="w-full flex items-center gap-3 px-4 py-3 bg-primary-container text-on-primary rounded-xl hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
              onClick={() => onResolve("always")}
            >
              <span className="material-symbols-outlined text-[20px]">check_circle</span>
              <div className="text-left">
                <div className="font-medium">Sí, siempre</div>
                <div className="text-body-sm opacity-75">Recordar para este proyecto</div>
              </div>
            </button>
            <button
              className="w-full flex items-center gap-3 px-4 py-3 bg-surface-container rounded-xl border border-border-subtle text-on-surface hover:bg-surface-container-high transition-colors text-body-md"
              onClick={() => onResolve("once")}
            >
              <span className="material-symbols-outlined text-[20px]">play_arrow</span>
              <div className="text-left">
                <div>Solo esta vez</div>
                <div className="text-label-caps text-text-muted">Preguntar de nuevo la próxima</div>
              </div>
            </button>
            <button
              className="w-full flex items-center gap-3 px-4 py-3 bg-surface-container rounded-xl border border-border-subtle text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors text-body-md"
              onClick={() => onResolve("no")}
            >
              <span className="material-symbols-outlined text-[20px]">close</span>
              No, responder normalmente
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
