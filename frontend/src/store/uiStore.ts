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
  coworkMode: boolean;
  settingsOpen: boolean;
  settingsCategory: string;
  onboardingStep: number;
  editingProjectId: string | null;
  setActiveView: (view: View) => void;
  toggleSidebar: () => void;
  toggleCoworkMode: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSearchOpen: (open: boolean) => void;
  setSettingsOpen: (open: boolean) => void;
  setSettingsCategory: (cat: string) => void;
  openSettings: (cat?: string) => void;
  setOnboardingStep: (step: number) => void;
  setEditingProjectId: (id: string | null) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      activeView: "home",
      sidebarOpen: true,
      searchOpen: false,
      coworkMode: false,
      settingsOpen: false,
      settingsCategory: "general",
      onboardingStep: 0,
      editingProjectId: null,
      setActiveView: (view) => set({ activeView: view }),
      toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
      toggleCoworkMode: () => set((s) => ({ coworkMode: !s.coworkMode })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSearchOpen: (open) => set({ searchOpen: open }),
      setSettingsOpen: (open) => set({ settingsOpen: open }),
      setSettingsCategory: (cat) => set({ settingsCategory: cat }),
      openSettings: (cat = "general") => set({ settingsOpen: true, settingsCategory: cat }),
      setOnboardingStep: (step) => set({ onboardingStep: step }),
      setEditingProjectId: (id) => set({ editingProjectId: id }),
    }),
    { name: "ozy-ui", partialize: (state) => ({
      activeView: state.activeView,
      sidebarOpen: state.sidebarOpen,
      coworkMode: state.coworkMode,
      settingsCategory: state.settingsCategory,
      onboardingStep: state.onboardingStep,
      editingProjectId: state.editingProjectId,
    })},
  ),
);
