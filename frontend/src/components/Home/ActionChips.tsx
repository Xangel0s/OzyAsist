import { useUIStore } from "../../store/uiStore";

const chips = [
  { icon: "code", label: "Código", view: "code" as const },
  { icon: "edit", label: "Escribir", view: "chat" as const },
  { icon: "school", label: "Aprender", view: "chat" as const },
  { icon: "local_cafe", label: "Vida personal", view: "chat" as const },
  { icon: "lightbulb", label: "Selección de modelo", view: "chat" as const },
];

export default function ActionChips() {
  const setActiveView = useUIStore((s) => s.setActiveView);

  return (
    <div className="flex flex-wrap items-center justify-center gap-3 mt-6 w-full">
      {chips.map((chip) => (
        <button
          key={chip.label}
          className="flex items-center gap-2 px-4 py-2 bg-surface-container rounded-full border border-border-subtle text-on-surface hover:bg-surface-variant transition-colors text-[13px] font-medium"
          onClick={() => setActiveView(chip.view)}
        >
          <span className="material-symbols-outlined text-[16px] text-text-muted">
            {chip.icon}
          </span>
          {chip.label}
        </button>
      ))}
    </div>
  );
}
