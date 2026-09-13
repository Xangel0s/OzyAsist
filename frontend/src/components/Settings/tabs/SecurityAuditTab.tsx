import { useState, useEffect, useCallback } from "react";
import { api, type AuditEntryDTO, type ShadowSnapshotDTO } from "../../../services/api";
import { useToastStore } from "../../../store/toastStore";
import { ShieldCheck, RefreshCw, AlertTriangle, Undo2, Terminal, CheckCircle2, Flame } from "lucide-react";

export default function SecurityAuditTab() {
  const toast = useToastStore((s) => s.show);

  const [loading, setLoading] = useState(false);
  const [isValid, setIsValid] = useState(true);
  const [totalVerified, setTotalVerified] = useState(0);
  const [logs, setLogs] = useState<AuditEntryDTO[]>([]);

  // Time-Travel Undo State
  const [undoTaskId, setUndoTaskId] = useState("");
  const [snapshots, setSnapshots] = useState<ShadowSnapshotDTO[]>([]);
  const [loadingSnapshots, setLoadingSnapshots] = useState(false);
  const [reverting, setReverting] = useState(false);

  const loadAuditTrail = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.security.getAuditTrail();
      setIsValid(data.is_valid);
      setTotalVerified(data.total_verified);
      setLogs(data.logs || []);
    } catch (err: any) {
      toast(err.message || "Error al cargar la auditoría", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    loadAuditTrail();
  }, [loadAuditTrail]);

  const handleSearchSnapshots = async () => {
    if (!undoTaskId.trim()) return;
    setLoadingSnapshots(true);
    try {
      const data = await api.security.getTaskSnapshots(undoTaskId.trim());
      setSnapshots(data.snapshots || []);
      if ((data.snapshots || []).length === 0) {
        toast("No se encontraron snapshots para esta tarea", "info");
      }
    } catch (err: any) {
      toast(err.message || "Error al buscar snapshots", "error");
    } finally {
      setLoadingSnapshots(false);
    }
  };

  const handleRollbackTask = async () => {
    if (!undoTaskId.trim()) return;
    setReverting(true);
    try {
      const res = await api.security.undoTask(undoTaskId.trim());
      toast(res.message || "Archivos restaurados con éxito", "success");
      loadAuditTrail();
      setSnapshots([]);
      setUndoTaskId("");
    } catch (err: any) {
      toast(err.message || "Error en el rollback", "error");
    } finally {
      setReverting(false);
    }
  };

  const handleTriggerPanicKill = async () => {
    try {
      await api.post("/tasks/emergency-kill");
      toast("Kill-Switch activado. Procesos detenidos.", "warning");
      loadAuditTrail();
    } catch (err: any) {
      toast(err.message || "Error al activar Kill-Switch", "error");
    }
  };

  const formatHash = (h: string) => {
    if (!h || h.length < 12) return h;
    return `${h.slice(0, 6)}...${h.slice(-6)}`;
  };

  return (
    <div className="flex flex-col gap-6 max-w-2xl font-sans text-neutral-200">
      {/* 1. Banner de Integridad Criptográfica */}
      <div className="p-4 rounded-2xl bg-black/40 border border-[#d1f107]/20 flex items-center justify-between shadow-lg">
        <div className="flex items-center gap-3.5">
          <div className="w-10 h-10 rounded-xl bg-[#d1f107]/10 border border-[#d1f107]/30 flex items-center justify-center text-[#d1f107]">
            <ShieldCheck className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-[15px] font-bold text-white">Cadena de Auditoría Inmutable</h3>
              {isValid ? (
                <span className="flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-semibold">
                  <CheckCircle2 className="w-3 h-3" /> Verificada
                </span>
              ) : (
                <span className="flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20 font-semibold">
                  <AlertTriangle className="w-3 h-3" /> Ruptura detectada
                </span>
              )}
            </div>
            <p className="text-xs text-neutral-400 mt-0.5">
              Sellado criptográfico SHA-256 (Hash Chaining) • {totalVerified} bloques auditados
            </p>
          </div>
        </div>

        <button
          type="button"
          onClick={loadAuditTrail}
          disabled={loading}
          className="p-2 rounded-xl bg-white/5 hover:bg-white/10 text-neutral-300 hover:text-white transition-colors disabled:opacity-50"
          title="Re-auditar cadena en vivo"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin text-[#d1f107]" : ""}`} />
        </button>
      </div>

      {/* 2. Control de Emergencia & Watchdog EDR */}
      <div className="p-4 rounded-2xl bg-black/30 border border-white/5 flex items-center justify-between">
        <div>
          <h4 className="text-[14px] font-semibold text-white flex items-center gap-2">
            <Flame className="w-4 h-4 text-amber-400" />
            Watchdog EDR & Freno de Pánico
          </h4>
          <p className="text-xs text-neutral-400 mt-0.5">
            Detección activa de binarios anómalos en directorios temporales y Kill-Switch instantáneo.
          </p>
        </div>
        <button
          type="button"
          onClick={handleTriggerPanicKill}
          className="px-3.5 py-2 rounded-xl bg-rose-600/20 hover:bg-rose-600/30 text-rose-300 hover:text-rose-200 border border-rose-500/30 text-xs font-semibold transition-colors flex items-center gap-1.5"
        >
          <AlertTriangle className="w-3.5 h-3.5" />
          <span>Detener Procesos</span>
        </button>
      </div>

      {/* 3. Time-Travel Undo Engine */}
      <div className="flex flex-col gap-3 p-4 rounded-2xl bg-black/30 border border-white/5">
        <h4 className="text-[14px] font-semibold text-white flex items-center gap-2">
          <Undo2 className="w-4 h-4 text-[#d1f107]" />
          Reversión de Tareas (Time-Travel Rollback)
        </h4>
        <p className="text-xs text-neutral-400 leading-relaxed">
          Restaura los archivos modificados por el agente a su estado exacto antes de la ejecución de una tarea.
        </p>

        <div className="flex items-center gap-2 mt-1">
          <input
            type="text"
            placeholder="Introduce el ID de la tarea (ej: task-679)..."
            value={undoTaskId}
            onChange={(e) => setUndoTaskId(e.target.value)}
            className="flex-1 bg-[#131313] border border-white/10 rounded-xl px-3.5 py-2 text-xs text-white placeholder-neutral-500 outline-none focus:border-[#d1f107]/50 transition-colors font-mono"
          />
          <button
            type="button"
            onClick={handleSearchSnapshots}
            disabled={loadingSnapshots || !undoTaskId.trim()}
            className="px-3 py-2 rounded-xl bg-white/10 hover:bg-white/15 text-white text-xs font-medium transition-colors disabled:opacity-40"
          >
            {loadingSnapshots ? "Buscando..." : "Buscar Snapshots"}
          </button>
        </div>

        {snapshots.length > 0 && (
          <div className="mt-2 flex flex-col gap-2 pt-2 border-t border-white/5">
            <div className="text-xs text-neutral-400 font-medium">
              Se encontraron {snapshots.length} archivos respaldados:
            </div>
            <div className="flex flex-col gap-1 max-h-32 overflow-y-auto font-mono text-[11px] text-neutral-300">
              {snapshots.map((s) => (
                <div key={s.id} className="p-1.5 rounded bg-white/5 border border-white/5 truncate">
                  {s.file_path}
                </div>
              ))}
            </div>
            <button
              type="button"
              onClick={handleRollbackTask}
              disabled={reverting}
              className="mt-1 self-start px-4 py-2 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-xs transition-colors flex items-center gap-1.5 shadow-md disabled:opacity-50"
            >
              <Undo2 className="w-3.5 h-3.5" />
              <span>{reverting ? "Restaurando..." : "Ejecutar Reversión Ahora"}</span>
            </button>
          </div>
        )}
      </div>

      {/* 4. Tabla de Registros Inmutables Recientes */}
      <div className="flex flex-col gap-3">
        <h4 className="text-[14px] font-semibold text-white flex items-center gap-2">
          <Terminal className="w-4 h-4 text-neutral-400" />
          Registros Criptográficos Recientes
        </h4>

        {logs.length === 0 ? (
          <div className="p-6 text-center text-xs text-neutral-500 rounded-xl bg-black/20 border border-white/5 font-mono">
            No hay registros de auditoría aún.
          </div>
        ) : (
          <div className="flex flex-col gap-1.5 max-h-72 overflow-y-auto pr-1">
            {logs.map((log) => (
              <div
                key={log.id}
                className="p-2.5 rounded-xl bg-black/25 border border-white/5 flex flex-col gap-1 text-xs hover:border-white/10 transition-colors font-mono"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="px-1.5 py-0.5 rounded bg-white/10 text-[10px] text-[#d1f107] font-semibold">
                      #{log.id} {log.agent}
                    </span>
                    <span className="text-white font-medium text-[11px]">{log.action}</span>
                  </div>
                  <span className="text-[10px] text-neutral-500">
                    {new Date(log.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                  </span>
                </div>
                <div className="text-[11px] text-neutral-400 font-sans truncate">
                  {log.details}
                </div>
                <div className="flex items-center gap-3 text-[10px] text-neutral-500 pt-1 border-t border-white/5 font-mono">
                  <span>Prev: {formatHash(log.prev_hash)}</span>
                  <span>→</span>
                  <span className="text-neutral-400">Hash: {formatHash(log.record_hash)}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
