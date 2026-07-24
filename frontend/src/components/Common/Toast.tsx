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
          className="pointer-events-auto bg-[#1c1c1c] border border-white/20 rounded-2xl px-5 py-3 shadow-2xl shadow-black flex items-center gap-3 animate-slideUp cursor-pointer backdrop-blur-md"
          onClick={() => dismiss(t.id)}
        >
          <div className="flex items-center gap-1.5 bg-[#d1f107]/15 border border-[#d1f107]/30 text-[#d1f107] text-[11px] font-bold px-2.5 py-1 rounded-lg tracking-wider">
            <span className="material-symbols-outlined text-[14px]">check_circle</span>
            <span>ÉXITO</span>
          </div>
          <span className="text-[13px] font-medium text-white">{t.message}</span>
        </div>
      ))}
    </div>
  );
}
