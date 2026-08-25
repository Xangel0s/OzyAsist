interface Chip {
  icon: string;
  label: string;
  prompt: string;
}

const chips: Chip[] = [
  { icon: "code", label: "Código", prompt: "Ayúdame a escribir y revisar código para " },
  { icon: "edit", label: "Escribir", prompt: "Ayúdame a redactar un texto sobre " },
  { icon: "school", label: "Aprender", prompt: "Quiero aprender sobre " },
  { icon: "lightbulb", label: "Ideas", prompt: "Dame ideas innovadoras sobre " },
];

interface Props {
  onSend?: (message: string) => void;
}

export default function ActionChips({ onSend }: Props) {
  const handleClick = (chip: Chip) => {
    if (chip.prompt && onSend) {
      onSend(chip.prompt);
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
