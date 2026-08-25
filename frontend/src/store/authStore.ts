import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api, type UserDTO } from "../services/api";

export interface User {
  id: string;
  name: string;
  email?: string;
  avatarColor?: string;
  hasPin: boolean;
  initials: string;
  role: string;
  plan: "free" | "pro";
  hasUsedAI: boolean;
  profileMd?: string;
}

export function dtoToUser(dto: UserDTO): User {
  const parts = dto.name.trim().split(" ");
  const initials =
    parts.length > 1
      ? (parts[0][0] + parts[1][0]).toUpperCase()
      : dto.name.slice(0, 2).toUpperCase();

  return {
    id: dto.id,
    name: dto.name,
    email: dto.email,
    avatarColor: dto.avatarColor || "#d1f107",
    hasPin: dto.hasPin,
    initials: initials || "OZ",
    role: dto.role || "developer",
    plan: (dto.plan as "free" | "pro") || "free",
    hasUsedAI: true,
    profileMd: dto.profileMd,
  };
}

interface AuthState {
  user: User | null;
  profiles: User[];
  isAuthenticated: boolean;
  isLocked: boolean;
  onboardingCompleted: boolean;
  fetchProfiles: () => Promise<User[]>;
  setUser: (user: User) => void;
  selectProfile: (profile: User, pin?: string) => Promise<{ success: boolean; error?: string }>;
  createProfile: (data: {
    name: string;
    email?: string;
    avatarColor?: string;
    pin?: string;
    role?: string;
  }) => Promise<{ success: boolean; user?: User; error?: string }>;
  lockSession: () => void;
  unlockSession: (pin: string) => Promise<{ success: boolean; error?: string }>;
  updatePin: (newPin: string, oldPin?: string) => Promise<{ success: boolean; error?: string }>;
  updateProfileMd: (md: string) => Promise<void>;
  completeOnboarding: () => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      profiles: [],
      isAuthenticated: false,
      isLocked: false,
      onboardingCompleted: false,

      fetchProfiles: async () => {
        try {
          const list = await api.auth.listProfiles();
          const mapped = list.map(dtoToUser);
          set({ profiles: mapped });
          return mapped;
        } catch {
          return get().profiles;
        }
      },

      setUser: (user) => set({ user, isAuthenticated: true, isLocked: false }),

      selectProfile: async (profile, pin) => {
        try {
          const res = await api.auth.login({ userId: profile.id, pin });
          const user = dtoToUser(res.user);
          set({ user, isAuthenticated: true, isLocked: false, onboardingCompleted: true });
          return { success: true };
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "PIN de seguridad incorrecto";
          return { success: false, error: msg };
        }
      },

      createProfile: async (data) => {
        try {
          const res = await api.auth.createProfile(data);
          const newUser = dtoToUser(res);
          set((s) => ({
            user: newUser,
            profiles: [...s.profiles.filter((p) => p.id !== newUser.id), newUser],
            isAuthenticated: true,
            isLocked: false,
            onboardingCompleted: true,
          }));
          return { success: true, user: newUser };
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "Error al crear perfil";
          return { success: false, error: msg };
        }
      },

      lockSession: () => {
        set({ isLocked: true });
      },

      unlockSession: async (pin: string) => {
        const currentUser = get().user;
        if (!currentUser) return { success: false, error: "No hay sesión activa" };

        try {
          const res = await api.auth.verifyPin({ userId: currentUser.id, pin });
          if (res.valid) {
            set({ isLocked: false });
            return { success: true };
          }
          return { success: false, error: "PIN incorrecto" };
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "PIN de seguridad incorrecto";
          return { success: false, error: msg };
        }
      },

      updatePin: async (newPin, oldPin) => {
        const currentUser = get().user;
        if (!currentUser) return { success: false, error: "No hay usuario activo" };

        try {
          const res = await api.auth.updatePin({
            userId: currentUser.id,
            oldPin,
            newPin,
          });
          set((s) => ({
            user: s.user ? { ...s.user, hasPin: res.hasPin } : null,
          }));
          return { success: true };
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : "Error al actualizar PIN";
          return { success: false, error: msg };
        }
      },

      updateProfileMd: async (md) => {
        set((s) => (s.user ? { user: { ...s.user, profileMd: md } } : {}));
        try {
          await api.users.updateProfile(md);
        } catch {
          // non-critical
        }
      },

      completeOnboarding: () => set({ onboardingCompleted: true }),

      logout: () => {
        set({ user: null, isAuthenticated: false, isLocked: true });
      },
    }),
    {
      name: "ozy-auth",
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        isLocked: state.isLocked,
        onboardingCompleted: state.onboardingCompleted,
      }),
    }
  )
);
