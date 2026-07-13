import { create } from "zustand";
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
  addSkill: (skill: Skill) => Promise<string | null>;
  removeSkill: (id: string) => Promise<void>;
  updateSkill: (id: string, skill: Partial<Skill>) => void;
  loadSkills: () => Promise<void>;
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

export const useSkillsStore = create<SkillsState>((set) => ({
  skills: [],
  loading: true,

  addSkill: async (skill) => {
    try {
      const dto = await api.skills.create({
        name: skill.name,
        description: skill.description,
        triggerPattern: skill.triggerPattern,
        executionType: skill.executionType,
        config: JSON.stringify(skill.config),
      });
      const mapped = dtoToSkill(dto);
      set((s) => ({ skills: [...s.skills, mapped] }));
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
}));

useSkillsStore.getState().loadSkills();
