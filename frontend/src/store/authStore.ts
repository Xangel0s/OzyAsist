import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "../services/api";

export interface User {
  id: string;
  name: string;
  initials: string;
  role: string;
  plan: "free" | "pro";
  hasUsedAI: boolean;
  profileMd?: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  onboardingCompleted: boolean;
  setUser: (user: User) => void;
  updateProfileMd: (md: string) => Promise<void>;
  completeOnboarding: () => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      onboardingCompleted: false,
      setUser: (user) => set({ user, isAuthenticated: true }),
      updateProfileMd: async (md) => {
        set((s) => s.user ? { user: { ...s.user, profileMd: md } } : {});
        try {
          await api.users.updateProfile(md);
        } catch {
          // non-critical: profile persists locally even if backend unreachable
        }
      },
      completeOnboarding: () => set({ onboardingCompleted: true }),
      logout: () => set({ user: null, isAuthenticated: false, onboardingCompleted: false }),
    }),
    {
      name: "ozy-auth",
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        onboardingCompleted: state.onboardingCompleted,
      }),
    },
  ),
);
