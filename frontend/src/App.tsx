import { useCallback, useEffect, useState } from "react";
import { useUIStore } from "./store/uiStore";
import { useAuthStore } from "./store/authStore";
import { useKeyboard } from "./hooks";
import TopAppBar from "./components/Layout/TopAppBar";
import Sidebar from "./components/Layout/Sidebar";
import HomePage from "./components/Home/HomePage";
import CodePage from "./components/Code/CodePage";
import ChatPage from "./components/Chat/ChatPage";
import ProjectsPage from "./components/ProjectPanel/ProjectsPage";
import SkillsPage from "./components/Skills/SkillsPage";
import ConnectorsPage from "./components/Connectors/ConnectorsPage";
import OnboardingPage from "./components/Onboarding/OnboardingPage";
import WelcomeScreen from "./components/Onboarding/WelcomeScreen";
import SearchModal from "./components/Search/SearchModal";
import Toast from "./components/Common/Toast";
import ErrorBoundary from "./components/Common/ErrorBoundary";

import SettingsModal from "./components/Settings/SettingsModal";

const pageMap: Record<string, React.ComponentType> = {
  home: HomePage,
  code: CodePage,
  chat: ChatPage,
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
  const activeView = useUIStore((s) => s.activeView);
  const setSearchOpen = useUIStore((s) => s.setSearchOpen);
  const toggleSearch = useCallback(() => setSearchOpen(true), [setSearchOpen]);

  useKeyboard("k", toggleSearch, { meta: true });

  if (!user) {
    return (
      <HydrationGate>
        <WelcomeScreen />
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
      <SearchModal />
      <SettingsModal />
      <Toast />
    </HydrationGate>
  );
}
