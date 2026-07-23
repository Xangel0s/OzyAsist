import { useToastStore } from "../../store/toastStore";

export default function Toast() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-20 left-1/2 -translate-x-1/2 z-[9999] flex flex-col gap-2 items-center pointer-events-none">
      {toasts.map((t) => (
        <div
          key={t.id}
          className="pointer-events-auto bg-[#252525] border border-white/20 rounded-xl px-5 py-3 shadow-2xl shadow-black/80 flex items-center gap-3 animate-slideUp cursor-pointer backdrop-blur-md"
          onClick={() => dismiss(t.id)}
        >
          <span className="material-symbols-outlined text-[18px] text-[#c8e64a]">
            {t.icon || "check_circle"}
          </span>
          <span className="text-[13px] font-medium text-white">{t.message}</span>
        </div>
      ))}
    </div>
  );
}
