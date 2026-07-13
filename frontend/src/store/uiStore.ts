import { create } from "zustand";

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
  setActiveView: (view: View) => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSearchOpen: (open: boolean) => void;
  setOnboardingStep: (step: number) => void;
}

export const useUIStore = create<UIState>((set) => ({
  activeView: "home",
  sidebarOpen: true,
  searchOpen: false,
  onboardingStep: 0,
  setActiveView: (view) => set({ activeView: view }),
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  setSearchOpen: (open) => set({ searchOpen: open }),
  setOnboardingStep: (step) => set({ onboardingStep: step }),
}));
