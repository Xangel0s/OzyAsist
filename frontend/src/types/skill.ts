export interface SkillConfig {
  template?: string;
  command?: string;
  args?: string[];
  [key: string]: any;
}

export interface Skill {
  id: string;
  name: string;
  description: string;
  triggerPattern?: string;
  executionType?: string;
  config?: SkillConfig;
  author?: string;
  date?: string;
  custom?: boolean;
}
