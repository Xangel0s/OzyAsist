import { useToastStore } from "../../store/toastStore";

export default function Toast() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 items-center">
      {toasts.map((t) => (
        <div
          key={t.id}
          className="bg-surface-elevated border border-primary-container/30 rounded-xl px-5 py-3 shadow-lg shadow-black/30 flex items-center gap-3 animate-slideUp cursor-pointer"
          onClick={() => dismiss(t.id)}
        >
          {t.icon && (
            <span className="material-symbols-outlined text-[18px] text-primary-container">
              {t.icon}
            </span>
          )}
          <span className="text-body-md text-on-surface">{t.message}</span>
        </div>
      ))}
    </div>
  );
}
