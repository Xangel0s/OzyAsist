import { useRef } from "react";
import JSZip from "jszip";
import { useSkillsStore } from "../../../store/skillsStore";
import { useToastStore } from "../../../store/toastStore";

interface UploadSkillModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function UploadSkillModal({ isOpen, onClose }: UploadSkillModalProps) {
  const addSkill = useSkillsStore((s) => s.addSkill);
  const toast = useToastStore((s) => s.show);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen) return null;

  const processUploadedSkillFile = async (file: File) => {
    try {
      let skillName = file.name.replace(/\.[^/.]+$/, "").toLowerCase().replace(/\s+/g, "-");
      let description = "Habilidad importada desde " + file.name;
      let template = "";

      if (file.name.endsWith(".zip")) {
        const zip = await JSZip.loadAsync(file);
        let skillMdFile = zip.file("SKILL.md") || zip.file("skill.md");
        if (!skillMdFile) {
          const files = Object.keys(zip.files);
          const found = files.find((f) => f.toLowerCase().endsWith(".md"));
          if (found) skillMdFile = zip.file(found);
        }
        if (skillMdFile) {
          template = await skillMdFile.async("string");
        } else {
          template = `# Habilidad ${skillName}\nImportado desde archivo comprimido ${file.name}`;
        }
      } else {
        template = await file.text();
      }

      const nameMatch = template.match(/name:\s*([^\n\r]+)/i);
      if (nameMatch) skillName = nameMatch[1].trim().toLowerCase().replace(/\s+/g, "-");
      const descMatch = template.match(/description:\s*([^\n\r]+)/i);
      if (descMatch) description = descMatch[1].trim();

      const id = await addSkill({
        id: "",
        name: skillName,
        description: description,
        triggerPattern: "/" + skillName,
        executionType: "prompt_template",
        config: { template },
      });

      if (id) {
        toast(`Habilidad /${skillName} importada e instalada exitosamente`, "success");
        onClose();
      } else {
        toast("Error al registrar la habilidad", "error");
      }
    } catch (err: any) {
      toast("Error al procesar el archivo de habilidad: " + err.message, "error");
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h3 className="text-[18px] font-semibold">Subir habilidad o plugin (.zip, .yaml, .md)</h3>
          <button
            className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined text-[20px]">close</span>
          </button>
        </div>

        <input
          type="file"
          ref={fileInputRef}
          accept=".yaml,.yml,.md,.json,.zip,.txt,.skill"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) processUploadedSkillFile(file);
          }}
        />

        <div
          className="border-2 border-dashed border-white/15 hover:border-white/30 rounded-2xl p-8 flex flex-col items-center justify-center gap-3 cursor-pointer bg-[#1c1c1c] transition-all group"
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            e.preventDefault();
            const file = e.dataTransfer.files?.[0];
            if (file) processUploadedSkillFile(file);
          }}
        >
          <span className="material-symbols-outlined text-[40px] text-white/30 group-hover:text-white/60 transition-colors">
            folder_zip
          </span>
          <span className="text-[13px] text-white/60 group-hover:text-white transition-colors">
            Arrastra y suelta o haz clic para cargar (.zip, .yaml, .md)
          </span>
        </div>
      </div>
    </div>
  );
}
