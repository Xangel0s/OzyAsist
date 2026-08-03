import { useToastStore } from "../../store/toastStore";

export default function Toast() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  if (toasts.length === 0) return null;

  const getIcon = (type?: string) => {
    if (type === "error") return "error";
    if (type === "warning") return "warning";
    if (type === "info") return "info";
    return "check_circle";
  };

  const getIconStyle = (type?: string) => {
    if (type === "error") return "bg-red-500/15 border-red-500/30 text-red-400";
    if (type === "warning") return "bg-amber-500/15 border-amber-500/30 text-amber-400";
    if (type === "info") return "bg-sky-500/15 border-sky-500/30 text-sky-400";
    return "bg-[#d1f107]/15 border-[#d1f107]/30 text-[#d1f107]";
  };

  return (
    <div className="fixed bottom-20 left-1/2 -translate-x-1/2 z-[9999] flex flex-col gap-2 items-center pointer-events-none">
      {toasts.map((t) => (
        <div
          key={t.id}
          className="pointer-events-auto bg-[#1c1c1c] border border-white/15 rounded-2xl px-4 py-2.5 shadow-2xl shadow-black flex items-center gap-3 animate-slideUp cursor-pointer backdrop-blur-md"
          onClick={() => dismiss(t.id)}
        >
          <div className={`flex items-center justify-center p-1.5 rounded-xl border ${getIconStyle(t.icon)}`}>
            <span className="material-symbols-outlined text-[16px]">{getIcon(t.icon)}</span>
          </div>
          <span className="text-[13px] font-medium text-white pr-1">{t.message}</span>
        </div>
      ))}
    </div>
  );
}
