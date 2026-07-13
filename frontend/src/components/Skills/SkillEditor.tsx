import { useState } from "react";
import {
  useSkillsStore,
  type Skill,
  type SkillExecutionType,
} from "../../store/skillsStore";

interface SkillEditorProps {
  skill: Skill | null;
  onClose: () => void;
}

export default function SkillEditor({ skill, onClose }: SkillEditorProps) {
  const addSkill = useSkillsStore((s) => s.addSkill);
  const updateSkill = useSkillsStore((s) => s.updateSkill);
  const [name, setName] = useState(skill?.name || "");
  const [description, setDescription] = useState(skill?.description || "");
  const [triggerPattern, setTriggerPattern] = useState(
    skill?.triggerPattern || "",
  );
  const [executionType, setExecutionType] = useState<SkillExecutionType>(
    skill?.executionType || "prompt_template",
  );
  const [configValue, setConfigValue] = useState(
    skill?.config?.template || "",
  );

  const handleSave = () => {
    const config: Record<string, string> = {};
    if (executionType === "prompt_template") {
      config.template = configValue;
    }
    if (skill) {
      updateSkill(skill.id, {
        name,
        description,
        triggerPattern,
        executionType,
        config,
      });
    } else {
      addSkill({
        id: Date.now().toString(),
        name,
        description,
        triggerPattern,
        executionType,
        config,
      });
    }
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-surface-elevated border border-border-subtle rounded-2xl w-full max-w-lg mx-4 overflow-hidden">
        <div className="flex items-center justify-between p-5 border-b border-border-subtle">
          <h2 className="text-headline-md font-headline-md text-on-surface">
            {skill ? "Editar skill" : "Nueva skill"}
          </h2>
          <button
            className="text-text-muted hover:text-on-surface transition-colors"
            onClick={onClose}
          >
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>
        <div className="p-5 flex flex-col gap-4">
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Nombre
            </label>
            <input
              className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ej: Revisar código"
            />
          </div>
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Descripción
            </label>
            <textarea
              className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md resize-none"
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="¿Qué hace esta skill?"
            />
          </div>
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Patrón de activación
            </label>
            <input
              className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-body-md"
              value={triggerPattern}
              onChange={(e) => setTriggerPattern(e.target.value)}
              placeholder="Ej: revisa este código"
            />
          </div>
          <div>
            <label className="text-label-caps text-text-muted mb-1.5 block">
              Tipo de ejecución
            </label>
            <div className="flex gap-2">
              {(["script", "prompt_template", "api_call"] as const).map(
                (type) => (
                  <button
                    key={type}
                    className={`px-3 py-1.5 rounded-lg text-body-md transition-colors ${
                      executionType === type
                        ? "bg-primary-container text-on-primary"
                        : "bg-surface-container text-text-muted hover:text-on-surface border border-border-subtle"
                    }`}
                    onClick={() => setExecutionType(type)}
                  >
                    {type === "script"
                      ? "Script"
                      : type === "prompt_template"
                        ? "Prompt"
                        : "API Call"}
                  </button>
                ),
              )}
            </div>
          </div>
          {executionType === "prompt_template" && (
            <div>
              <label className="text-label-caps text-text-muted mb-1.5 block">
                Template
              </label>
              <textarea
                className="w-full bg-surface-container border border-border-subtle rounded-lg px-3 py-2 text-on-surface outline-none focus:border-primary-container transition-colors text-code-sm resize-none"
                rows={3}
                value={configValue}
                onChange={(e) => setConfigValue(e.target.value)}
                placeholder='Ej: Revisa el siguiente código y encuentra errores: {{input}}'
              />
            </div>
          )}
        </div>
        <div className="flex justify-end gap-3 p-5 border-t border-border-subtle">
          <button
            className="px-4 py-2 text-body-md text-text-muted hover:text-on-surface transition-colors"
            onClick={onClose}
          >
            Cancelar
          </button>
          <button
            className="px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={handleSave}
          >
            Guardar
          </button>
        </div>
      </div>
    </div>
  );
}
