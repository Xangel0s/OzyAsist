import React, { useState, useRef, useEffect } from 'react';
import {
  Plus,
  Mic,
  ArrowUp,
  Square,
  ChevronDown,
  ChevronRight,
  Cpu,
  Cloud,
  Settings,
  Sparkles,
  Paperclip,
  Folder,
  Scroll,
  Blocks,
  Plug,
  Globe,
  Search,
  Briefcase,
  Check,
} from 'lucide-react';
import { useAvailableModels, type ModelOption } from '../../hooks/useAvailableModels';
import { useUIStore } from '../../store/uiStore';
import { useProjectsStore } from '../../store/projectsStore';
import { useSkillsStore } from '../../store/skillsStore';
import { useConnectorsStore } from '../../store/connectorsStore';
import { useOnClickOutside } from '../../hooks/useOnClickOutside';

interface OzyChatConsoleProps {
  onSendMessage: (text: string, mode: 'chat' | 'cowork', model: string) => void;
  onStopStreaming?: () => void;
  isStreaming?: boolean;
}

export const OzyChatConsole: React.FC<OzyChatConsoleProps> = ({
  onSendMessage,
  onStopStreaming,
  isStreaming = false,
}) => {
  const [text, setText] = useState('');
  const [mode, setMode] = useState<'chat' | 'cowork'>('chat');
  const [selectedModel, setSelectedModel] = useState<ModelOption | null>(null);
  const [showModelDropdown, setShowModelDropdown] = useState(false);
  const [showPlusMenu, setShowPlusMenu] = useState(false);
  const [activeSubmenu, setActiveSubmenu] = useState<'project' | 'skills' | 'connectors' | null>(null);
  const [webSearchEnabled, setWebSearchEnabled] = useState(false);
  const [projectSearch, setProjectSearch] = useState('');
  const [modelSearch, setModelSearch] = useState('');
  const [isVoiceConnecting, setIsVoiceConnecting] = useState(false);
  const [voiceTranscript, setVoiceTranscript] = useState('');
  const voiceRecRef = useRef<any>(null);
  const [connectorToggles, setConnectorToggles] = useState<Record<string, boolean>>({
    opencode: true,
  });

  // Handle active speech recognition when voice mode is opened
  useEffect(() => {
    if (!isVoiceConnecting) {
      if (voiceRecRef.current) {
        try {
          voiceRecRef.current.stop();
        } catch {}
        voiceRecRef.current = null;
      }
      return;
    }

    const SpeechRec = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRec) {
      setIsVoiceConnecting(false);
      alert("El reconocimiento de voz no está soportado en este navegador. Usa Chrome o Edge.");
      return;
    }

    setVoiceTranscript('');
    try {
      const rec = new SpeechRec();
      rec.lang = 'es-ES';
      rec.continuous = true;
      rec.interimResults = true;

      rec.onresult = (event: any) => {
        let currentText = '';
        for (let i = 0; i < event.results.length; ++i) {
          currentText += event.results[i][0].transcript;
        }
        setVoiceTranscript(currentText);
      };

      rec.onerror = (err: any) => {
        if (err.error !== 'no-speech') {
          console.debug('Voice input error:', err.error);
        }
      };

      rec.onend = () => {
        // keep listening if still in voice mode
        if (voiceRecRef.current) {
          try {
            rec.start();
          } catch {}
        }
      };

      voiceRecRef.current = rec;
      rec.start();
    } catch (e) {
      console.debug('Failed to start speech recognition:', e);
    }

    return () => {
      if (voiceRecRef.current) {
        try {
          voiceRecRef.current.stop();
        } catch {}
        voiceRecRef.current = null;
      }
    };
  }, [isVoiceConnecting]);

  const handleSendVoiceMessage = () => {
    const textToSend = voiceTranscript.trim();
    if (!textToSend) return;
    setIsVoiceConnecting(false);
    if (!selectedModel && models.length === 0) {
      useUIStore.getState().openSettings('proveedores');
      return;
    }
    const modelToUse = selectedModel ? selectedModel.id : 'ozy-hybrid';
    onSendMessage(textToSend, mode, modelToUse);
    setVoiceTranscript('');
  };

  const { models, loading, refetch } = useAvailableModels();
  const filteredModels = models.filter((m) =>
    m.name.toLowerCase().includes(modelSearch.toLowerCase()) ||
    m.id.toLowerCase().includes(modelSearch.toLowerCase()) ||
    m.group.toLowerCase().includes(modelSearch.toLowerCase())
  );
  const projects = useProjectsStore((s) => s.projects);
  const activeProjectId = useProjectsStore((s) => s.activeProjectId);
  const setActiveProject = useProjectsStore((s) => s.setActiveProject);
  const skills = useSkillsStore((s) => s.skills);
  const connectors = useConnectorsStore((s) => s.connectors);

  const dropdownRef = useRef<HTMLDivElement>(null);
  const plusMenuRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useOnClickOutside(dropdownRef, () => setShowModelDropdown(false));
  useOnClickOutside(plusMenuRef, () => {
    setShowPlusMenu(false);
    setActiveSubmenu(null);
  });

  // Ctrl+U / Cmd+U shortcut for file attachment
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'u') {
        e.preventDefault();
        fileInputRef.current?.click();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Default models selection
  useEffect(() => {
    if (models.length > 0) {
      if (!selectedModel || !models.some((m) => m.id === selectedModel.id)) {
        const defaultModel = models.find((m) => m.is_default) || models[0];
        setSelectedModel(defaultModel);
      }
    } else {
      setSelectedModel(null);
    }
  }, [models, selectedModel]);

  // Auto-ajuste de altura al escribir
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
    }
  }, [text]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleSubmit = () => {
    if (!text.trim() || isStreaming) return;
    if (!selectedModel && models.length === 0) {
      import('../../store/toastStore').then(({ useToastStore }) => {
        useToastStore.getState().show('Por favor configura un proveedor LLM o inicia Ollama', 'warning');
      });
      useUIStore.getState().openSettings('proveedores');
      return;
    }
    const modelToUse = selectedModel ? selectedModel.id : 'ozy-hybrid';
    let messageToSend = text;
    if (webSearchEnabled && !messageToSend.startsWith('/deep-search')) {
      messageToSend = `/deep-search ${messageToSend}`;
    }
    onSendMessage(messageToSend, mode, modelToUse);
    setText('');
    if (textareaRef.current) textareaRef.current.style.height = 'auto';
  };

  const closePlusMenu = () => {
    setShowPlusMenu(false);
    setActiveSubmenu(null);
  };

  // Built-in / loaded skills
  const displaySkills = skills.length > 0 ? skills : [
    { id: '1', name: 'import-memory', description: 'Importar memoria al sistema' },
    { id: '2', name: 'morning', description: 'Resumen matutino de tareas' },
    { id: '3', name: 'skill-creator', description: 'Crear nueva habilidad' },
  ];

  // Filtered projects
  const filteredProjects = projects.filter((p) =>
    p.name.toLowerCase().includes(projectSearch.toLowerCase())
  );

  return (
    <div className="relative w-full max-w-3xl mx-auto">
      {/* Input Oculto de Archivos */}
      <input
        type="file"
        ref={fileInputRef}
        className="hidden"
        onChange={(e) => {
          if (e.target.files?.[0]) {
            setText((prev) => `${prev} [Archivo adjunto: ${e.target.files![0].name}] `);
          }
        }}
      />

      {/* Contenedor Principal Estilo Claude */}
      <div className="relative rounded-[24px] border border-white/10 bg-[#1e1e1e]/90 p-3.5 shadow-2xl backdrop-blur-xl transition-all duration-200 focus-within:border-white/20 focus-within:bg-[#202020]">
        {!isVoiceConnecting ? (
          <>
            {/* Textarea Autoexpandible */}
            <textarea
              ref={textareaRef}
              rows={1}
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="¿Cómo puedo ayudarte hoy?"
              className="w-full resize-none bg-transparent px-2 font-sans text-[15px] leading-relaxed text-neutral-100 placeholder-neutral-500 focus:outline-none"
            />

            {/* Barra Inferior */}
            <div className="mt-2 flex items-center justify-between pt-1">
              <div className="flex items-center gap-1.5">
                {/* Botón Acciones Contextuales (+) */}
                <div className="relative" ref={plusMenuRef}>
                  <button
                    type="button"
                    onClick={() => {
                      setShowPlusMenu(!showPlusMenu);
                      setActiveSubmenu(null);
                    }}
                    title="Acciones, habilidades y conectores"
                    className={`flex h-8 w-8 items-center justify-center rounded-full transition-all ${
                      showPlusMenu
                        ? 'bg-white/20 text-white'
                        : 'text-neutral-400 hover:bg-white/5 hover:text-neutral-200'
                    }`}
                  >
                    <Plus className="h-4 w-4" />
                  </button>

                  {/* Popover Menú + */}
                  {showPlusMenu && (
                    <div className="absolute left-0 bottom-full mb-2 w-64 rounded-2xl border border-white/10 bg-[#1a1a1a]/95 p-1.5 shadow-2xl backdrop-blur-2xl z-40 animate-fadeIn select-none">
                      {/* 1. Agregar archivos o fotos */}
                      <button
                        type="button"
                        onClick={() => {
                          fileInputRef.current?.click();
                          closePlusMenu();
                        }}
                        className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left group"
                      >
                        <div className="flex items-center gap-2.5">
                          <Paperclip className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                          <span>Agregar archivos o fotos</span>
                        </div>
                        <span className="flex items-center gap-0.5 text-[10px] text-neutral-400 bg-white/5 border border-white/10 px-1.5 py-0.5 rounded font-mono">
                          <span>Ctrl</span>
                          <span>U</span>
                        </span>
                      </button>

                      {/* 2. Agregar al proyecto > */}
                      <div className="relative">
                        <button
                          type="button"
                          onMouseEnter={() => setActiveSubmenu('project')}
                          onClick={() =>
                            setActiveSubmenu(activeSubmenu === 'project' ? null : 'project')
                          }
                          className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs transition-all text-left group ${
                            activeSubmenu === 'project'
                              ? 'bg-white/10 text-white font-medium'
                              : 'text-neutral-200 hover:bg-white/5'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <Folder className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                            <span>Agregar al proyecto</span>
                          </div>
                          <ChevronRight className="h-4 w-4 text-neutral-500 group-hover:text-neutral-300" />
                        </button>

                        {/* Submenú Proyectos */}
                        {activeSubmenu === 'project' && (
                          <div className="absolute left-full top-0 ml-1.5 w-64 rounded-2xl border border-white/10 bg-[#1a1a1a]/95 p-1.5 shadow-2xl backdrop-blur-2xl z-50 animate-fadeIn">
                            <div className="p-1.5 border-b border-white/10 flex items-center gap-2 mb-1">
                              <Search className="h-3.5 w-3.5 text-neutral-400 shrink-0" />
                              <input
                                type="text"
                                placeholder="Buscar proyectos"
                                value={projectSearch}
                                onChange={(e) => setProjectSearch(e.target.value)}
                                className="w-full bg-transparent text-xs text-white placeholder-neutral-500 focus:outline-none"
                              />
                            </div>

                            <div className="max-h-48 overflow-y-auto scrollbar-thin flex flex-col gap-0.5">
                              {filteredProjects.length === 0 ? (
                                <div className="px-3 py-2.5 text-xs text-neutral-500">
                                  Aún no hay proyectos
                                </div>
                              ) : (
                                filteredProjects.map((p) => (
                                  <button
                                    key={p.id}
                                    type="button"
                                    onClick={() => {
                                      setActiveProject(p.id);
                                      setText((prev) => `[Proyecto: ${p.name}] ${prev}`);
                                      closePlusMenu();
                                    }}
                                    className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs transition-all text-left ${
                                      activeProjectId === p.id
                                        ? 'bg-white/10 text-white font-medium'
                                        : 'text-neutral-300 hover:bg-white/5 hover:text-white'
                                    }`}
                                  >
                                    <div className="flex items-center gap-2 truncate">
                                      <Folder className="h-3.5 w-3.5 text-neutral-400 shrink-0" />
                                      <span className="truncate">{p.name}</span>
                                    </div>
                                    {activeProjectId === p.id && (
                                      <Check className="h-3.5 w-3.5 text-[#d1f107] shrink-0" />
                                    )}
                                  </button>
                                ))
                              )}
                            </div>

                            <div className="my-1 border-t border-white/10" />

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().setActiveView('projects');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left font-medium"
                            >
                              <Plus className="h-3.5 w-3.5 text-neutral-400" />
                              <span>Iniciar un nuevo proyecto</span>
                            </button>
                          </div>
                        )}
                      </div>

                      {/* 3. Habilidades > */}
                      <div className="relative">
                        <button
                          type="button"
                          onMouseEnter={() => setActiveSubmenu('skills')}
                          onClick={() =>
                            setActiveSubmenu(activeSubmenu === 'skills' ? null : 'skills')
                          }
                          className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs transition-all text-left group ${
                            activeSubmenu === 'skills'
                              ? 'bg-white/10 text-white font-medium'
                              : 'text-neutral-200 hover:bg-white/5'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <Scroll className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                            <span>Habilidades</span>
                          </div>
                          <ChevronRight className="h-4 w-4 text-neutral-500 group-hover:text-neutral-300" />
                        </button>

                        {/* Submenú Habilidades */}
                        {activeSubmenu === 'skills' && (
                          <div className="absolute left-full top-0 ml-1.5 w-64 rounded-2xl border border-white/10 bg-[#1a1a1a]/95 p-1.5 shadow-2xl backdrop-blur-2xl z-50 animate-fadeIn">
                            <div className="max-h-48 overflow-y-auto scrollbar-thin flex flex-col gap-0.5">
                              {displaySkills.map((s) => (
                                <button
                                  key={s.id}
                                  type="button"
                                  onClick={() => {
                                    setText((prev) => `/${s.name} ${prev}`);
                                    textareaRef.current?.focus();
                                    closePlusMenu();
                                  }}
                                  className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-xs text-neutral-300 hover:bg-white/5 hover:text-white transition-all text-left font-mono"
                                >
                                  <Scroll className="h-3.5 w-3.5 text-neutral-400 shrink-0" />
                                  <span className="truncate">{s.name}</span>
                                </button>
                              ))}
                            </div>

                            <div className="my-1 border-t border-white/10" />

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('habilidades');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left"
                            >
                              <Briefcase className="h-3.5 w-3.5 text-neutral-400" />
                              <span>Administrar habilidades</span>
                            </button>

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('habilidades');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left"
                            >
                              <Plus className="h-3.5 w-3.5 text-neutral-400" />
                              <span>Explorar habilidades</span>
                            </button>
                          </div>
                        )}
                      </div>

                      {/* 4. Conectores > */}
                      <div className="relative">
                        <button
                          type="button"
                          onMouseEnter={() => setActiveSubmenu('connectors')}
                          onClick={() =>
                            setActiveSubmenu(activeSubmenu === 'connectors' ? null : 'connectors')
                          }
                          className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs transition-all text-left group ${
                            activeSubmenu === 'connectors'
                              ? 'bg-white/10 text-white font-medium'
                              : 'text-neutral-200 hover:bg-white/5'
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <Blocks className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                            <span>Conectores</span>
                          </div>
                          <ChevronRight className="h-4 w-4 text-neutral-500 group-hover:text-neutral-300" />
                        </button>

                        {/* Submenú Conectores */}
                        {activeSubmenu === 'connectors' && (
                          <div className="absolute left-full top-0 ml-1.5 w-64 rounded-2xl border border-white/10 bg-[#1a1a1a]/95 p-1.5 shadow-2xl backdrop-blur-2xl z-50 animate-fadeIn">
                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('conectores');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left"
                            >
                              <div className="flex items-center gap-2">
                                <Plus className="h-3.5 w-3.5 text-neutral-400" />
                                <span>Agregar conector</span>
                              </div>
                              <ChevronRight className="h-3.5 w-3.5 text-neutral-500" />
                            </button>

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('conectores');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left"
                            >
                              <Briefcase className="h-3.5 w-3.5 text-neutral-400" />
                              <span>Administrar conectores</span>
                            </button>

                            <div className="my-1 border-t border-white/10" />

                            {/* Lista de Conectores Activos con Switch */}
                            <div className="flex flex-col gap-0.5">
                              <div className="flex items-center justify-between px-3 py-1.5 text-xs text-neutral-200 rounded-xl hover:bg-white/5 transition-all">
                                <div className="flex items-center gap-2">
                                  <span className="h-2 w-2 rounded-full border border-neutral-400" />
                                  <span className="font-mono text-xs">opencode</span>
                                </div>
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setConnectorToggles((prev) => ({
                                      ...prev,
                                      opencode: !prev.opencode,
                                    }));
                                  }}
                                  className={`relative inline-flex h-4 w-7 shrink-0 cursor-pointer rounded-full transition-colors duration-200 ease-in-out ${
                                    connectorToggles.opencode ? 'bg-blue-500' : 'bg-neutral-700'
                                  }`}
                                >
                                  <span
                                    className={`inline-block h-3 w-3 transform rounded-full bg-white transition duration-200 ease-in-out mt-0.5 ml-0.5 ${
                                      connectorToggles.opencode ? 'translate-x-3' : 'translate-x-0'
                                    }`}
                                  />
                                </button>
                              </div>

                              {connectors.map((c) => (
                                <div
                                  key={c.id}
                                  className="flex items-center justify-between px-3 py-1.5 text-xs text-neutral-200 rounded-xl hover:bg-white/5 transition-all"
                                >
                                  <div className="flex items-center gap-2 truncate">
                                    <span className="h-2 w-2 rounded-full border border-neutral-400 shrink-0" />
                                    <span className="truncate font-mono text-xs">{c.name}</span>
                                  </div>
                                  <button
                                    type="button"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      setConnectorToggles((prev) => ({
                                        ...prev,
                                        [c.name]: !prev[c.name],
                                      }));
                                    }}
                                    className={`relative inline-flex h-4 w-7 shrink-0 cursor-pointer rounded-full transition-colors duration-200 ease-in-out ${
                                      connectorToggles[c.name] !== false
                                        ? 'bg-blue-500'
                                        : 'bg-neutral-700'
                                    }`}
                                  >
                                    <span
                                      className={`inline-block h-3 w-3 transform rounded-full bg-white transition duration-200 ease-in-out mt-0.5 ml-0.5 ${
                                        connectorToggles[c.name] !== false
                                          ? 'translate-x-3'
                                          : 'translate-x-0'
                                      }`}
                                    />
                                  </button>
                                </div>
                              ))}
                            </div>

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('conectores');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs text-neutral-300 hover:bg-white/5 transition-all text-left"
                            >
                              <div className="flex items-center gap-2">
                                <span className="h-2 w-2 rounded-full border border-neutral-400" />
                                <span>Agregar desde opencode</span>
                              </div>
                              <ChevronRight className="h-3.5 w-3.5 text-neutral-500" />
                            </button>

                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('conectores');
                                closePlusMenu();
                              }}
                              className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs text-neutral-300 hover:bg-white/5 transition-all text-left"
                            >
                              <div className="flex items-center gap-2">
                                <Search className="h-3.5 w-3.5 text-neutral-400" />
                                <span>Acceso a herramientas</span>
                              </div>
                              <ChevronRight className="h-3.5 w-3.5 text-neutral-500" />
                            </button>
                          </div>
                        )}
                      </div>

                      {/* 5. Agregar plugins... */}
                      <button
                        type="button"
                        onClick={() => {
                          useUIStore.getState().openSettings('plugins');
                          closePlusMenu();
                        }}
                        className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left group"
                      >
                        <Plug className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                        <span>Agregar plugins...</span>
                      </button>

                      <div className="my-1 border-t border-white/10" />

                      {/* 6. Búsqueda web (con check) */}
                      <button
                        type="button"
                        onClick={() => setWebSearchEnabled(!webSearchEnabled)}
                        className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs text-neutral-200 hover:bg-white/5 transition-all text-left group"
                      >
                        <div className="flex items-center gap-2.5">
                          <Globe className="h-4 w-4 text-neutral-400 group-hover:text-white" />
                          <span>Búsqueda web</span>
                        </div>
                        {webSearchEnabled && (
                          <Check className="h-4 w-4 text-blue-400 stroke-[2.5]" />
                        )}
                      </button>
                    </div>
                  )}
                </div>

                {/* Interruptor Modo Chat / Cowork */}
                <div className="flex items-center rounded-xl bg-black/40 p-0.5 border border-white/5 text-xs">
                  <button
                    type="button"
                    onClick={() => setMode('chat')}
                    className={`rounded-lg px-2.5 py-1 font-medium transition-all ${
                      mode === 'chat'
                        ? 'bg-white/10 text-white shadow-sm'
                        : 'text-neutral-400 hover:text-neutral-200'
                    }`}
                  >
                    Chat
                  </button>
                  <button
                    type="button"
                    onClick={() => setMode('cowork')}
                    className={`rounded-lg px-2.5 py-1 font-medium transition-all ${
                      mode === 'cowork'
                        ? 'bg-[#d1f107]/10 text-[#d1f107]'
                        : 'text-neutral-400 hover:text-neutral-200'
                    }`}
                  >
                    Cowork
                  </button>
                </div>
              </div>

              {/* Selector de Modelo + Micrófono + Envío */}
              <div className="flex items-center gap-2">
                {/* Selector de Modelo Dinámico */}
                <div className="relative" ref={dropdownRef}>
                  <button
                    type="button"
                    onClick={() => {
                      setShowModelDropdown(!showModelDropdown);
                      if (!showModelDropdown) refetch();
                    }}
                    className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium text-neutral-400 hover:bg-white/5 hover:text-neutral-200 transition-all select-none"
                  >
                    <span className="max-w-[150px] truncate">
                      {selectedModel
                        ? selectedModel.name
                        : loading
                        ? 'Detectando...'
                        : 'Por configurar'}
                    </span>
                    {selectedModel && (
                      <span className="h-1.5 w-1.5 rounded-full bg-[#d1f107] shrink-0" />
                    )}
                    <ChevronDown className="h-3.5 w-3.5 opacity-60 shrink-0" />
                  </button>

                  {showModelDropdown && (
                    <div className="absolute right-0 bottom-full mb-2 w-80 rounded-2xl border border-white/10 bg-[#1a1a1a]/95 p-2 shadow-2xl backdrop-blur-xl z-30 max-h-80 overflow-y-auto scrollbar-thin animate-fadeIn select-none">
                      {loading && models.length === 0 ? (
                        <div className="p-4 text-center text-xs text-neutral-400 flex items-center justify-center gap-2">
                          <span className="h-3.5 w-3.5 border-2 border-[#d1f107] border-t-transparent rounded-full animate-spin" />
                          <span>Detectando modelos disponibles...</span>
                        </div>
                      ) : models.length === 0 ? (
                        <div className="p-4 text-center text-xs text-neutral-400 flex flex-col items-center">
                          <div className="w-9 h-9 rounded-xl bg-white/5 border border-white/10 flex items-center justify-center text-neutral-400 mb-2">
                            <Settings className="h-4 w-4" />
                          </div>
                          <span className="font-semibold text-white text-xs mb-1">
                            Sin modelos configurados
                          </span>
                          <p className="text-[11px] text-neutral-400 leading-relaxed mb-3">
                            No se detectó ninguna API Key funcional ni Ollama local con modelos descargados.
                          </p>
                          <button
                            type="button"
                            onClick={() => {
                              useUIStore.getState().openSettings('proveedores');
                              setShowModelDropdown(false);
                            }}
                            className="w-full py-2 px-3 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[11px] transition-all flex items-center justify-center gap-1.5 shadow-lg shadow-[#d1f107]/10"
                          >
                            <Settings className="h-3.5 w-3.5" />
                            <span>Configurar Proveedores</span>
                          </button>
                        </div>
                      ) : (
                        <div className="flex flex-col gap-1">
                          {models.length > 5 && (
                            <div className="p-1.5 border-b border-white/10 flex items-center gap-2 mb-1">
                              <Search className="h-3.5 w-3.5 text-neutral-400 shrink-0" />
                              <input
                                type="text"
                                placeholder="Buscar modelo (ej. claude, deepseek)..."
                                value={modelSearch}
                                onChange={(e) => setModelSearch(e.target.value)}
                                className="w-full bg-transparent text-xs text-white placeholder-neutral-500 focus:outline-none"
                              />
                            </div>
                          )}

                          <div className="max-h-60 overflow-y-auto scrollbar-thin flex flex-col gap-0.5">
                            {filteredModels.map((m) => (
                              <button
                                key={m.id}
                                type="button"
                                onClick={() => {
                                  setSelectedModel(m);
                                  setShowModelDropdown(false);
                                }}
                                className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-xs transition-all text-left group ${
                                  selectedModel?.id === m.id
                                    ? 'bg-white/10 text-white'
                                    : 'text-neutral-300 hover:bg-white/5 hover:text-white'
                                }`}
                              >
                                <div className="flex items-center gap-2.5 truncate">
                                  {m.provider === 'hybrid' ? (
                                    <Sparkles className="h-3.5 w-3.5 text-[#d1f107] shrink-0" />
                                  ) : m.is_local ? (
                                    <Cpu className="h-3.5 w-3.5 text-[#d1f107] shrink-0" />
                                  ) : (
                                    <Cloud className="h-3.5 w-3.5 text-sky-400 shrink-0" />
                                  )}
                                  <div className="truncate">
                                    <span className="font-medium text-white truncate block">
                                      {m.name}
                                    </span>
                                    <p className="text-[10px] text-neutral-500 truncate">{m.group}</p>
                                  </div>
                                </div>
                                {selectedModel?.id === m.id && (
                                  <span className="h-1.5 w-1.5 rounded-full bg-[#d1f107] shrink-0 ml-2" />
                                )}
                              </button>
                            ))}
                          </div>

                          <div className="mt-1 pt-2 border-t border-white/5 flex items-center justify-between px-2 text-[11px]">
                            <span className="text-neutral-500">¿Añadir más claves?</span>
                            <button
                              type="button"
                              onClick={() => {
                                useUIStore.getState().openSettings('proveedores');
                                setShowModelDropdown(false);
                              }}
                              className="text-[#d1f107] hover:underline font-semibold"
                            >
                              Ajustes
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* Dictado por Voz */}
                <button
                  type="button"
                  onClick={() => useUIStore.getState().setVoiceLiveOpen(true)}
                  title="Ozy Live — Voz en Tiempo Real (Hey Ozy)"
                  className="flex h-8 w-8 items-center justify-center rounded-full text-neutral-400 hover:bg-white/5 hover:text-[#d1f107] transition-all"
                >
                  <Mic className="h-4 w-4" />
                </button>

                {/* Botón de Envío / Detención */}
                {isStreaming ? (
                  <button
                    type="button"
                    onClick={onStopStreaming}
                    title="Detener respuesta"
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 transition-all"
                  >
                    <Square className="h-3.5 w-3.5 fill-current" />
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={!text.trim()}
                    onClick={handleSubmit}
                    title="Enviar mensaje"
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-[#d1f107] text-black font-bold hover:opacity-90 disabled:opacity-20 disabled:hover:opacity-20 transition-all shadow-[0_0_12px_rgba(209,241,7,0.25)]"
                  >
                    <ArrowUp className="h-4 w-4 stroke-[2.5]" />
                  </button>
                )}
              </div>
            </div>
          </>
        ) : (
          /* Estado de Conexión y Dictado de Voz en Tiempo Real */
          <div className="flex flex-col gap-3 p-2 animate-fadeIn">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="relative flex h-8 w-8 items-center justify-center rounded-full bg-[#d1f107]/20 border border-[#d1f107]/40 text-[#d1f107]">
                  <Mic className="h-4 w-4 animate-bounce" />
                  <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-[#d1f107] animate-ping" />
                </div>
                <div className="flex flex-col">
                  <span className="text-xs font-bold text-[#d1f107]">Escuchando tu voz...</span>
                  <span className="text-[11px] text-neutral-400">Habla con claridad en español</span>
                </div>
              </div>

              {/* Audio Wave Spectrum Animation */}
              <div className="flex items-center gap-1 h-5 px-3">
                {[40, 80, 55, 95, 70, 30, 85, 60, 100, 45, 75].map((h, i) => (
                  <span
                    key={i}
                    style={{ height: `${h}%` }}
                    className="w-1 bg-[#d1f107] rounded-full animate-pulse transition-all duration-150"
                  />
                ))}
              </div>
            </div>

            {/* Live Transcript Display */}
            <div className="min-h-[48px] max-h-32 overflow-y-auto rounded-xl bg-black/40 border border-white/5 p-3 text-[14px] text-neutral-100 font-sans">
              {voiceTranscript ? (
                <span>{voiceTranscript}</span>
              ) : (
                <span className="text-neutral-500 italic">Di algo como: "Explica cómo funciona la concurrencia en Go..."</span>
              )}
            </div>

            {/* Actions Bar */}
            <div className="flex items-center justify-end gap-2 pt-1">
              <button
                type="button"
                onClick={() => {
                  setIsVoiceConnecting(false);
                  setVoiceTranscript('');
                }}
                className="rounded-full bg-white/10 px-4 py-1.5 text-xs font-medium text-neutral-300 hover:bg-white/20 transition-all"
              >
                Cancelar
              </button>
              <button
                type="button"
                disabled={!voiceTranscript.trim()}
                onClick={handleSendVoiceMessage}
                className="flex items-center gap-1.5 rounded-full bg-[#d1f107] px-4 py-1.5 text-xs font-bold text-black hover:opacity-90 disabled:opacity-30 transition-all shadow-[0_0_12px_rgba(209,241,7,0.25)]"
              >
                <span>Enviar audio</span>
                <ArrowUp className="h-3.5 w-3.5 stroke-[2.5]" />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Sugerencias Rápidas Funcionales */}
      <div className="mt-3 flex flex-wrap items-center justify-center gap-2">
        {[
          { label: 'Escribir', prompt: 'Ayúdame a redactar ' },
          { label: 'Aprender', prompt: 'Explícame el concepto de ' },
          { label: 'Código', prompt: 'Escribe una función en Go para ' },
          { label: 'DeepSearch', prompt: '/deep-search ' },
        ].map((item) => (
          <button
            key={item.label}
            type="button"
            onClick={() => {
              setText(item.prompt);
              textareaRef.current?.focus();
            }}
            className="rounded-full border border-white/5 bg-white/[0.02] px-3.5 py-1 text-xs text-neutral-400 hover:border-white/15 hover:bg-white/[0.05] hover:text-neutral-200 transition-all"
          >
            {item.label}
          </button>
        ))}
      </div>
    </div>
  );
};
export default OzyChatConsole;
