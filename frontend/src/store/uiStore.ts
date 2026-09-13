import { create } from "zustand";
import { persist } from "zustand/middleware";

export type View =
  | "home"
  | "chat"
  | "chats"
  | "projects"
  | "skills"
  | "connectors"
  | "onboarding"
  | "search";

export type Theme = "system" | "light" | "dark";

interface UIState {
  activeView: View;
  sidebarOpen: boolean;
  searchOpen: boolean;
  coworkMode: boolean;
  settingsOpen: boolean;
  settingsCategory: string;
  onboardingStep: number;
  editingProjectId: string | null;
  theme: Theme;
  voiceLiveOpen: boolean;
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
  setTheme: (theme: Theme) => void;
  setVoiceLiveOpen: (open: boolean) => void;
  toggleVoiceLive: () => void;
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
      theme: "dark",
      voiceLiveOpen: false,
      setActiveView: (view) => set({ activeView: view }),
      toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
      toggleCoworkMode: () => set((s) => ({ coworkMode: !s.coworkMode })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setSearchOpen: (open) => set({ searchOpen: open }),
      setSettingsOpen: (open) => set({ settingsOpen: open }),
      setSettingsCategory: (cat) => {
        let target = cat;
        if (target === "models" || target === "providers") target = "proveedores";
        if (target === "skills") target = "habilidades";
        if (target === "connectors") target = "conectores";
        set({ settingsCategory: target });
      },
      openSettings: (cat = "general") => {
        let target = cat;
        if (target === "models" || target === "providers") target = "proveedores";
        if (target === "skills") target = "habilidades";
        if (target === "connectors") target = "conectores";
        set({ settingsOpen: true, settingsCategory: target });
      },
      setOnboardingStep: (step) => set({ onboardingStep: step }),
      setEditingProjectId: (id) => set({ editingProjectId: id }),
      setTheme: (theme) => set({ theme }),
      setVoiceLiveOpen: (open) => set({ voiceLiveOpen: open }),
      toggleVoiceLive: () => set((s) => ({ voiceLiveOpen: !s.voiceLiveOpen })),
    }),
    { name: "ozy-ui", partialize: (state) => ({
      activeView: state.activeView,
      sidebarOpen: state.sidebarOpen,
      coworkMode: state.coworkMode,
      settingsCategory: state.settingsCategory,
      onboardingStep: state.onboardingStep,
      editingProjectId: state.editingProjectId,
      theme: state.theme,
    })},
  ),
);
