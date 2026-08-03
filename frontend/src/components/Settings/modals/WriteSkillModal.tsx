import { useState, useEffect } from "react";
import { useSkillsStore } from "../../../store/skillsStore";
import { useToastStore } from "../../../store/toastStore";

interface WriteSkillModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialCode?: string;
}

export default function WriteSkillModal({ isOpen, onClose, initialCode }: WriteSkillModalProps) {
  const addSkill = useSkillsStore((s) => s.addSkill);
  const toast = useToastStore((s) => s.show);
  const defaultTemplate = `---
name: mi-habilidad
description: Instrucciones personalizadas para la IA
---

# Instrucciones
Escribe aquí la plantilla de prompt o comportamiento deseado...`;

  const [customSkillCode, setCustomSkillCode] = useState(initialCode || defaultTemplate);

  useEffect(() => {
    if (isOpen) {
      setCustomSkillCode(initialCode || defaultTemplate);
    }
  }, [isOpen, initialCode]);

  if (!isOpen) return null;

  const handleSave = async () => {
    if (!customSkillCode.trim()) {
      toast("Ingresa las instrucciones de la habilidad", "error");
      return;
    }

    let skillName = "custom-skill-" + Date.now().toString().slice(-4);
    let description = "Habilidad personalizada redactada en editor";
    const nameMatch = customSkillCode.match(/name:\s*([^\n\r]+)/i);
    if (nameMatch) skillName = nameMatch[1].trim().toLowerCase().replace(/\s+/g, "-");
    const descMatch = customSkillCode.match(/description:\s*([^\n\r]+)/i);
    if (descMatch) description = descMatch[1].trim();

    const id = await addSkill({
      id: "",
      name: skillName,
      description: description,
      triggerPattern: "/" + skillName,
      executionType: "prompt_template",
      config: { template: customSkillCode },
    });

    if (id) {
      toast(`Habilidad /${skillName} guardada e instalada`, "success");
      onClose();
    } else {
      toast("Error guardando la habilidad", "error");
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h3 className="text-[18px] font-semibold">Escribir instrucciones de la habilidad (SKILL.md)</h3>
          <button
            className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined text-[20px]">close</span>
          </button>
        </div>

        <textarea
          className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl p-4 text-[13px] font-mono text-white placeholder:text-white/30 outline-none focus:border-white/30 h-64 resize-none"
          value={customSkillCode}
          onChange={(e) => setCustomSkillCode(e.target.value)}
        />

        <div className="flex justify-end gap-2.5">
          <button
            className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-xl transition-colors"
            onClick={onClose}
          >
            Cancelar
          </button>
          <button
            className="px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-colors"
            onClick={handleSave}
          >
            Guardar e Instalar
          </button>
        </div>
      </div>
    </div>
  );
}
