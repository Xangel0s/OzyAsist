import { useToastStore } from "../../store/toastStore";

export default function Toast() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  if (toasts.length === 0) return null;

  const getIndicator = (type?: string) => {
    switch (type) {
      case "error":
        return {
          dot: "bg-rose-500",
          icon: "error",
          iconColor: "text-rose-400",
        };
      case "warning":
        return {
          dot: "bg-amber-400",
          icon: "warning",
          iconColor: "text-amber-400",
        };
      case "info":
        return {
          dot: "bg-sky-400",
          icon: "info",
          iconColor: "text-sky-400",
        };
      default:
        return {
          dot: "bg-[#d1f107]",
          icon: "check_circle",
          iconColor: "text-[#d1f107]",
        };
    }
  };

  return (
    <div className="fixed bottom-6 right-6 z-[99999] flex flex-col gap-2 pointer-events-none select-none max-w-sm w-auto font-sans">
      {toasts.map((t) => {
        const ind = getIndicator(t.type);
        return (
          <div
            key={t.id}
            className="pointer-events-auto flex items-center gap-2.5 px-3.5 py-2.5 bg-[#202020]/95 hover:bg-[#262626] backdrop-blur-xl border border-white/10 rounded-2xl shadow-xl shadow-black/60 transition-all duration-200 animate-slideUp group cursor-pointer hover:border-white/20"
            onClick={() => dismiss(t.id)}
          >
            {/* Status Dot / Clean Icon without box frame */}
            <span className={`material-symbols-outlined text-[16px] shrink-0 ${ind.iconColor}`}>
              {ind.icon}
            </span>

            {/* Notification message */}
            <span className="text-[12px] font-sans font-medium text-zinc-100 leading-snug whitespace-nowrap pr-1">
              {t.message}
            </span>

            {/* Subtle dismiss button */}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                dismiss(t.id);
              }}
              className="text-zinc-500 hover:text-zinc-200 transition-colors p-0.5 ml-1 rounded"
              title="Cerrar"
            >
              <span className="material-symbols-outlined text-[14px]">close</span>
            </button>
          </div>
        );
      })}
    </div>
  );
}
