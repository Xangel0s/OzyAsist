import { create } from 'zustand';

export interface TaskTelemetry {
  cpu_pct: number;
  ram_free_mb: number;
}

export interface TaskStepProgress {
  step_id: string;
  step_order: number;
  step_description: string;
  agent_assigned: 'ozy' | 'charc' | 'nine';
  status: 'pending' | 'running' | 'verifying' | 'recovering' | 'completed' | 'failed';
}

export interface ActiveTaskState {
  task_id: string;
  title: string;
  status: 'pending' | 'planning' | 'running' | 'blocked_approval' | 'completed' | 'failed' | 'cancelled';
  step_current: number;
  step_total: number;
  current_step_description?: string;
  telemetry?: TaskTelemetry;
}

export interface PINApprovalRequest {
  task_id: string;
  step_id: string;
  action: string;
  command?: string;
  risk_level: 'low' | 'medium' | 'high';
  reason: string;
}

interface TaskStore {
  activeTask: ActiveTaskState | null;
  pinRequest: PINApprovalRequest | null;
  isHUDVisible: boolean;
  isHUDExpanded: boolean;
  
  // Acciones
  setActiveTask: (task: ActiveTaskState | null) => void;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  updateProgress: (payload: any) => void;
  setPINRequest: (req: PINApprovalRequest | null) => void;
  toggleHUD: () => void;
  toggleHUDExpanded: () => void;
  clearTask: () => void;
}

export const useTaskStore = create<TaskStore>((set) => ({
  activeTask: null,
  pinRequest: null,
  isHUDVisible: true,
  isHUDExpanded: false,

  setActiveTask: (task) => set({ activeTask: task }),

  updateProgress: (data) =>
    set((state) => {
      const active = state.activeTask;
      return {
        activeTask: {
          task_id: data.task_id,
          title: data.title || active?.title || 'Tarea en segundo plano',
          status: data.status,
          step_current: data.step_current,
          step_total: data.step_total,
          current_step_description: data.step_description,
          telemetry: data.telemetry,
        },
      };
    }),

  setPINRequest: (pinRequest) => set({ pinRequest }),
  toggleHUD: () => set((state) => ({ isHUDVisible: !state.isHUDVisible })),
  toggleHUDExpanded: () => set((state) => ({ isHUDExpanded: !state.isHUDExpanded })),
  clearTask: () => set({ activeTask: null, pinRequest: null, isHUDExpanded: false }),
}));
