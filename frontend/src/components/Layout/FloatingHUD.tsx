import React from 'react';
import { ShieldAlert, CheckCircle, AlertTriangle, ChevronDown, ChevronUp, Cpu, HardDrive, Sparkles } from 'lucide-react';
import { useTaskStore } from '../../store/taskStore';
import { HUDPinModal } from './HUDPinModal';
import { api } from '../../services/api';

export const FloatingHUD: React.FC = () => {
  const { activeTask, pinRequest, isHUDVisible, isHUDExpanded, toggleHUDExpanded, setPINRequest } = useTaskStore();

  if (!isHUDVisible || !activeTask) return null;

  const handleAuthorizePIN = async (pin: string) => {
    if (!pinRequest) return;
    await api.post(`/api/v1/tasks/${pinRequest.task_id}/authorize`, {
      step_id: pinRequest.step_id,
      pin,
    });
    setPINRequest(null);
  };

  const handleRejectPIN = async () => {
    if (!pinRequest) return;
    await api.post(`/api/v1/tasks/${pinRequest.task_id}/cancel`, {});
    setPINRequest(null);
  };

  const progressPct = Math.round((activeTask.step_current / Math.max(activeTask.step_total, 1)) * 100);

  const getStatusBadge = () => {
    switch (activeTask.status) {
      case 'running':
        return (
          <span className="flex items-center gap-1.5 text-xs font-semibold text-sky-400">
            <span className="h-2 w-2 animate-ping rounded-full bg-sky-400" />
            Ejecutando
          </span>
        );
      case 'blocked_approval':
        return (
          <span className="flex items-center gap-1.5 text-xs font-bold text-red-400 animate-pulse">
            <ShieldAlert className="h-3.5 w-3.5 text-red-400" />
            Requiere PIN
          </span>
        );
      case 'completed':
        return (
          <span className="flex items-center gap-1.5 text-xs font-bold text-brand-lime">
            <CheckCircle className="h-3.5 w-3.5 text-brand-lime" />
            Completado
          </span>
        );
      case 'failed':
        return (
          <span className="flex items-center gap-1.5 text-xs font-bold text-rose-400">
            <AlertTriangle className="h-3.5 w-3.5" />
            Error
          </span>
        );
      default:
        return <span className="text-xs text-slate-400">{activeTask.status}</span>;
    }
  };

  return (
    <>
      <div className="fixed bottom-5 right-5 z-40 flex flex-col items-end animate-slideUp font-sans">
        <div className="flex items-center gap-3 rounded-full border border-white/10 bg-[#131313]/90 px-4 py-2 shadow-2xl backdrop-blur-xl transition-all hover:border-white/20">
          <div className="flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-[#d1f107]" />
            {getStatusBadge()}
          </div>

          <div className="h-4 w-px bg-slate-800" />

          <span className="max-w-[200px] truncate text-xs font-medium text-slate-200">
            {activeTask.title}
          </span>

          <span className="rounded-full bg-slate-800 px-2 py-0.5 font-mono text-[10px] text-slate-300 font-bold">
            {activeTask.step_current}/{activeTask.step_total} ({progressPct}%)
          </span>

          {activeTask.telemetry && (
            <div className="hidden sm:flex items-center gap-2 border-l border-slate-800 pl-2 text-[10px] text-slate-400 font-mono">
              <span className="flex items-center gap-0.5">
                <Cpu className="h-3 w-3 text-slate-500" />
                {activeTask.telemetry.cpu_pct.toFixed(0)}%
              </span>
              <span className="flex items-center gap-0.5">
                <HardDrive className="h-3 w-3 text-slate-500" />
                {activeTask.telemetry.ram_free_mb}MB
              </span>
            </div>
          )}

          {activeTask.status === 'blocked_approval' && (
            <button
              onClick={() => {}}
              className="rounded-full bg-red-500/20 px-2.5 py-0.5 text-xs font-bold text-red-400 hover:bg-red-500/30 transition-colors"
            >
              Ingresar PIN
            </button>
          )}

          <button onClick={toggleHUDExpanded} className="text-slate-400 hover:text-white transition-colors">
            {isHUDExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
          </button>
        </div>

        {/* Panel Expandido con logs y detalles */}
        {isHUDExpanded && (
          <div className="mt-2 w-80 rounded-2xl border border-white/10 bg-[#131313]/95 p-3.5 shadow-2xl backdrop-blur-xl animate-fadeIn">
            <h4 className="text-xs font-semibold text-slate-300 mb-1.5 flex items-center justify-between">
              <span>Subtarea activa:</span>
              <span className="font-mono text-[10px] text-[#d1f107]">Paso {activeTask.step_current}</span>
            </h4>
            <p className="font-mono text-xs text-slate-300 rounded-xl bg-[#0a0a0a] p-2.5 border border-slate-800/80 leading-relaxed break-words">
              {activeTask.current_step_description || 'Procesando pipeline en segundo plano...'}
            </p>
          </div>
        )}
      </div>

      {pinRequest && (
        <HUDPinModal onAuthorize={handleAuthorizePIN} onReject={handleRejectPIN} />
      )}
    </>
  );
};
