import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type SkillDTO } from "../services/api";

export type SkillExecutionType = "script" | "prompt_template" | "api_call";

export interface Skill {
  id: string;
  name: string;
  description: string;
  triggerPattern: string;
  executionType: SkillExecutionType;
  config: Record<string, string>;
}

interface SkillsState {
  skills: Skill[];
  loading: boolean;
  disabledSkillsMap: Record<string, boolean>; // name -> true if disabled
  addSkill: (skill: Skill) => Promise<string | null>;
  removeSkill: (id: string) => Promise<void>;
  updateSkill: (id: string, skill: Partial<Skill>) => void;
  loadSkills: () => Promise<void>;
  toggleSkillEnabled: (name: string) => void;
}

const dtoToSkill = (dto: SkillDTO): Skill => {
  let config: Record<string, string> = {};
  try { config = JSON.parse(dto.config || "{}"); } catch { /* ignore */ }
  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    triggerPattern: dto.triggerPattern,
    executionType: dto.executionType,
    config,
  };
};

export const useSkillsStore = create<SkillsState>()(
  persist(
    (set) => ({
      skills: [],
      loading: true,
      disabledSkillsMap: {},

      toggleSkillEnabled: (name) =>
        set((s) => ({
          disabledSkillsMap: {
            ...s.disabledSkillsMap,
            [name]: !s.disabledSkillsMap[name],
          },
        })),

      addSkill: async (skill) => {
        try {
          const currentSkills = useSkillsStore.getState().skills;
          const existing = currentSkills.find((s) => s.name.toLowerCase() === skill.name.toLowerCase());
          if (existing) {
            await api.skills.delete(existing.id).catch(() => {});
          }

          const dto = await api.skills.create({
            name: skill.name,
            description: skill.description,
            triggerPattern: skill.triggerPattern,
            executionType: skill.executionType,
            config: JSON.stringify(skill.config),
          });
          const mapped = dtoToSkill(dto);
          set((s) => ({
            skills: [...s.skills.filter((sk) => sk.name.toLowerCase() !== skill.name.toLowerCase()), mapped],
          }));
          return mapped.id;
        } catch {
          return null;
        }
      },

      removeSkill: async (id) => {
        try {
          await api.skills.delete(id);
          set((s) => ({ skills: s.skills.filter((sk) => sk.id !== id) }));
        } catch {
          // ignore
        }
      },

      updateSkill: (id, updated) =>
        set((s) => ({
          skills: s.skills.map((sk) => (sk.id === id ? { ...sk, ...updated } : sk)),
        })),

      loadSkills: async () => {
        set({ loading: true });
        try {
          const dtos = await api.skills.list();
          set({ skills: dtos.map(dtoToSkill), loading: false });
        } catch {
          set({ loading: false });
        }
      },
    }),
    {
      name: "ozy-skills",
      partialize: (state) => ({
        skills: state.skills,
        disabledSkillsMap: state.disabledSkillsMap,
      }),
    },
  ),
);

useSkillsStore.getState().loadSkills();
