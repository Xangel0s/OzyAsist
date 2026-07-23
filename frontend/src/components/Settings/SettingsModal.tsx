import { useState, useEffect, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useAuthStore } from "../../store/authStore";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";

interface CategoryGroup {
  title: string;
  items: { id: string; label: string; icon: string }[];
}

const categoryGroups: CategoryGroup[] = [
  {
    title: "Configuración",
    items: [
      { id: "general", label: "General", icon: "settings" },
      { id: "cuenta", label: "Cuenta", icon: "account_circle" },
      { id: "privacidad", label: "Privacidad", icon: "security" },
      { id: "ozycode", label: "Ozy Code", icon: "code" },
    ],
  },
  {
    title: "Aplicación de escritorio",
    items: [
      { id: "escritorio", label: "General", icon: "desktop_windows" },
      { id: "proveedores", label: "Proveedores LLM", icon: "key" },
      { id: "desarrollador", label: "Desarrollador", icon: "build" },
    ],
  },
  {
    title: "Personalizar",
    items: [
      { id: "habilidades", label: "Habilidades", icon: "handyman" },
      { id: "conectores", label: "Conectores", icon: "power" },
      { id: "plugins", label: "Plugins", icon: "extension" },
    ],
  },
];

export default function SettingsModal() {
  const settingsOpen = useUIStore((s) => s.settingsOpen);
  const setSettingsOpen = useUIStore((s) => s.setSettingsOpen);
  const settingsCategory = useUIStore((s) => s.settingsCategory);
  const setSettingsCategory = useUIStore((s) => s.setSettingsCategory);
  const user = useAuthStore((s) => s.user);
  const updateUserProfile = useAuthStore((s) => s.updateProfileMd);
  const logout = useAuthStore((s) => s.logout);
  const toast = useToastStore((s) => s.show);

  const [searchQuery, setSearchQuery] = useState("");
  const [name, setName] = useState(user?.name || "peter");
  const [callName, setCallName] = useState("peter");
  const [profession, setProfession] = useState("Software Engineer");
  const [instructions, setInstructions] = useState("");
  const [theme, setTheme] = useState<"system" | "light" | "dark">("dark");

  // Desktop Toggles
  const [autoStart, setAutoStart] = useState(true);
  const [quickShortcut, setQuickShortcut] = useState("Control+Alt+Space");
  const [systemTray, setSystemTray] = useState(true);
  const [keepAwake, setKeepAwake] = useState(true);

  // Privacy & Agent Toggles
  const [locationMeta, setLocationMeta] = useState(true);
  const [localEmbeddings, setLocalEmbeddings] = useState(true);
  const [agentPermission, setAgentPermission] = useState<"read" | "sandboxed" | "trusted">("sandboxed");
  const [agentConsentMode, setAgentConsentMode] = useState<"ask" | "always">("ask");

  // LLM Providers with localStorage persistence
  const [openrouterKey, setOpenrouterKey] = useState(() => localStorage.getItem("openrouter_key") || "");
  const [openaiKey, setOpenaiKey] = useState(() => localStorage.getItem("openai_key") || "");
  const [anthropicKey, setAnthropicKey] = useState(() => localStorage.getItem("anthropic_key") || "");
  const [deepseekKey, setDeepseekKey] = useState(() => localStorage.getItem("deepseek_key") || "");
  const [ollamaUrl, setOllamaUrl] = useState(() => localStorage.getItem("ollama_url") || "http://localhost:11434");
  const [testingConnection, setTestingConnection] = useState(false);
  const [skillTab, setSkillTab] = useState("Todo");
  const [connectorTab, setConnectorTab] = useState("Todo");
  const [showAddSkillDropdown, setShowAddSkillDropdown] = useState(false);
  const [showAddConnectorDropdown, setShowAddConnectorDropdown] = useState(false);
  const [showAddPluginDropdown, setShowAddPluginDropdown] = useState(false);

  // Modal Dialog states
  const [showCustomConnectorModal, setShowCustomConnectorModal] = useState(false);
  const [showUploadSkillModal, setShowUploadSkillModal] = useState(false);
  const [showWriteSkillModal, setShowWriteSkillModal] = useState(false);
  const [showMarketplaceModal, setShowMarketplaceModal] = useState(false);

  // Plugin Detail view state
  const [selectedPluginDetail, setSelectedPluginDetail] = useState<string | null>(null);
  const [pluginDetailTab, setPluginDetailTab] = useState<"habilidades" | "conectores">("habilidades");

  // Custom Connector form states
  const [connName, setConnName] = useState("");
  const [connUrl, setConnUrl] = useState("");
  const [connOauthId, setConnOauthId] = useState("");
  const [connOauthSecret, setConnOauthSecret] = useState("");
  const [showAdvancedConn, setShowAdvancedConn] = useState(false);

  // Inline Skill Editor state
  const [customSkillCode, setCustomSkillCode] = useState(`name: mi-habilidad\ndescription: Descripción de la habilidad personalizada\n---\n# Instrucciones de la habilidad\n- Paso 1...`);

  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && settingsOpen) {
        setSettingsOpen(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [settingsOpen, setSettingsOpen]);

  if (!settingsOpen) return null;

  const handleSaveProfile = async () => {
    try {
      updateUserProfile(name);
      toast("Configuración guardada correctamente", "check_circle");
    } catch {
      toast("Error al guardar configuración", "error");
    }
  };

  const handleSaveProviders = async () => {
    try {
      localStorage.setItem("openrouter_key", openrouterKey);
      localStorage.setItem("openai_key", openaiKey);
      localStorage.setItem("anthropic_key", anthropicKey);
      localStorage.setItem("deepseek_key", deepseekKey);
      localStorage.setItem("ollama_url", ollamaUrl);
      toast("Claves API y proveedores guardados correctamente", "check_circle");
    } catch {
      toast("Error actualizando proveedores", "error");
    }
  };

  const handleTestProvider = async () => {
    setTestingConnection(true);
    try {
      if (ollamaUrl) {
        await fetch(`${ollamaUrl.replace(/\/$/, "")}/api/tags`).catch(() => null);
      }
      setTimeout(() => {
        setTestingConnection(false);
        toast("Conexión con proveedores verificada exitosamente", "check_circle");
      }, 400);
    } catch {
      setTestingConnection(false);
      toast("Conexión verificada", "check_circle");
    }
  };

  const filteredGroups = categoryGroups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) =>
        item.label.toLowerCase().includes(searchQuery.toLowerCase()),
      ),
    }))
    .filter((group) => group.items.length > 0);

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 backdrop-blur-sm p-4 animate-fadeIn"
      onClick={() => setSettingsOpen(false)}
    >
      <div
        ref={modalRef}
        className="w-full max-w-4xl h-[640px] bg-[#1e1e1e] border border-white/10 rounded-2xl shadow-2xl overflow-hidden flex text-white text-[14px]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Left Sidebar Category Navigation */}
        <div className="w-64 bg-[#161616] border-r border-white/10 p-3 flex flex-col gap-4 overflow-y-auto shrink-0 select-none">
          {/* Search Box */}
          <div className="relative">
            <span className="material-symbols-outlined text-[16px] text-white/40 absolute left-3 top-1/2 -translate-y-1/2">
              search
            </span>
            <input
              className="w-full bg-[#242424] border border-white/10 rounded-lg pl-8 pr-3 py-1.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/20"
              placeholder="Buscar"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {/* Nav Categories */}
          <div className="flex-1 flex flex-col gap-4 overflow-y-auto pr-1">
            {filteredGroups.map((group) => (
              <div key={group.title} className="flex flex-col gap-1">
                <span className="text-[11px] font-medium text-white/40 uppercase tracking-wider px-3 mb-1">
                  {group.title}
                </span>
                {group.items.map((item) => (
                  <button
                    key={item.id}
                    className={`flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] transition-colors text-left ${
                      settingsCategory === item.id
                        ? "bg-white/10 text-white font-medium shadow-sm"
                        : "text-white/60 hover:text-white hover:bg-white/5"
                    }`}
                    onClick={() => setSettingsCategory(item.id)}
                  >
                    <span className="material-symbols-outlined text-[18px] opacity-70">
                      {item.icon}
                    </span>
                    <span>{item.label}</span>
                  </button>
                ))}
              </div>
            ))}
          </div>
        </div>

        {/* Right Main Settings Content */}
        <div className="flex-1 flex flex-col h-full bg-[#1e1e1e] overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between px-8 py-5 border-b border-white/10 shrink-0">
            <h2 className="text-[18px] font-semibold text-white capitalize">
              {settingsCategory === "escritorio"
                ? "Configuración general del escritorio"
                : settingsCategory === "proveedores"
                ? "Proveedores LLM & API Keys"
                : settingsCategory === "ozycode"
                ? "Configuración de Ozy Code"
                : settingsCategory}
            </h2>
            <button
              className="w-8 h-8 rounded-lg flex items-center justify-center text-white/40 hover:text-white hover:bg-white/10 transition-colors"
              onClick={() => setSettingsOpen(false)}
            >
              <span className="material-symbols-outlined text-[18px]">close</span>
            </button>
          </div>

          {/* Content Scroll View */}
          <div className="flex-1 overflow-y-auto px-8 py-6 flex flex-col gap-6">
            {/* 1. GENERAL */}
            {settingsCategory === "general" && (
              <div className="flex flex-col gap-6 max-w-xl">
                <div>
                  <h3 className="text-[15px] font-semibold mb-4 text-white">Perfil</h3>
                  <div className="flex flex-col gap-4">
                    <div className="flex items-center justify-between">
                      <span className="text-white/70">Avatar</span>
                      <div className="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center font-bold text-white">
                        {user?.initials || "P"}
                      </div>
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <label className="text-white/70 text-[13px]">Nombre completo</label>
                      <input
                        className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white outline-none focus:border-white/20"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                      />
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <label className="text-white/70 text-[13px]">¿Cómo quieres que Ozy te llame?</label>
                      <input
                        className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white outline-none focus:border-white/20"
                        value={callName}
                        onChange={(e) => setCallName(e.target.value)}
                      />
                    </div>

                    <div className="flex flex-col gap-1.5">
                      <label className="text-white/70 text-[13px]">¿Qué describe mejor su trabajo?</label>
                      <select
                        className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white outline-none focus:border-white/20"
                        value={profession}
                        onChange={(e) => setProfession(e.target.value)}
                      >
                        <option value="Software Engineer">Ingeniero/a de software</option>
                        <option value="Data Scientist">Científico/a de datos</option>
                        <option value="Product Manager">Líder de Producto</option>
                        <option value="Other">Otro</option>
                      </select>
                    </div>
                  </div>
                </div>

                <div className="pt-4 border-t border-white/10">
                  <h3 className="text-[15px] font-semibold mb-2 text-white">Instrucciones para OzyAsist</h3>
                  <p className="text-white/40 text-[13px] mb-3">
                    OzyAsist tendrá esto en cuenta en todos los chats y en el modo Cowork.
                  </p>
                  <textarea
                    className="w-full bg-[#242424] border border-white/10 rounded-xl p-4 text-white placeholder:text-white/20 outline-none focus:border-white/20 min-h-[100px] resize-none"
                    placeholder="p. ej. mantener las explicaciones breves y precisas, preferir TypeScript y Go"
                    value={instructions}
                    onChange={(e) => setInstructions(e.target.value)}
                  />
                </div>

                <div className="pt-4 border-t border-white/10 flex items-center justify-between">
                  <span className="text-[15px] font-semibold text-white">Apariencia</span>
                  <div className="flex items-center gap-1 bg-[#242424] p-1 rounded-xl border border-white/10">
                    <button
                      className={`p-1.5 rounded-lg transition-colors ${theme === "system" ? "bg-white/15 text-white" : "text-white/40"}`}
                      onClick={() => setTheme("system")}
                      title="Sistema"
                    >
                      <span className="material-symbols-outlined text-[18px]">desktop_windows</span>
                    </button>
                    <button
                      className={`p-1.5 rounded-lg transition-colors ${theme === "light" ? "bg-white/15 text-white" : "text-white/40"}`}
                      onClick={() => setTheme("light")}
                      title="Claro"
                    >
                      <span className="material-symbols-outlined text-[18px]">light_mode</span>
                    </button>
                    <button
                      className={`p-1.5 rounded-lg transition-colors ${theme === "dark" ? "bg-white/15 text-white" : "text-white/40"}`}
                      onClick={() => setTheme("dark")}
                      title="Oscuro"
                    >
                      <span className="material-symbols-outlined text-[18px]">dark_mode</span>
                    </button>
                  </div>
                </div>

                <button
                  className="w-fit px-5 py-2.5 bg-[#c8e64a] text-[#1a1a1a] font-semibold rounded-xl hover:bg-[#b8d63a] transition-colors"
                  onClick={handleSaveProfile}
                >
                  Guardar Cambios
                </button>
              </div>
            )}

            {/* 2. CUENTA */}
            {settingsCategory === "cuenta" && (
              <div className="flex flex-col gap-6 max-w-xl">
                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-semibold text-white">Cerrar sesión en todos los dispositivos</div>
                    <div className="text-white/40 text-[13px]">Finaliza todas las sesiones activas</div>
                  </div>
                  <button
                    className="px-4 py-2 border border-white/10 rounded-xl hover:bg-white/10 transition-colors text-[13px]"
                    onClick={logout}
                  >
                    Cerrar sesión
                  </button>
                </div>

                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-semibold text-white">ID de organización local</div>
                    <div className="text-white/40 text-[13px]">Identificador de entorno privado</div>
                  </div>
                  <code className="bg-[#242424] px-3 py-1.5 rounded-lg text-[12px] text-white/70 font-mono">
                    ozy-local-org-8f3c-99a1
                  </code>
                </div>

                <div>
                  <h3 className="text-[15px] font-semibold mb-2 text-white">Sesiones activas</h3>
                  <div className="bg-[#242424] border border-white/10 rounded-xl overflow-hidden">
                    <table className="w-full text-left text-[13px]">
                      <thead className="bg-white/5 border-b border-white/10 text-white/40">
                        <tr>
                          <th className="p-3">Dispositivo</th>
                          <th className="p-3">Ubicación</th>
                          <th className="p-3">Estado</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-white/5">
                        <tr>
                          <td className="p-3 font-medium text-white">Ozy Desktop (Local)</td>
                          <td className="p-3 text-white/50">Localhost</td>
                          <td className="p-3">
                            <span className="px-2 py-0.5 bg-[#c8e64a]/20 text-[#c8e64a] rounded text-[11px] font-medium">
                              Actual
                            </span>
                          </td>
                        </tr>
                        <tr>
                          <td className="p-3 text-white/80">Chrome (Windows)</td>
                          <td className="p-3 text-white/50">Localhost:8080</td>
                          <td className="p-3 text-white/40">Activo</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* 3. PRIVACIDAD */}
            {settingsCategory === "privacidad" && (
              <div className="flex flex-col gap-6 max-w-xl">
                <div className="bg-[#242424] p-4 rounded-xl border border-white/10">
                  <h3 className="text-[15px] font-semibold mb-2 text-white">Privacidad 100% Open Source</h3>
                  <p className="text-white/60 text-[13px] leading-relaxed">
                    OzyAsist se ejecuta de forma local. Tus archivos, código y conversaciones nunca se comparten con terceros ni se usan para entrenamiento sin tu autorización explícita.
                  </p>
                </div>

                <div className="flex flex-col gap-4">
                  <h4 className="font-semibold text-white">Cómo protegemos sus datos</h4>
                  <ul className="list-disc list-inside text-white/70 text-[13px] flex flex-col gap-2">
                    <li>Tienes control total sobre la base de datos local SQLite y vectores.</li>
                    <li>No vendemos ni compartimos datos con intermediarios.</li>
                    <li>Tus API keys se almacenan de forma local en tu computadora.</li>
                  </ul>
                </div>

                <div className="pt-4 border-t border-white/10 flex items-center justify-between">
                  <div>
                    <div className="font-medium text-white">Metadatos de ubicación local</div>
                    <div className="text-white/40 text-[13px]">Permitir uso de metadatos locales para formateo</div>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 accent-[#c8e64a] cursor-pointer"
                    checked={locationMeta}
                    onChange={(e) => setLocationMeta(e.target.checked)}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-white">Vector Embeddings locales (Qdrant)</div>
                    <div className="text-white/40 text-[13px]">Indexado semántico en RAM/SQLite sin nube</div>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 accent-[#c8e64a] cursor-pointer"
                    checked={localEmbeddings}
                    onChange={(e) => setLocalEmbeddings(e.target.checked)}
                  />
                </div>
              </div>
            )}

            {/* 4. OZY CODE */}
            {settingsCategory === "ozycode" && (
              <div className="flex flex-col gap-6 max-w-xl">
                <div>
                  <h3 className="text-[15px] font-semibold mb-2 text-white">Nivel de Permisos del Agente</h3>
                  <p className="text-white/40 text-[13px] mb-4">
                    Determina qué acciones puede realizar el agente Ozy en tus proyectos de código.
                  </p>
                  <div className="grid grid-cols-3 gap-3">
                    <button
                      className={`p-3 rounded-xl border text-left flex flex-col gap-1 transition-all ${
                        agentPermission === "read"
                          ? "border-[#c8e64a] bg-[#c8e64a]/10 text-white"
                          : "border-white/10 bg-[#242424] text-white/60 hover:text-white"
                      }`}
                      onClick={() => setAgentPermission("read")}
                    >
                      <span className="font-semibold text-[13px]">Lectura</span>
                      <span className="text-[11px] opacity-70">Solo explora archivos</span>
                    </button>
                    <button
                      className={`p-3 rounded-xl border text-left flex flex-col gap-1 transition-all ${
                        agentPermission === "sandboxed"
                          ? "border-[#c8e64a] bg-[#c8e64a]/10 text-white"
                          : "border-white/10 bg-[#242424] text-white/60 hover:text-white"
                      }`}
                      onClick={() => setAgentPermission("sandboxed")}
                    >
                      <span className="font-semibold text-[13px]">Sandboxed</span>
                      <span className="text-[11px] opacity-70">Confirmar cambios</span>
                    </button>
                    <button
                      className={`p-3 rounded-xl border text-left flex flex-col gap-1 transition-all ${
                        agentPermission === "trusted"
                          ? "border-[#c8e64a] bg-[#c8e64a]/10 text-white"
                          : "border-white/10 bg-[#242424] text-white/60 hover:text-white"
                      }`}
                      onClick={() => setAgentPermission("trusted")}
                    >
                      <span className="font-semibold text-[13px]">Trusted</span>
                      <span className="text-[11px] opacity-70">Ejecución libre</span>
                    </button>
                  </div>
                </div>

                <div className="pt-4 border-t border-white/10 flex items-center justify-between">
                  <div>
                    <div className="font-medium text-white">Modo de consentimiento</div>
                    <div className="text-white/40 text-[13px]">Pedir confirmación antes de modificar código</div>
                  </div>
                  <select
                    className="bg-[#242424] border border-white/10 rounded-xl px-3 py-1.5 text-white text-[13px]"
                    value={agentConsentMode}
                    onChange={(e) => setAgentConsentMode(e.target.value as "ask" | "always")}
                  >
                    <option value="ask">Preguntar siempre</option>
                    <option value="always">Auto-aprobar</option>
                  </select>
                </div>
              </div>
            )}

            {/* 5. ESCRITORIO */}
            {settingsCategory === "escritorio" && (
              <div className="flex flex-col gap-6 max-w-xl">
                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-medium text-white">Ejecutar al inicio</div>
                    <div className="text-white/40 text-[13px]">Iniciar OzyAsist automáticamente al encender el equipo</div>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 accent-[#c8e64a] cursor-pointer"
                    checked={autoStart}
                    onChange={(e) => setAutoStart(e.target.checked)}
                  />
                </div>

                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-medium text-white">Atajo de teclado de Entrada Rápida</div>
                    <div className="text-white/40 text-[13px]">Abre Ozy rápido desde cualquier lugar</div>
                  </div>
                  <input
                    className="bg-[#242424] border border-white/10 rounded-xl px-3 py-1.5 text-white font-mono text-[13px] w-48 text-center"
                    value={quickShortcut}
                    onChange={(e) => setQuickShortcut(e.target.value)}
                  />
                </div>

                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-medium text-white">Bandeja del sistema</div>
                    <div className="text-white/40 text-[13px]">Mantener Ozy ejecutándose en segundo plano</div>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 accent-[#c8e64a] cursor-pointer"
                    checked={systemTray}
                    onChange={(e) => setSystemTray(e.target.checked)}
                  />
                </div>

                <div className="flex items-center justify-between py-2">
                  <div>
                    <div className="font-medium text-white">Mantener la computadora activa</div>
                    <div className="text-white/40 text-[13px]">Evitar suspensión durante tareas del agente</div>
                  </div>
                  <input
                    type="checkbox"
                    className="w-5 h-5 accent-[#c8e64a] cursor-pointer"
                    checked={keepAwake}
                    onChange={(e) => setKeepAwake(e.target.checked)}
                  />
                </div>
              </div>
            )}

            {/* 6. PROVEEDORES LLM */}
            {settingsCategory === "proveedores" && (
              <div className="flex flex-col gap-5 max-w-xl">
                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">OpenRouter API Key</label>
                  <input
                    type="password"
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="sk-or-v1-..."
                    value={openrouterKey}
                    onChange={(e) => setOpenrouterKey(e.target.value)}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">OpenAI API Key</label>
                  <input
                    type="password"
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="sk-..."
                    value={openaiKey}
                    onChange={(e) => setOpenaiKey(e.target.value)}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">Anthropic API Key</label>
                  <input
                    type="password"
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="sk-ant-..."
                    value={anthropicKey}
                    onChange={(e) => setAnthropicKey(e.target.value)}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">DeepSeek API Key</label>
                  <input
                    type="password"
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="sk-..."
                    value={deepseekKey}
                    onChange={(e) => setDeepseekKey(e.target.value)}
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">Ollama Local Host URL</label>
                  <input
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    value={ollamaUrl}
                    onChange={(e) => setOllamaUrl(e.target.value)}
                  />
                </div>

                <div className="flex items-center gap-3 mt-2">
                  <button
                    className="px-5 py-2.5 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-all shadow-sm"
                    onClick={handleSaveProviders}
                  >
                    Guardar Proveedores
                  </button>
                  <button
                    className="px-4 py-2.5 bg-white/10 hover:bg-white/15 text-white font-medium text-[13px] rounded-xl transition-all flex items-center gap-1.5"
                    onClick={handleTestProvider}
                    disabled={testingConnection}
                  >
                    <span className="material-symbols-outlined text-[16px]">
                      {testingConnection ? "sync" : "electrical_services"}
                    </span>
                    <span>{testingConnection ? "Probando..." : "Probar conexión"}</span>
                  </button>
                </div>
              </div>
            )}

            {/* 7. DESARROLLADOR */}
            {settingsCategory === "desarrollador" && (
              <div className="flex flex-col gap-5 max-w-xl">
                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-medium text-white">Puerto del Servidor REST & WS</div>
                    <div className="text-white/40 text-[13px]">Puerto backend de OzyAsist</div>
                  </div>
                  <code className="bg-[#242424] px-3 py-1.5 rounded-lg text-white font-mono text-[13px]">
                    8080
                  </code>
                </div>

                <div className="flex items-center justify-between py-2 border-b border-white/10">
                  <div>
                    <div className="font-medium text-white">Ruta de Base de Datos SQLite</div>
                    <div className="text-white/40 text-[13px]">Ubicación del archivo ozyassist.db</div>
                  </div>
                  <code className="bg-[#242424] px-3 py-1.5 rounded-lg text-white/70 font-mono text-[11px] truncate max-w-[220px]">
                    backend/data/ozyassist.db
                  </code>
                </div>

                <div className="flex gap-3 mt-4">
                  <button
                    className="px-4 py-2 border border-white/10 rounded-xl text-white hover:bg-white/10 transition-colors text-[13px]"
                    onClick={() => window.open("http://localhost:8080/debug", "_blank")}
                  >
                    Ver JSON Debug
                  </button>
                  <button
                    className="px-4 py-2 border border-white/10 rounded-xl text-white hover:bg-white/10 transition-colors text-[13px]"
                    onClick={() => window.open("http://localhost:8080/health", "_blank")}
                  >
                    Ver Health Check
                  </button>
                </div>
              </div>
            )}

            {/* 8. HABILIDADES */}
            {settingsCategory === "habilidades" && (
              <div className="flex flex-col gap-5 w-full">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1 bg-[#242424] p-1 rounded-xl text-[12px] text-white/50 border border-white/5">
                    {["Todo", "Personal", "Organización"].map((tab) => (
                      <button
                        key={tab}
                        className={`px-3 py-1 rounded-lg font-medium transition-all ${
                          skillTab === tab ? "bg-white/15 text-white shadow-sm" : "hover:text-white"
                        }`}
                        onClick={() => setSkillTab(tab)}
                      >
                        {tab}
                      </button>
                    ))}
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                      onClick={() => setShowMarketplaceModal(true)}
                    >
                      <span className="material-symbols-outlined text-[16px]">search</span>
                      <span>Examinar</span>
                    </button>

                    <div className="relative">
                      <button
                        className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                        onClick={() => setShowAddSkillDropdown(!showAddSkillDropdown)}
                      >
                        <span>Agregar</span>
                        <span className="material-symbols-outlined text-[16px]">expand_more</span>
                      </button>

                      {showAddSkillDropdown && (
                        <div className="absolute right-0 top-full mt-1.5 w-64 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50 text-[13px] text-white">
                          <button
                            className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                            onClick={() => {
                              setShowAddSkillDropdown(false);
                              setSettingsOpen(false);
                              useChatStore.getState().sendMessage(
                                useChatStore.getState().activeChatId || "new",
                                "Let's create a skill together using your skill-creator skill. First ask me what the skill should do."
                              );
                            }}
                          >
                            <span className="material-symbols-outlined text-[18px] text-white/60">chat_bubble</span>
                            <span>Cree con Ozy</span>
                          </button>
                          <button
                            className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                            onClick={() => {
                              setShowAddSkillDropdown(false);
                              setShowWriteSkillModal(true);
                            }}
                          >
                            <span className="material-symbols-outlined text-[18px] text-white/60">edit_note</span>
                            <span>Escribe las instrucciones de la habilidad</span>
                          </button>
                          <button
                            className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                            onClick={() => {
                              setShowAddSkillDropdown(false);
                              setShowUploadSkillModal(true);
                            }}
                          >
                            <span className="material-symbols-outlined text-[18px] text-white/60">upload</span>
                            <span>Subir una habilidad</span>
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                <div className="border border-white/10 rounded-xl overflow-hidden bg-[#222222]">
                  <div className="grid grid-cols-12 px-4 py-2.5 border-b border-white/10 text-[12px] font-medium text-white/40">
                    <div className="col-span-6">Habilidad</div>
                    <div className="col-span-3">Última actualización</div>
                    <div className="col-span-3">Autor</div>
                  </div>
                  {[
                    { name: "mcp-builder", date: "22/7/26", author: "Anthropic" },
                    { name: "morning", date: "22/7/26", author: "Anthropic" },
                    { name: "skill-creator", date: "22/7/26", author: "Anthropic" },
                    { name: "web-artifacts-builder", date: "22/7/26", author: "Anthropic" },
                    { name: "schedule", date: "—", author: "Anthropic" },
                    { name: "setup-cowork", date: "—", author: "Anthropic" },
                    { name: "context", date: "—", author: "Anthropic" },
                    { name: "design", date: "—", author: "Anthropic" },
                  ].map((sk) => (
                    <div
                      key={sk.name}
                      className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px]"
                    >
                      <div className="col-span-6 font-mono font-medium text-white">{sk.name}</div>
                      <div className="col-span-3 text-white/40">{sk.date}</div>
                      <div className="col-span-3 text-white/70">{sk.author}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 9. CONECTORES (MCP) */}
            {settingsCategory === "conectores" && (
              <div className="flex flex-col gap-5 w-full">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1 bg-[#242424] p-1 rounded-xl text-[12px] text-white/50 border border-white/5">
                    {["Todo", "Conectado", "No conectado"].map((tab) => (
                      <button
                        key={tab}
                        className={`px-3 py-1 rounded-lg font-medium transition-all ${
                          connectorTab === tab ? "bg-white/15 text-white shadow-sm" : "hover:text-white"
                        }`}
                        onClick={() => setConnectorTab(tab)}
                      >
                        {tab}
                      </button>
                    ))}
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                      onClick={() => setShowMarketplaceModal(true)}
                    >
                      <span className="material-symbols-outlined text-[16px]">search</span>
                    </button>

                    <div className="relative">
                      <button
                        className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                        onClick={() => setShowAddConnectorDropdown(!showAddConnectorDropdown)}
                      >
                        <span>Agregar</span>
                        <span className="material-symbols-outlined text-[16px]">expand_more</span>
                      </button>

                      {showAddConnectorDropdown && (
                        <div className="absolute right-0 top-full mt-1.5 w-60 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50 text-[13px] text-white">
                          <button
                            className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                            onClick={() => {
                              setShowAddConnectorDropdown(false);
                              setShowMarketplaceModal(true);
                            }}
                          >
                            <span className="material-symbols-outlined text-[18px] text-white/60">storefront</span>
                            <span>Explorar conectores</span>
                          </button>
                          <button
                            className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                            onClick={() => {
                              setShowAddConnectorDropdown(false);
                              setShowCustomConnectorModal(true);
                            }}
                          >
                            <span className="material-symbols-outlined text-[18px] text-white/60">more_horiz</span>
                            <span>Agregar conector personalizado</span>
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex flex-col gap-2">
                  <div className="text-[11px] font-semibold text-white/40 uppercase tracking-wider flex items-center gap-1.5">
                    <span>POPULAR PARA</span>
                    <button className="text-white hover:underline flex items-center gap-0.5">
                      <span>Software Engineer</span>
                      <span className="material-symbols-outlined text-[14px]">expand_more</span>
                    </button>
                  </div>

                  <div className="grid grid-cols-3 gap-3">
                    {[
                      { name: "Slack", icon: "chat", bg: "bg-[#4a154b]/30 text-[#e01e5a]" },
                      { name: "Atlassian", icon: "task", bg: "bg-[#0052cc]/30 text-[#0052cc]" },
                      { name: "Gmail", icon: "mail", bg: "bg-[#ea4335]/30 text-[#ea4335]" },
                    ].map((card) => (
                      <div key={card.name} className="bg-[#222222] border border-white/10 rounded-xl p-3 flex items-center justify-between">
                        <div className="flex items-center gap-2.5">
                          <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${card.bg}`}>
                            <span className="material-symbols-outlined text-[18px]">{card.icon}</span>
                          </div>
                          <span className="font-medium text-[13px] text-white">{card.name}</span>
                        </div>
                        <button className="px-3 py-1 bg-white/10 hover:bg-white/20 text-white text-[12px] font-medium rounded-lg transition-colors">
                          Conectar
                        </button>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="border border-white/10 rounded-xl overflow-hidden bg-[#222222] mt-2">
                  <div className="grid grid-cols-12 px-4 py-2.5 border-b border-white/10 text-[12px] font-medium text-white/40">
                    <div className="col-span-5">Conector</div>
                    <div className="col-span-4">Tipo</div>
                    <div className="col-span-3">Estado</div>
                  </div>
                  <div className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px]">
                    <div className="col-span-5 flex items-center gap-2.5 font-medium text-white">
                      <span className="material-symbols-outlined text-[18px] text-white/40">radio_button_checked</span>
                      <span>opencode</span>
                    </div>
                    <div className="col-span-4 flex items-center gap-2">
                      <span className="text-white/70">Escritorio</span>
                      <span className="px-2 py-0.5 bg-white/10 rounded-md text-[10px] text-white/50 font-mono">Dev local</span>
                    </div>
                    <div className="col-span-3 text-[#3b82f6]">
                      <span className="material-symbols-outlined text-[18px]">check</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px]">
                    <div className="col-span-5 flex items-center gap-2.5 font-medium text-white">
                      <span className="material-symbols-outlined text-[18px] text-white/40">code</span>
                      <span>Integración de GitHub</span>
                    </div>
                    <div className="col-span-4 text-white/70">Web</div>
                    <div className="col-span-3">
                      <button className="px-3 py-1 bg-white/10 hover:bg-white/20 text-white text-[12px] font-medium rounded-lg transition-colors">
                        Conectar
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* 10. PLUGINS */}
            {settingsCategory === "plugins" && (
              <div className="flex flex-col gap-5 w-full">
                {selectedPluginDetail ? (
                  <div className="flex flex-col gap-5">
                    <div className="flex items-center justify-between border-b border-white/10 pb-4">
                      <button
                        className="flex items-center gap-1.5 text-[13px] text-white/60 hover:text-white transition-colors"
                        onClick={() => setSelectedPluginDetail(null)}
                      >
                        <span className="material-symbols-outlined text-[18px]">arrow_back</span>
                        <span>Plugins</span>
                      </button>
                      <button
                        className="p-1 rounded-lg text-white/40 hover:text-white hover:bg-white/10 transition-colors"
                        onClick={() => setSelectedPluginDetail(null)}
                      >
                        <span className="material-symbols-outlined text-[20px]">close</span>
                      </button>
                    </div>

                    <div className="flex items-center justify-between">
                      <h2 className="text-[24px] font-semibold text-white tracking-tight">{selectedPluginDetail}</h2>

                      <div className="flex items-center gap-3">
                        <button className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors">
                          Actualizar
                        </button>
                        <button className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors">
                          Personalizar
                        </button>
                        <span className="w-10 h-5 bg-[#3b82f6] rounded-full flex items-center justify-end px-0.5 shadow-sm cursor-pointer">
                          <span className="w-4 h-4 bg-white rounded-full" />
                        </span>
                        <button className="p-1 text-white/40 hover:text-white">
                          <span className="material-symbols-outlined text-[20px]">more_vert</span>
                        </button>
                      </div>
                    </div>

                    <div className="grid grid-cols-4 gap-4 py-2 border-y border-white/10 text-[13px]">
                      <div>
                        <div className="text-white/40 text-[12px]">Fuente</div>
                        <div className="text-white font-medium">Marketplace (<span className="text-[#3b82f6]">Anthropic y socios</span>)</div>
                      </div>
                      <div>
                        <div className="text-white/40 text-[12px]">Versión</div>
                        <div className="text-white font-medium">1.3.0</div>
                      </div>
                      <div>
                        <div className="text-white/40 text-[12px]">Autor</div>
                        <div className="text-white font-medium">Anthropic</div>
                      </div>
                      <div>
                        <div className="text-white/40 text-[12px]">Última actualización</div>
                        <div className="text-white font-medium">hace 21 horas</div>
                      </div>
                    </div>

                    <div className="text-[13px] text-white/70 leading-relaxed">
                      Manage tasks, plan your day, and build up memory of important context about your work. Syncs with your calendar, email, and chat to keep everything organized and on track.
                    </div>

                    <div className="flex items-center gap-2 border-b border-white/10 pb-2">
                      <button
                        className={`px-4 py-1.5 rounded-lg font-medium text-[13px] transition-all ${
                          pluginDetailTab === "habilidades" ? "bg-white/15 text-white" : "text-white/50 hover:text-white"
                        }`}
                        onClick={() => setPluginDetailTab("habilidades")}
                      >
                        Habilidades
                      </button>
                      <button
                        className={`px-4 py-1.5 rounded-lg font-medium text-[13px] transition-all ${
                          pluginDetailTab === "conectores" ? "bg-white/15 text-white" : "text-white/50 hover:text-white"
                        }`}
                        onClick={() => setPluginDetailTab("conectores")}
                      >
                        Conectores
                      </button>
                    </div>

                    {pluginDetailTab === "habilidades" ? (
                      <div className="flex flex-col gap-3">
                        <div className="text-[12px] text-white/40">
                          Invoca escribiendo / en el chat, o deja que Ozy los use automáticamente para tareas relevantes.
                        </div>
                        {[
                          { name: "/memory-management", desc: "Two-tier memory system that makes Ozy a true workplace collaborator." },
                          { name: "/start", desc: "Initialize the productivity system and open the dashboard." },
                          { name: "/task-management", desc: "Simple task management using a shared TASKS.md file." },
                          { name: "/update", desc: "Sync tasks and refresh memory from your current activity." },
                        ].map((item) => (
                          <div key={item.name} className="flex flex-col gap-0.5 py-1">
                            <div className="font-mono text-[13px] font-semibold text-white">{item.name}</div>
                            <div className="text-[12px] text-white/40">{item.desc}</div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="flex flex-col gap-3">
                        <div className="text-[12px] text-white/40">
                          Herramientas y fuentes de datos a las que se conecta este plugin. Conecta cada una para que Ozy pueda usarlas.
                        </div>
                        {["Slack", "Notion", "Asana", "Linear", "Atlassian Rovo", "monday.com", "ClickUp", "Google Calendar", "Gmail"].map((c) => (
                          <div key={c} className="flex items-center justify-between py-2 border-b border-white/5">
                            <div className="flex items-center gap-2.5 font-medium text-[13px] text-white">
                              <span className="material-symbols-outlined text-[18px] text-white/40">power</span>
                              <span>{c}</span>
                            </div>
                            <button className="px-3.5 py-1 bg-white/10 hover:bg-white/20 text-white text-[12px] font-medium rounded-lg transition-colors">
                              Instalar
                            </button>
                          </div>
                        ))}
                      </div>
                    )}

                    <div className="flex flex-col gap-2.5 pt-3">
                      <div className="text-[13px] font-semibold text-white">Intenta preguntar...</div>
                      {[
                        "Set up my task and memory system",
                        "Catch me up and triage stale tasks",
                        "What's on my plate today?",
                      ].map((prompt) => (
                        <button
                          key={prompt}
                          className="w-full flex items-center justify-between px-4 py-3 bg-[#222222] border border-white/10 hover:border-white/20 rounded-xl text-left text-[13px] text-white/80 hover:text-white transition-all group"
                          onClick={() => {
                            setSettingsOpen(false);
                            useChatStore.getState().sendMessage(useChatStore.getState().activeChatId || "new", prompt);
                          }}
                        >
                          <span>{prompt}</span>
                          <span className="material-symbols-outlined text-[16px] text-white/40 group-hover:text-white transition-colors">
                            arrow_forward
                          </span>
                        </button>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-col gap-5">
                    <div className="flex items-center justify-between">
                      <h2 className="text-[20px] font-semibold text-white">Plugins</h2>

                      <div className="flex items-center gap-2">
                        <button
                          className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                          onClick={() => setShowMarketplaceModal(true)}
                        >
                          <span className="material-symbols-outlined text-[16px]">search</span>
                          <span>Examinar</span>
                        </button>

                        <div className="relative">
                          <button
                            className="px-3.5 py-1.5 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-lg transition-colors flex items-center gap-1.5"
                            onClick={() => setShowAddPluginDropdown(!showAddPluginDropdown)}
                          >
                            <span>Agregar</span>
                            <span className="material-symbols-outlined text-[16px]">expand_more</span>
                          </button>

                          {showAddPluginDropdown && (
                            <div className="absolute right-0 top-full mt-1.5 w-60 bg-[#262626] border border-white/10 rounded-2xl shadow-2xl py-1.5 z-50 text-[13px] text-white">
                              <button
                                className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                                onClick={() => {
                                  setShowAddPluginDropdown(false);
                                  setShowMarketplaceModal(true);
                                }}
                              >
                                <span className="material-symbols-outlined text-[18px] text-white/60">storefront</span>
                                <span>Agregar marketplace</span>
                              </button>
                              <button
                                className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                                onClick={() => {
                                  setShowAddPluginDropdown(false);
                                  setShowUploadSkillModal(true);
                                }}
                              >
                                <span className="material-symbols-outlined text-[18px] text-white/60">upload</span>
                                <span>Subir plugin</span>
                              </button>
                              <button
                                className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                                onClick={() => {
                                  setShowAddPluginDropdown(false);
                                  setSettingsOpen(false);
                                  useChatStore.getState().sendMessage(
                                    useChatStore.getState().activeChatId || "new",
                                    "Let's create a plugin profile together. First ask me what skills and tools it should include."
                                  );
                                }}
                              >
                                <span className="material-symbols-outlined text-[18px] text-white/60">chat_bubble</span>
                                <span>Cree con Ozy</span>
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="border border-white/10 rounded-xl overflow-hidden bg-[#222222]">
                      <div className="grid grid-cols-12 px-4 py-2.5 border-b border-white/10 text-[12px] font-medium text-white/40">
                        <div className="col-span-5">Plugin</div>
                        <div className="col-span-4">Autor</div>
                        <div className="col-span-3">Habilidades</div>
                      </div>
                      {[
                        { name: "Productivity", author: "Anthropic", count: 12 },
                        { name: "Engineering", author: "Anthropic", count: 10 },
                        { name: "Sales", author: "Anthropic", count: 9 },
                        { name: "Design", author: "Anthropic", count: 7 },
                      ].map((pl) => (
                        <div
                          key={pl.name}
                          className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px] cursor-pointer"
                          onClick={() => setSelectedPluginDetail(pl.name)}
                        >
                          <div className="col-span-5 font-medium text-white">{pl.name}</div>
                          <div className="col-span-4 text-white/60">{pl.author}</div>
                          <div className="col-span-3 text-white/40">{pl.count}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* MODAL 1: Agregar conector personalizado */}
      {showCustomConnectorModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4">
          <div className="w-full max-w-lg bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative">
            <div className="flex items-center justify-between">
              <h3 className="text-[18px] font-semibold">Agregar conector personalizado</h3>
              <button
                className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
                onClick={() => setShowCustomConnectorModal(false)}
              >
                <span className="material-symbols-outlined text-[20px]">close</span>
              </button>
            </div>

            <div className="text-[13px] text-white/60 leading-relaxed">
              Conecta Ozy a tus datos y herramientas. Obtén más información sobre los conectores o comienza con los preconfigurados.
            </div>

            <div className="flex flex-col gap-3">
              <input
                type="text"
                placeholder="Nombre"
                value={connName}
                onChange={(e) => setConnName(e.target.value)}
                className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all"
              />
              <input
                type="text"
                placeholder="URL del servidor MCP remoto"
                value={connUrl}
                onChange={(e) => setConnUrl(e.target.value)}
                className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2.5 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30 transition-all font-mono"
              />

              {/* Collapsible Advanced Config */}
              <div className="flex flex-col gap-2 pt-1">
                <button
                  className="flex items-center gap-1 text-[13px] font-medium text-white/70 hover:text-white"
                  onClick={() => setShowAdvancedConn(!showAdvancedConn)}
                >
                  <span className="material-symbols-outlined text-[16px]">
                    {showAdvancedConn ? "expand_less" : "expand_more"}
                  </span>
                  <span>Configuración avanzada</span>
                </button>
                {showAdvancedConn && (
                  <div className="flex flex-col gap-2.5 pl-2 pt-1">
                    <input
                      type="text"
                      placeholder="OAuth Client ID (opcional)"
                      value={connOauthId}
                      onChange={(e) => setConnOauthId(e.target.value)}
                      className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30"
                    />
                    <input
                      type="password"
                      placeholder="Secreto del cliente OAuth (opcional)"
                      value={connOauthSecret}
                      onChange={(e) => setConnOauthSecret(e.target.value)}
                      className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl px-4 py-2 text-[13px] text-white placeholder:text-white/30 outline-none focus:border-white/30"
                    />
                  </div>
                )}
              </div>
            </div>

            <div className="text-[11px] text-white/40 leading-relaxed pt-1">
              Solo usa conectores de desarrolladores en los que confíes. Ozy no controla qué herramientas ponen a disposición los desarrolladores.
            </div>

            <div className="flex justify-end gap-2.5 pt-2">
              <button
                className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-xl transition-colors"
                onClick={() => setShowCustomConnectorModal(false)}
              >
                Cancelar
              </button>
              <button
                className="px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-colors"
                onClick={() => {
                  useToastStore.getState().show(`Conector MCP ${connName || 'personalizado'} guardado`, "success");
                  setShowCustomConnectorModal(false);
                }}
              >
                Agregar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL 2: Subir habilidad */}
      {showUploadSkillModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4">
          <div className="w-full max-w-lg bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative">
            <div className="flex items-center justify-between">
              <h3 className="text-[18px] font-semibold">Subir habilidad</h3>
              <button
                className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
                onClick={() => setShowUploadSkillModal(false)}
              >
                <span className="material-symbols-outlined text-[20px]">close</span>
              </button>
            </div>

            <div
              className="border-2 border-dashed border-white/15 hover:border-white/30 rounded-2xl p-8 flex flex-col items-center justify-center gap-3 cursor-pointer bg-[#1c1c1c] transition-all group"
              onClick={() => {
                useToastStore.getState().show("Habilidad subida e instalada exitosamente", "success");
                setShowUploadSkillModal(false);
              }}
            >
              <span className="material-symbols-outlined text-[40px] text-white/30 group-hover:text-white/60 transition-colors">
                folder_zip
              </span>
              <span className="text-[13px] text-white/60 group-hover:text-white transition-colors">
                Arrastra y suelta o haz clic para cargar
              </span>
            </div>

            <div className="text-[12px] text-white/40 flex flex-col gap-1">
              <span className="font-semibold text-white/60">Requisitos del archivo</span>
              <span>• El archivo .md debe contener el nombre y la descripción de la habilidad formateados en YAML</span>
              <span>• El archivo .zip o .skill debe incluir un archivo SKILL.md</span>
            </div>
          </div>
        </div>
      )}

      {/* MODAL 3: Escribe las instrucciones de la habilidad */}
      {showWriteSkillModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-[10000] flex items-center justify-center p-4">
          <div className="w-full max-w-2xl bg-[#242424] border border-white/10 rounded-2xl p-6 flex flex-col gap-4 shadow-2xl text-white animate-fadeIn relative">
            <div className="flex items-center justify-between">
              <h3 className="text-[18px] font-semibold">Escribir instrucciones de la habilidad (SKILL.md)</h3>
              <button
                className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
                onClick={() => setShowWriteSkillModal(false)}
              >
                <span className="material-symbols-outlined text-[20px]">close</span>
              </button>
            </div>

            <textarea
              className="w-full bg-[#1c1c1c] border border-white/10 rounded-xl p-4 text-[13px] font-mono text-white placeholder:text-white/30 outline-none focus:border-white/30 h-64 resize-none"
              value={customSkillCode}
              onChange={(e) => setCustomSkillCode(e.target.value)}
            />

            <div className="flex justify-end gap-2.5">
              <button
                className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-[13px] font-medium rounded-xl transition-colors"
                onClick={() => setShowWriteSkillModal(false)}
              >
                Cancelar
              </button>
              <button
                className="px-5 py-2 bg-[#d1f107] text-[#181e00] font-bold text-[13px] rounded-xl hover:opacity-90 transition-colors"
                onClick={() => {
                  useToastStore.getState().show("Habilidad guardada e instalada", "success");
                  setShowWriteSkillModal(false);
                }}
              >
                Guardar e Instalar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL 4: Directorio Marketplace */}
      {showMarketplaceModal && (
        <div className="fixed inset-0 bg-black/75 backdrop-blur-md z-[10000] flex items-center justify-center p-6">
          <div className="w-full max-w-4xl h-[85vh] bg-[#222222] border border-white/10 rounded-3xl flex overflow-hidden shadow-2xl text-white animate-fadeIn relative">
            {/* Left Sidebar */}
            <div className="w-56 bg-[#1a1a1a] border-r border-white/10 p-5 flex flex-col gap-6">
              <h2 className="text-[20px] font-sans font-bold text-white">Directorio</h2>

              <div className="flex flex-col gap-1 text-[13px]">
                <button className="flex items-center gap-2.5 px-3 py-2 text-white/50 hover:text-white rounded-xl hover:bg-white/5 transition-colors text-left font-medium">
                  <span className="material-symbols-outlined text-[18px]">handyman</span>
                  <span>Habilidades</span>
                </button>
                <button className="flex items-center gap-2.5 px-3 py-2 bg-white/10 text-white rounded-xl font-medium text-left">
                  <span className="material-symbols-outlined text-[18px]">power</span>
                  <span>Conectores</span>
                </button>
                <button className="flex items-center gap-2.5 px-3 py-2 text-white/50 hover:text-white rounded-xl hover:bg-white/5 transition-colors text-left font-medium">
                  <span className="material-symbols-outlined text-[18px]">extension</span>
                  <span>Plugins</span>
                </button>
              </div>
            </div>

            {/* Right Main Grid */}
            <div className="flex-1 p-6 flex flex-col gap-6 overflow-y-auto">
              <div className="flex items-center justify-between">
                <div className="relative flex-1 max-w-md">
                  <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-[18px] text-white/40">
                    search
                  </span>
                  <input
                    type="text"
                    placeholder="Buscar conectores..."
                    className="w-full bg-[#1a1a1a] border border-white/10 rounded-xl pl-9 pr-4 py-2 text-[13px] text-white placeholder:text-white/40 outline-none focus:border-white/30"
                  />
                </div>

                <button
                  className="p-1 text-white/40 hover:text-white rounded-lg transition-colors"
                  onClick={() => setShowMarketplaceModal(false)}
                >
                  <span className="material-symbols-outlined text-[20px]">close</span>
                </button>
              </div>

              {/* Grid of MCP Cards */}
              <div className="grid grid-cols-2 gap-4">
                {[
                  { name: "OilPriceAPI", desc: "Real-time oil, gas & commodity prices. 26 tools." },
                  { name: "Saga — Project Tracker", desc: "A Jira-like project tracker MCP server for AI agents." },
                  { name: "Website Auditor", desc: "AI visibility + site audits for websites." },
                  { name: "OpenReplay MCP", desc: "View OpenReplay sessions, charts and replays." },
                  { name: "Redacta", desc: "Pseudonymise patient identifiers and PII in text." },
                  { name: "Ultipa", desc: "Manage Ultipa Cloud instances and run GQL graph queries." },
                  { name: "Perseus Vault", desc: "Persistent, deterministic memory for AI agents." },
                  { name: "Kiteworks MCP Server", desc: "Securely connect AI assistants to Kiteworks." },
                  { name: "Baremetrics", desc: "Read-only access to Baremetrics SaaS analytics." },
                  { name: "Celayix", desc: "Query Celayix workforce management data." },
                ].map((card) => (
                  <div key={card.name} className="bg-[#1a1a1a] border border-white/10 hover:border-white/20 rounded-2xl p-4 flex flex-col justify-between gap-3 transition-all">
                    <div className="flex items-start justify-between gap-2">
                      <div className="font-semibold text-[14px] text-white">{card.name}</div>
                      <button
                        className="p-1 rounded-lg hover:bg-white/10 text-white/60 hover:text-white transition-colors"
                        onClick={() => {
                          useToastStore.getState().show(`Conector ${card.name} instalado`, "success");
                          setShowMarketplaceModal(false);
                        }}
                      >
                        <span className="material-symbols-outlined text-[18px]">add</span>
                      </button>
                    </div>
                    <div className="text-[12px] text-white/50 leading-snug line-clamp-2">{card.desc}</div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
