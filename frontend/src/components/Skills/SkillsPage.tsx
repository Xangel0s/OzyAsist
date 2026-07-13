import { useState } from "react";
import { useSkillsStore, type Skill } from "../../store/skillsStore";
import SkillEditor from "./SkillEditor";

export default function SkillsPage() {
  const skills = useSkillsStore((s) => s.skills);
  const removeSkill = useSkillsStore((s) => s.removeSkill);
  const [editing, setEditing] = useState<Skill | null>(null);
  const [showEditor, setShowEditor] = useState(false);

  const executionLabels: Record<string, string> = {
    script: "Script",
    prompt_template: "Prompt template",
    api_call: "API call",
  };

  return (
    <div className="flex-1 flex h-full bg-surface-deep overflow-y-auto">
      <div className="max-w-container-max mx-auto w-full px-gutter py-8">
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-headline-md font-headline-md text-on-surface">
            Skills
          </h1>
          <button
            className="flex items-center gap-2 px-4 py-2 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium"
            onClick={() => {
              setEditing(null);
              setShowEditor(true);
            }}
          >
            <span className="material-symbols-outlined text-[18px]">add</span>
            Nueva skill
          </button>
        </div>

        {showEditor && (
          <SkillEditor
            skill={editing}
            onClose={() => setShowEditor(false)}
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {skills.map((skill) => (
            <div
              key={skill.id}
              className="bg-surface-container rounded-xl border border-border-subtle p-5"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-lg bg-primary-container/20 flex items-center justify-center">
                    <span className="material-symbols-outlined text-primary-container text-[18px]">
                      settings_suggest
                    </span>
                  </div>
                  <div>
                    <h3 className="text-body-lg font-medium text-on-surface">
                      {skill.name}
                    </h3>
                    <span className="text-label-caps text-text-muted">
                      {executionLabels[skill.executionType]}
                    </span>
                  </div>
                </div>
                <div className="flex gap-1">
                  <button
                    className="p-1.5 rounded-lg text-text-muted hover:text-on-surface hover:bg-surface-variant transition-colors"
                    onClick={() => {
                      setEditing(skill);
                      setShowEditor(true);
                    }}
                  >
                    <span className="material-symbols-outlined text-[16px]">edit</span>
                  </button>
                  <button
                    className="p-1.5 rounded-lg text-text-muted hover:text-error hover:bg-surface-variant transition-colors"
                    onClick={() => removeSkill(skill.id)}
                  >
                    <span className="material-symbols-outlined text-[16px]">delete</span>
                  </button>
                </div>
              </div>
              <p className="text-text-muted text-body-md mb-3">
                {skill.description}
              </p>
              <div className="bg-surface-deep rounded-lg px-3 py-2 text-code-sm text-text-muted">
                <span className="text-primary-container">trigger:</span>{" "}
                {skill.triggerPattern}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
