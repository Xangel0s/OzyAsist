import { useCallback, useEffect, useState } from "react";
import { useUIStore } from "./store/uiStore";
import { useAuthStore } from "./store/authStore";
import { useKeyboard, useGlobalEmergencyHandler, useWakeWord } from "./hooks";
import TopAppBar from "./components/Layout/TopAppBar";
import Sidebar from "./components/Layout/Sidebar";
import HomePage from "./components/Home/HomePage";
import ChatPage from "./components/Chat/ChatPage";
import ChatsTasksPage from "./components/Chat/ChatsTasksPage";
import ProjectsPage from "./components/ProjectPanel/ProjectsPage";
import SkillsPage from "./components/Skills/SkillsPage";
import ConnectorsPage from "./components/Connectors/ConnectorsPage";
import OnboardingPage from "./components/Onboarding/OnboardingPage";
import ProfileSelectorModal from "./components/Auth/ProfileSelectorModal";
import SearchModal from "./components/Search/SearchModal";
import Toast from "./components/Common/Toast";
import ErrorBoundary from "./components/Common/ErrorBoundary";
import { FloatingHUD } from "./components/Layout/FloatingHUD";
import { useTaskStore } from "./store/taskStore";
import SettingsModal from "./components/Settings/SettingsModal";
import { OzyLiveOverlay } from "./components/Voice/OzyLiveOverlay";
import { wsService } from "./services/ws";

const pageMap: Record<string, React.ComponentType> = {
  home: HomePage,
  chat: ChatPage,
  chats: ChatsTasksPage,
  projects: ProjectsPage,
  skills: SkillsPage,
  connectors: ConnectorsPage,
  onboarding: OnboardingPage,
};

function HydrationGate({ children }: { children: React.ReactNode }) {
  const [hydrated, setHydrated] = useState(() => useAuthStore.persist.hasHydrated());

  useEffect(() => {
    const unsub = useAuthStore.persist.onFinishHydration(() => setHydrated(true));
    return unsub;
  }, []);

  if (!hydrated) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-background">
        <div className="w-10 h-10 rounded-2xl flex items-center justify-center overflow-hidden animate-pulse">
          <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
        </div>
      </div>
    );
  }

  return <>{children}</>;
}

export default function App() {
  const user = useAuthStore((s) => s.user);
  const isLocked = useAuthStore((s) => s.isLocked);
  const activeView = useUIStore((s) => s.activeView);
  const theme = useUIStore((s) => s.theme);
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const toggleSearch = useCallback(() => setSearchOpen(true), [setSearchOpen]);

  const updateProgress = useTaskStore((s) => s.updateProgress);
  const setPINRequest = useTaskStore((s) => s.setPINRequest);

  const voiceLiveOpen = useUIStore((s) => s.voiceLiveOpen);

  useGlobalEmergencyHandler();
  useKeyboard("k", toggleSearch, { meta: true });
  useWakeWord({
    enabled: !voiceLiveOpen && !isLocked && !!user,
    onWakeWordDetected: () => {
      useUIStore.getState().setVoiceLiveOpen(true);
    },
  });

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const unsubProgress = wsService.subscribe("task:progress", (event: any) => {
      if (event?.task) {
        updateProgress({
          task_id: event.task.id,
          title: event.task.title,
          status: event.task.status,
          step_current: event.task.current_step,
          step_total: event.task.total_steps,
          step_description: event.step?.payload || event.step?.action_type,
        });
      } else if (event?.data) {
        updateProgress(event.data);
      }
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const unsubPIN = wsService.subscribe("task:require_approval", (event: any) => {
      setPINRequest(event.data || event);
    });

    return () => {
      unsubProgress();
      unsubPIN();
    };
  }, [updateProgress, setPINRequest]);

  useEffect(() => {
    const root = document.documentElement;
    const applyTheme = (isDark: boolean) => {
      if (isDark) {
        root.classList.add("dark");
        root.classList.remove("light");
      } else {
        root.classList.add("light");
        root.classList.remove("dark");
      }
    };

    if (theme === "system") {
      const media = window.matchMedia("(prefers-color-scheme: dark)");
      applyTheme(media.matches);
      const listener = (e: MediaQueryListEvent) => applyTheme(e.matches);
      media.addEventListener("change", listener);
      return () => media.removeEventListener("change", listener);
    } else {
      applyTheme(theme === "dark");
    }
  }, [theme]);

  if (!user || isLocked) {
    return (
      <HydrationGate>
        <ProfileSelectorModal />
        <Toast />
      </HydrationGate>
    );
  }

  const Page = pageMap[activeView];

  return (
    <HydrationGate>
      <div className="h-screen w-screen overflow-hidden flex flex-col bg-background text-on-surface font-body-md selection:bg-primary-container selection:text-on-primary-container">
        <TopAppBar />
        <div className="flex flex-1 overflow-hidden">
          <Sidebar />
          <main className="flex-1 flex flex-col overflow-hidden relative">
            <div className="flex-1 flex flex-col animate-fadeIn min-h-0">
              <ErrorBoundary>
                {Page && <Page />}
              </ErrorBoundary>
            </div>
          </main>
        </div>
      </div>
      <FloatingHUD />
      <SearchModal />
      <SettingsModal />
      <OzyLiveOverlay />
      <Toast />
    </HydrationGate>
  );
}
