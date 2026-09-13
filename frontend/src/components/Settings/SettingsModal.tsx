import { useState, useEffect, useRef } from "react";
import { useUIStore } from "../../store/uiStore";
import { useAuthStore } from "../../store/authStore";
import { useChatStore } from "../../store/chatStore";
import { useToastStore } from "../../store/toastStore";
import { useSkillsStore } from "../../store/skillsStore";
import AddConnectorModal from "./modals/AddConnectorModal";
import UploadSkillModal from "./modals/UploadSkillModal";
import WriteSkillModal from "./modals/WriteSkillModal";
import MarketplaceModal from "./modals/MarketplaceModal";
import SecurityAuditTab from "./tabs/SecurityAuditTab";
import { MCPConnectorsList } from "./MCPConnectorsList";

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
      { id: "seguridad", label: "Seguridad & EDR", icon: "verified_user" },
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

  const normalizedCategory =
    settingsCategory === "models" || settingsCategory === "providers"
      ? "proveedores"
      : settingsCategory === "skills"
      ? "habilidades"
      : settingsCategory === "connectors"
      ? "conectores"
      : settingsCategory || "general";

  const user = useAuthStore((s) => s.user);
  const updateUserProfile = useAuthStore((s) => s.updateProfileMd);
  const logout = useAuthStore((s) => s.logout);
  const toast = useToastStore((s) => s.show);

  const [searchQuery, setSearchQuery] = useState("");
  const [name, setName] = useState(user?.name || "peter");
  const [callName, setCallName] = useState("peter");
  const [profession, setProfession] = useState("Software Engineer");
  const [instructions, setInstructions] = useState("");
  const theme = useUIStore((s) => s.theme);
  const setTheme = useUIStore((s) => s.setTheme);

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
  const [opencodeKey, setOpencodeKey] = useState(() => localStorage.getItem("opencode_key") || "");
  const [openrouterKey, setOpenrouterKey] = useState(() => localStorage.getItem("openrouter_key") || "");
  const [openaiKey, setOpenaiKey] = useState(() => localStorage.getItem("openai_key") || "");
  const [anthropicKey, setAnthropicKey] = useState(() => localStorage.getItem("anthropic_key") || "");
  const [deepseekKey, setDeepseekKey] = useState(() => localStorage.getItem("deepseek_key") || "");
  const [ollamaUrl, setOllamaUrl] = useState(() => localStorage.getItem("ollama_url") || "http://localhost:11434");
  const [testingConnection, setTestingConnection] = useState(false);
  const skills = useSkillsStore((s) => s.skills);
  const removeSkill = useSkillsStore((s) => s.removeSkill);

  // Modals & Navigation
  const [skillTab, setSkillTab] = useState("Todo");
  const [showAddSkillDropdown, setShowAddSkillDropdown] = useState(false);
  const [showAddPluginDropdown, setShowAddPluginDropdown] = useState(false);

  // Modal Dialog states
  const [showCustomConnectorModal, setShowCustomConnectorModal] = useState(false);
  const [showUploadSkillModal, setShowUploadSkillModal] = useState(false);
  const [showWriteSkillModal, setShowWriteSkillModal] = useState(false);
  const [showMarketplaceModal, setShowMarketplaceModal] = useState(false);
  const [editingSkillCode, setEditingSkillCode] = useState("");

  // Skill Detail view state
  const [selectedSkillDetail, setSelectedSkillDetail] = useState<{
    id: string;
    name: string;
    description: string;
    author: string;
    date: string;
    custom: boolean;
    template: string;
  } | null>(null);
  const [skillViewMode, setSkillViewMode] = useState<"preview" | "code">("preview");
  const [showSkillMenu, setShowSkillMenu] = useState(false);
  const [showPluginMenu, setShowPluginMenu] = useState(false);

  // Plugin Detail view state
  const [selectedPluginDetail, setSelectedPluginDetail] = useState<string | null>(null);
  const [pluginDetailTab, setPluginDetailTab] = useState<"habilidades" | "conectores">("habilidades");


  // Active / Inactive toggle states connected to global skillsStore
  const disabledSkillsMap = useSkillsStore((s) => s.disabledSkillsMap);
  const toggleStoreSkill = useSkillsStore((s) => s.toggleSkillEnabled);

  const isSkillEnabled = (name: string) => disabledSkillsMap[name] !== true;
  const toggleSkillEnabled = (name: string) => {
    toggleStoreSkill(name);
    const nextState = disabledSkillsMap[name] === true;
    toast(`Habilidad /${name} ${nextState ? "activada" : "desactivada"}`, "info");
  };

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
      localStorage.setItem("opencode_key", opencodeKey);
      localStorage.setItem("openrouter_key", openrouterKey);
      localStorage.setItem("openai_key", openaiKey);
      localStorage.setItem("anthropic_key", anthropicKey);
      localStorage.setItem("deepseek_key", deepseekKey);
      localStorage.setItem("ollama_url", ollamaUrl);

      const { api } = await import("../../services/api");
      await api.settings.update({
        opencode_key: opencodeKey,
        openai_key: openaiKey,
        openrouter_key: openrouterKey,
        anthropic_key: anthropicKey,
        deepseek_key: deepseekKey,
        local_host_url: ollamaUrl,
        ollama_url: ollamaUrl,
      }).catch(() => {});

      await useChatStore.getState().loadProviders();

      toast("Claves API y Host Local guardados correctamente", "check_circle");
    } catch {
      toast("Error actualizando proveedores", "error");
    }
  };

  const handleTestProvider = async () => {
    setTestingConnection(true);
    try {
      const { api } = await import("../../services/api");
      // Sincronizar primero las claves y host local con el backend
      await api.settings.update({
        opencode_key: opencodeKey,
        openai_key: openaiKey,
        openrouter_key: openrouterKey,
        anthropic_key: anthropicKey,
        deepseek_key: deepseekKey,
        local_host_url: ollamaUrl,
        ollama_url: ollamaUrl,
      }).catch(() => {});

      // Consultar modelos disponibles para verificar la conexión
      const res = await fetch("http://localhost:8080/api/models/available");
      const modelsData = await res.json();
      setTestingConnection(false);

      if (Array.isArray(modelsData) && modelsData.length > 0) {
        const modelNames = modelsData.slice(0, 3).map((m: any) => m.name || m.id).join(", ");
        const more = modelsData.length > 3 ? ` (+${modelsData.length - 3} más)` : "";
        toast(`Conexión exitosa. Modelos detectados: ${modelNames}${more}`, "check_circle");
      } else {
        toast("No se detectaron modelos activos. Verifica que LM Studio u Ollama esté corriendo o que las API Keys sean válidas.", "error");
      }
    } catch (err: any) {
      setTestingConnection(false);
      toast(`Error al probar conexión: ${err?.message || 'Verifica el host local y claves API'}`, "error");
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
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 p-4 transition-opacity"
      onClick={() => setSettingsOpen(false)}
    >
      <div
        ref={modalRef}
        className="w-full max-w-4xl h-[640px] bg-[#1e1e1e] border border-white/10 rounded-2xl shadow-2xl overflow-hidden flex text-white text-[14px] transform-gpu"
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
                      normalizedCategory === item.id
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
              {normalizedCategory === "escritorio"
                ? "Configuración general del escritorio"
                : normalizedCategory === "proveedores"
                ? "Proveedores LLM & API Keys"
                : normalizedCategory === "ozycode"
                ? "Configuración de Ozy Code"
                : normalizedCategory === "seguridad"
                ? "Seguridad EDR & Auditoría Criptográfica"
                : normalizedCategory === "habilidades"
                ? "Habilidades & Skills"
                : normalizedCategory === "conectores"
                ? "Conectores MCP"
                : normalizedCategory === "plugins"
                ? "Plugins de Sistema"
                : normalizedCategory}
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
            {normalizedCategory === "general" && (
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
            {normalizedCategory === "cuenta" && (
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
            {normalizedCategory === "privacidad" && (
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

            {/* SEGURIDAD & EDR */}
            {normalizedCategory === "seguridad" && (
              <SecurityAuditTab />
            )}

            {/* 4. OZY CODE */}
            {normalizedCategory === "ozycode" && (
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
            {normalizedCategory === "escritorio" && (
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
            {normalizedCategory === "proveedores" && (
              <div className="flex flex-col gap-5 max-w-xl">
                <div className="flex flex-col gap-1.5">
                  <label className="font-medium text-white text-[13px]">OpenCode API Key</label>
                  <input
                    type="password"
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="sk-..."
                    value={opencodeKey}
                    onChange={(e) => setOpencodeKey(e.target.value)}
                  />
                </div>

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
                  <div className="flex items-center justify-between">
                    <label className="font-medium text-white text-[13px]">Host Local (LM Studio / Ollama / LocalAI / vLLM)</label>
                    <span className="text-[11px] text-[#d1f107] font-mono">http://100.73.72.32:1234</span>
                  </div>
                  <input
                    className="bg-[#242424] border border-white/10 rounded-xl px-4 py-2 text-white placeholder:text-white/20 outline-none focus:border-white/20 font-mono text-[13px]"
                    placeholder="http://100.73.72.32:1234 o http://localhost:11434"
                    value={ollamaUrl}
                    onChange={(e) => setOllamaUrl(e.target.value)}
                  />
                  <p className="text-[11px] text-white/40">
                    Detecta automáticamente modelos cargados en LM Studio (vía /v1/models) u Ollama (vía /api/tags).
                  </p>
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
            {normalizedCategory === "desarrollador" && (
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
            {normalizedCategory === "habilidades" && (
              <div className="flex flex-col gap-5 w-full">
                {selectedSkillDetail ? (
                  <div className="flex flex-col gap-5">
                    {/* Top Nav */}
                    <div className="flex items-center justify-between border-b border-white/10 pb-4">
                      <button
                        className="flex items-center gap-1.5 text-[13px] text-white/60 hover:text-white transition-colors"
                        onClick={() => setSelectedSkillDetail(null)}
                      >
                        <span className="material-symbols-outlined text-[18px]">arrow_back</span>
                        <span>Habilidades</span>
                      </button>
                    </div>

                    {/* Title and Controls */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <h2 className="text-[24px] font-semibold text-white tracking-tight">{selectedSkillDetail.name}</h2>
                        <span className="material-symbols-outlined text-[18px] text-white/40 cursor-pointer hover:text-white/80" title="Información de la habilidad">info</span>
                      </div>

                      <div className="flex items-center gap-3 relative">
                        <button
                          className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors"
                          onClick={() => {
                            toast(`Buscando actualizaciones para /${selectedSkillDetail.name}...`, "info");
                            setTimeout(() => {
                              toast(`/${selectedSkillDetail.name} ya está en la versión más reciente.`, "success");
                            }, 800);
                          }}
                        >
                          Actualizar
                        </button>
                        <button
                          className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors"
                          onClick={() => {
                            setEditingSkillCode(
                              selectedSkillDetail.template ||
                              `name: ${selectedSkillDetail.name}\ndescription: ${selectedSkillDetail.description}\n---\n# Instrucciones`
                            );
                            setShowWriteSkillModal(true);
                          }}
                        >
                          Personalizar
                        </button>
                        <button
                          className={`w-10 h-5 rounded-full flex items-center px-0.5 shadow-sm transition-all cursor-pointer ${
                            isSkillEnabled(selectedSkillDetail.name) ? "bg-[#3b82f6] justify-end" : "bg-white/20 justify-start"
                          }`}
                          onClick={() => toggleSkillEnabled(selectedSkillDetail.name)}
                          title={isSkillEnabled(selectedSkillDetail.name) ? "Desactivar" : "Activar"}
                        >
                          <span className="w-4 h-4 bg-white rounded-full transition-transform" />
                        </button>
                        <button
                          className="p-1 text-white/40 hover:text-white rounded-lg hover:bg-white/10 transition-colors"
                          onClick={() => setShowSkillMenu(!showSkillMenu)}
                        >
                          <span className="material-symbols-outlined text-[20px]">more_vert</span>
                        </button>

                        {showSkillMenu && (
                          <div className="absolute right-0 top-full mt-1 w-48 bg-[#262626] border border-white/10 rounded-xl shadow-2xl py-1.5 z-50 text-[13px] text-white">
                            <button
                              className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                              onClick={() => {
                                setShowSkillMenu(false);
                                setSettingsOpen(false);
                                useChatStore.getState().sendMessage(
                                  useChatStore.getState().activeChatId || "new",
                                  `/${selectedSkillDetail.name}`
                                );
                              }}
                            >
                              <span className="material-symbols-outlined text-[18px] text-white/60">chat_bubble_outline</span>
                              <span>Probar en chat</span>
                            </button>
                            {selectedSkillDetail.custom && (
                              <button
                                className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left text-red-400"
                                onClick={async () => {
                                  setShowSkillMenu(false);
                                  await removeSkill(selectedSkillDetail.id);
                                  toast(`Habilidad /${selectedSkillDetail.name} desinstalada`, "success");
                                  setSelectedSkillDetail(null);
                                }}
                              >
                                <span className="material-symbols-outlined text-[18px]">delete</span>
                                <span>Desinstalar</span>
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="text-[12px] text-white/50">
                      por <span className="text-white/80 font-medium">{selectedSkillDetail.author}</span>
                    </div>

                    <div className="text-[13px] text-white/70 leading-relaxed">
                      {selectedSkillDetail.description} <span className="text-[#3b82f6] cursor-pointer hover:underline font-medium">Ver más</span>
                    </div>

                    {/* Main Code / File Container */}
                    <div className="border border-white/10 rounded-2xl bg-[#1c1c1c] overflow-hidden flex flex-col">
                      {/* Container Header */}
                      <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-[#222222]">
                        <div className="flex items-center gap-3">
                          <div className="flex items-center gap-1.5 px-3 py-1 bg-white/10 rounded-lg text-[13px] font-medium text-white cursor-pointer hover:bg-white/15">
                            <span>SKILL.md</span>
                            <span className="material-symbols-outlined text-[16px] text-white/60">expand_more</span>
                          </div>
                          <span className="text-[12px] text-white/40">1 archivo</span>
                        </div>

                        <div className="flex items-center gap-1 bg-[#1a1a1a] p-1 rounded-xl border border-white/10 text-[12px]">
                          <button
                            className={`p-1.5 rounded-lg transition-all ${
                              skillViewMode === "preview" ? "bg-white/20 text-white" : "text-white/40 hover:text-white"
                            }`}
                            onClick={() => setSkillViewMode("preview")}
                            title="Vista Previa"
                          >
                            <span className="material-symbols-outlined text-[18px]">visibility</span>
                          </button>
                          <button
                            className={`p-1.5 rounded-lg transition-all ${
                              skillViewMode === "code" ? "bg-white/20 text-white" : "text-white/40 hover:text-white"
                            }`}
                            onClick={() => setSkillViewMode("code")}
                            title="Ver Código Raw"
                          >
                            <span className="material-symbols-outlined text-[18px]">code</span>
                          </button>
                        </div>
                      </div>

                      {/* Container Content */}
                      <div className="p-5 max-h-96 overflow-y-auto">
                        {skillViewMode === "preview" ? (
                          <div className="prose prose-invert max-w-none text-[13px] text-white/80 leading-relaxed font-sans whitespace-pre-wrap">
                            {selectedSkillDetail.template || `# ${selectedSkillDetail.name}\n${selectedSkillDetail.description}`}
                          </div>
                        ) : (
                          <pre className="font-mono text-[12px] text-lime-400 bg-[#121212] p-4 rounded-xl overflow-x-auto whitespace-pre-wrap">
                            {selectedSkillDetail.template || `name: ${selectedSkillDetail.name}\ndescription: ${selectedSkillDetail.description}`}
                          </pre>
                        )}
                      </div>
                    </div>
                  </div>
                ) : (
                  <>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-1 bg-[#242424] p-1 rounded-xl text-[12px] text-white/50 border border-white/5">
                        {["Todo", "Personalizadas", "De Ozy"].map((tab) => (
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
                        ...skills.map((s) => ({
                          id: s.id,
                          name: s.name,
                          description: s.description || "Habilidad personalizada cargada por el usuario.",
                          date: "Hoy",
                          author: "Usuario",
                          custom: true,
                          template: s.config?.template || s.description,
                        })),
                        {
                          id: "b1",
                          name: "mcp-builder",
                          description: "Guide for creating high-quality MCP (Model Context Protocol) servers that enable LLMs to interact with external services through well-designed tools.",
                          date: "22/7/26",
                          author: "Ozy Inc",
                          custom: false,
                          template: `# mcp-builder\n\nGuide for creating high-quality MCP (Model Context Protocol) servers that enable LLMs to interact with external services through well-designed tools. Use when building MCP servers to integrate external APIs or services.\n\n## Core Documentation\n- MCP Protocol: Start with sitemap\n- Best practices: Server and tool naming conventions`,
                        },
                        {
                          id: "b2",
                          name: "morning",
                          description: "Automated morning briefing skill that compiles key updates, calendar events, and pending tasks.",
                          date: "22/7/26",
                          author: "Ozy Inc",
                          custom: false,
                          template: `# morning\n\nAutomated morning briefing skill that compiles key updates, calendar events, and pending tasks.`,
                        },
                        {
                          id: "b3",
                          name: "skill-creator",
                          description: "Distills completed user workflows into reusable agent skills.",
                          date: "22/7/26",
                          author: "Ozy Inc",
                          custom: false,
                          template: `# skill-creator\n\nDistills completed user workflows into reusable agent skills.`,
                        },
                        {
                          id: "b4",
                          name: "web-artifacts-builder",
                          description: "Builds responsive web application components and preview artifacts.",
                          date: "22/7/26",
                          author: "Ozy Inc",
                          custom: false,
                          template: `# web-artifacts-builder\n\nBuilds responsive web application components and preview artifacts.`,
                        },
                      ].map((sk) => (
                        <div
                          key={sk.name}
                          className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px] cursor-pointer"
                          onClick={() => setSelectedSkillDetail(sk)}
                        >
                          <div className="col-span-5 font-mono font-medium text-white flex items-center gap-2">
                            <span className="material-symbols-outlined text-[16px] text-white/40">settings_suggest</span>
                            <span>{sk.name}</span>
                          </div>
                          <div className="col-span-3 text-white/40">{sk.date}</div>
                          <div className="col-span-3 text-white/70">{sk.author}</div>
                          <div className="col-span-1 flex justify-end">
                            {sk.custom && (
                              <button
                                className="p-1 text-white/40 hover:text-red-400 transition-colors"
                                onClick={async (e) => {
                                  e.stopPropagation();
                                  await removeSkill(sk.id);
                                  toast(`Habilidad /${sk.name} eliminada`, "success");
                                }}
                              >
                                <span className="material-symbols-outlined text-[16px]">delete</span>
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </div>
            )}

            {/* 9. CONECTORES (MCP) */}
            {normalizedCategory === "conectores" && (
              <div className="flex flex-col gap-5 w-full">
                <MCPConnectorsList />
              </div>
            )}

            {/* 10. PLUGINS */}
            {normalizedCategory === "plugins" && (
              <div className="flex flex-col gap-5 w-full">
                {selectedPluginDetail ? (() => {
                  const customPluginObj = skills.find((s) => s.name === selectedPluginDetail);
                  const isCustomPlugin = !!customPluginObj;
                  const pluginAuthor = isCustomPlugin ? "Usuario" : "Ozy Inc";
                  const pluginSource = isCustomPlugin ? "Local / Personalizado" : "Marketplace (Ozy Inc y socios)";
                  const pluginVersion = isCustomPlugin ? (customPluginObj.config?.version || "1.0.0") : "1.3.0";
                  const pluginDesc = isCustomPlugin
                    ? (customPluginObj.description || "Skill básico para validar sintaxis y estilo de código.")
                    : "Manage tasks, plan your day, and build up memory of important context about your work. Syncs with your calendar, email, and chat to keep everything organized and on track.";

                  return (
                    <div className="flex flex-col gap-5">
                      <div className="flex items-center justify-between border-b border-white/10 pb-4">
                        <button
                          className="flex items-center gap-1.5 text-[13px] text-white/60 hover:text-white transition-colors"
                          onClick={() => setSelectedPluginDetail(null)}
                        >
                          <span className="material-symbols-outlined text-[18px]">arrow_back</span>
                          <span>Plugins</span>
                        </button>
                      </div>

                      <div className="flex items-center justify-between">
                        <h2 className="text-[24px] font-semibold text-white tracking-tight">{selectedPluginDetail}</h2>

                        <div className="flex items-center gap-3">
                          <button
                            className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors"
                            onClick={() => {
                              toast(`Buscando actualizaciones para el plugin ${selectedPluginDetail}...`, "info");
                              setTimeout(() => {
                                toast(`Plugin ${selectedPluginDetail} está actualizado.`, "success");
                              }, 800);
                            }}
                          >
                            Actualizar
                          </button>
                          <button
                            className="px-4 py-1.5 bg-white/10 hover:bg-white/20 text-white text-[13px] font-medium rounded-lg transition-colors"
                            onClick={() => {
                              if (customPluginObj) {
                                setEditingSkillCode(
                                  customPluginObj.config?.template ||
                                  `name: ${customPluginObj.name}\ndescription: ${customPluginObj.description}\n---\n# Instrucciones`
                                );
                              } else {
                                setEditingSkillCode(`name: ${selectedPluginDetail}\ndescription: Plugin personalizado\n---\n# Instrucciones`);
                              }
                              setShowWriteSkillModal(true);
                            }}
                          >
                            Personalizar
                          </button>
                          <button
                            className={`w-10 h-5 rounded-full flex items-center px-0.5 shadow-sm transition-all cursor-pointer ${
                              isSkillEnabled(selectedPluginDetail) ? "bg-[#3b82f6] justify-end" : "bg-white/20 justify-start"
                            }`}
                            onClick={() => toggleSkillEnabled(selectedPluginDetail)}
                            title={isSkillEnabled(selectedPluginDetail) ? "Desactivar" : "Activar"}
                          >
                            <span className="w-4 h-4 bg-white rounded-full transition-transform" />
                          </button>
                          <div className="relative">
                            <button
                              className="p-1 text-white/40 hover:text-white rounded-lg hover:bg-white/10 transition-colors"
                              onClick={() => setShowPluginMenu(!showPluginMenu)}
                            >
                              <span className="material-symbols-outlined text-[20px]">more_vert</span>
                            </button>
                            {showPluginMenu && (
                              <div className="absolute right-0 top-full mt-1 w-48 bg-[#262626] border border-white/10 rounded-xl shadow-2xl py-1.5 z-50 text-[13px] text-white font-normal">
                                <button
                                  className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setShowPluginMenu(false);
                                    setSettingsOpen(false);
                                    useChatStore.getState().sendMessage(
                                      useChatStore.getState().activeChatId || "new",
                                      `Let's test the skills in plugin ${selectedPluginDetail}`
                                    );
                                  }}
                                >
                                  <span className="material-symbols-outlined text-[18px] text-white/60">chat_bubble_outline</span>
                                  <span>Probar en chat</span>
                                </button>
                                {isCustomPlugin && (
                                  <button
                                    className="w-full flex items-center gap-2.5 px-3.5 py-2 hover:bg-white/10 transition-colors text-left text-red-400"
                                    onClick={async (e) => {
                                      e.stopPropagation();
                                      setShowPluginMenu(false);
                                      await removeSkill(customPluginObj.id);
                                      toast(`Plugin ${customPluginObj.name} desinstalado`, "success");
                                      setSelectedPluginDetail(null);
                                    }}
                                  >
                                    <span className="material-symbols-outlined text-[18px]">delete</span>
                                    <span>Desinstalar</span>
                                  </button>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>

                      <div className="grid grid-cols-4 gap-4 py-2 border-y border-white/10 text-[13px]">
                        <div>
                          <div className="text-white/40 text-[12px]">Fuente</div>
                          <div className="text-white font-medium">{pluginSource}</div>
                        </div>
                        <div>
                          <div className="text-white/40 text-[12px]">Versión</div>
                          <div className="text-white font-medium">{pluginVersion}</div>
                        </div>
                        <div>
                          <div className="text-white/40 text-[12px]">Autor</div>
                          <div className="text-white font-medium">{pluginAuthor}</div>
                        </div>
                        <div>
                          <div className="text-white/40 text-[12px]">Última actualización</div>
                          <div className="text-white font-medium">Hoy</div>
                        </div>
                      </div>

                      <div className="text-[13px] text-white/70 leading-relaxed">
                        {pluginDesc}
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
                          {(isCustomPlugin
                            ? [{ name: "/" + customPluginObj.name, desc: customPluginObj.description || "Habilidad personalizada instalada." }]
                            : [
                                { name: "/memory-management", desc: "Two-tier memory system that makes Ozy a true workplace collaborator." },
                                { name: "/start", desc: "Initialize the productivity system and open the dashboard." },
                                { name: "/task-management", desc: "Simple task management using a shared TASKS.md file." },
                                { name: "/update", desc: "Sync tasks and refresh memory from your current activity." },
                              ]
                          ).map((item) => (
                            <div key={item.name} className="flex flex-col gap-0.5 py-1.5 border-b border-white/5">
                              <div className="font-mono text-[13px] font-semibold text-white">{item.name}</div>
                              <div className="text-[12px] text-white/50">{item.desc}</div>
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
                                Instalado
                              </button>
                            </div>
                          ))}
                        </div>
                      )}

                      <div className="flex flex-col gap-2.5 pt-3">
                        <div className="text-[13px] font-semibold text-white">Intenta preguntar...</div>
                        {[
                          `Ejecuta la habilidad /${selectedPluginDetail} para procesar código`,
                          "Set up my task and memory system",
                          "Catch me up and triage stale tasks",
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
                  );
                })() : (
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
                        <div className="col-span-2">Habilidades</div>
                        <div className="col-span-1 text-right">Acción</div>
                      </div>
                      {[
                        ...skills.map((s) => ({ id: s.id, name: s.name, author: "Usuario", count: 1, custom: true })),
                        { id: "p1", name: "Productivity", author: "Ozy Inc", count: 12, custom: false },
                        { id: "p2", name: "Engineering", author: "Ozy Inc", count: 10, custom: false },
                        { id: "p3", name: "Sales", author: "Ozy Inc", count: 9, custom: false },
                        { id: "p4", name: "Design", author: "Ozy Inc", count: 7, custom: false },
                      ].map((pl) => (
                        <div
                          key={pl.name}
                          className="grid grid-cols-12 px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors items-center text-[13px] cursor-pointer"
                          onClick={() => setSelectedPluginDetail(pl.name)}
                        >
                          <div className="col-span-5 font-medium text-white flex items-center gap-2">
                            <span className="material-symbols-outlined text-[16px] text-white/40">extension</span>
                            <span>{pl.name}</span>
                          </div>
                          <div className="col-span-4 text-white/60">{pl.author}</div>
                          <div className="col-span-2 text-white/40">{pl.count} habilidades</div>
                          <div className="col-span-1 flex justify-end">
                            {pl.custom && (
                              <button
                                className="p-1 text-white/40 hover:text-red-400 transition-colors"
                                onClick={async (e) => {
                                  e.stopPropagation();
                                  await removeSkill(pl.id);
                                  toast(`Plugin ${pl.name} desinstalado`, "success");
                                }}
                              >
                                <span className="material-symbols-outlined text-[16px]">delete</span>
                              </button>
                            )}
                          </div>
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

      <AddConnectorModal
        isOpen={showCustomConnectorModal}
        onClose={() => setShowCustomConnectorModal(false)}
      />
      <UploadSkillModal
        isOpen={showUploadSkillModal}
        onClose={() => setShowUploadSkillModal(false)}
      />
      <WriteSkillModal
        isOpen={showWriteSkillModal}
        onClose={() => setShowWriteSkillModal(false)}
        initialCode={editingSkillCode}
      />
      <MarketplaceModal
        isOpen={showMarketplaceModal}
        onClose={() => setShowMarketplaceModal(false)}
      />
    </div>
  );
}
