import { create } from "zustand";
import { persist } from "zustand/middleware";

export type View =
  | "home"
  | "code"
  | "chat"
  | "projects"
  | "skills"
  | "connectors"
  | "onboarding"
  | "search";

interface UIState {
  activeView: View;
  sidebarOpen: boolean;
  searchOpen: boolean;
  onboardingStep: number;
  editingProjectId: string | null;
  setActiveView: (view: View) => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSearchOpen: (open: boolean) => void;
  setOnboardingStep: (step: number) => void;
  setEditingProjectId: (id: string | null) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      activeView: "home",
      sidebarOpen: true,
      searchOpen: false,
      onboardingStep: 0,
      editingProjectId: null,
      setActiveView: (view) => set({ activeView: view }),
      toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSearchOpen: (open) => set({ searchOpen: open }),
      setOnboardingStep: (step) => set({ onboardingStep: step }),
      setEditingProjectId: (id) => set({ editingProjectId: id }),
    }),
    { name: "ozy-ui", partialize: (state) => ({
      activeView: state.activeView,
      sidebarOpen: state.sidebarOpen,
      onboardingStep: state.onboardingStep,
      editingProjectId: state.editingProjectId,
    })},
  ),
);
