export interface UserProfile {
  id: string;
  name: string;
  email?: string;
  plan?: "free" | "pro" | "enterprise";
  avatarUrl?: string;
  instructions?: string;
  profession?: string;
  callName?: string;
}

export interface UserSettings {
  theme: "system" | "light" | "dark";
  autoStart: boolean;
  quickShortcut: string;
  systemTray: boolean;
  keepAwake: boolean;
  locationMeta: boolean;
  localEmbeddings: boolean;
  agentPermission: "read" | "sandboxed" | "trusted";
  agentConsentMode: "ask" | "always";
}
