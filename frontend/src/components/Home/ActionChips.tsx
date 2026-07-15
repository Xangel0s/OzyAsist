import { useUIStore } from "../../store/uiStore";

interface Chip {
  icon: string;
  label: string;
  view: "chat" | "code";
  prompt?: string;
}

const chips: Chip[] = [
  { icon: "code", label: "Código", view: "code" },
  { icon: "edit", label: "Escribir", view: "chat", prompt: "Ayúdame a escribir un texto sobre" },
  { icon: "school", label: "Aprender", view: "chat", prompt: "Quiero aprender sobre" },
  { icon: "local_cafe", label: "Vida personal", view: "chat", prompt: "Necesito consejos sobre" },
  { icon: "lightbulb", label: "Selección de modelo", view: "chat" },
];

interface Props {
  onSend?: (message: string) => void;
}

export default function ActionChips({ onSend }: Props) {
  const setActiveView = useUIStore((s) => s.setActiveView);

  const handleClick = (chip: Chip) => {
    if (chip.prompt && onSend) {
      onSend(chip.prompt);
    } else {
      setActiveView(chip.view);
    }
  };

  return (
    <div className="flex flex-wrap items-center justify-center gap-3 mt-6 w-full">
      {chips.map((chip) => (
        <button
          key={chip.label}
          className="flex items-center gap-2 px-4 py-2 bg-[#1e1e1e] rounded-full border border-white/10 text-white/70 hover:bg-[#252525] hover:text-white transition-colors text-[13px] font-medium"
          onClick={() => handleClick(chip)}
        >
          <span className="material-symbols-outlined text-[16px] text-white/40">
            {chip.icon}
          </span>
          {chip.label}
        </button>
      ))}
    </div>
  );
}
