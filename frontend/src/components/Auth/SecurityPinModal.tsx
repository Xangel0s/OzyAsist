import { useState } from "react";
import { useAuthStore } from "../../store/authStore";
import { useToastStore } from "../../store/toastStore";

interface SecurityPinModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function SecurityPinModal({ isOpen, onClose }: SecurityPinModalProps) {
  const user = useAuthStore((s) => s.user);
  const updatePin = useAuthStore((s) => s.updatePin);
  const toast = useToastStore((s) => s.show);

  const [oldPin, setOldPin] = useState("");
  const [newPin, setNewPin] = useState("");
  const [confirmPin, setConfirmPin] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  if (!isOpen || !user) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (newPin && newPin.length !== 6) {
      toast("El nuevo PIN debe tener exactamente 6 dígitos", "error");
      return;
    }
    if (newPin !== confirmPin) {
      toast("Los PINs no coinciden", "error");
      return;
    }

    setIsLoading(true);
    const res = await updatePin(newPin, oldPin);
    setIsLoading(false);

    if (res.success) {
      toast(
        newPin ? "PIN de seguridad de 6 dígitos actualizado" : "PIN de seguridad eliminado",
        "success"
      );
      onClose();
      setOldPin("");
      setNewPin("");
      setConfirmPin("");
    } else {
      toast(res.error || "Error al actualizar el PIN", "error");
    }
  };

  return (
    <div className="fixed inset-0 z-[100000] bg-black/85 backdrop-blur-xl flex items-center justify-center p-4 font-sans select-none animate-fadeIn">
      <div className="w-full max-w-md bg-[#1e1e1e] border border-white/10 rounded-3xl p-6 shadow-2xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between pb-3 mb-4 border-b border-white/10">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-amber-400 text-[20px]">
              lock
            </span>
            <h2 className="text-[16px] font-bold text-white">
              {user.hasPin ? "Modificar PIN de Seguridad" : "Configurar PIN de 6 Dígitos"}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="text-zinc-400 hover:text-white p-1 rounded transition-colors"
          >
            <span className="material-symbols-outlined text-[18px]">close</span>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {user.hasPin && (
            <div>
              <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
                PIN Actual (6 dígitos) *
              </label>
              <input
                type="password"
                inputMode="numeric"
                maxLength={6}
                value={oldPin}
                onChange={(e) => setOldPin(e.target.value.replace(/[^0-9]/g, ""))}
                placeholder="Ingresa tu PIN actual"
                className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white font-mono text-[13px] outline-none focus:border-[#d1f107]/50"
                required
              />
            </div>
          )}

          <div>
            <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
              Nuevo PIN (6 dígitos, dejar vacío para quitar)
            </label>
            <input
              type="password"
              inputMode="numeric"
              maxLength={6}
              value={newPin}
              onChange={(e) => setNewPin(e.target.value.replace(/[^0-9]/g, ""))}
              placeholder="6 dígitos numéricos"
              className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white font-mono text-[13px] outline-none focus:border-[#d1f107]/50"
            />
          </div>

          {newPin && (
            <div>
              <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
                Confirmar Nuevo PIN
              </label>
              <input
                type="password"
                inputMode="numeric"
                maxLength={6}
                value={confirmPin}
                onChange={(e) => setConfirmPin(e.target.value.replace(/[^0-9]/g, ""))}
                placeholder="Repite los 6 dígitos"
                className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white font-mono text-[13px] outline-none focus:border-[#d1f107]/50"
                required={!!newPin}
              />
            </div>
          )}

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-white/10 mt-2 font-mono">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl bg-white/5 hover:bg-white/10 text-white/70 text-[12px] transition-colors"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-5 py-2 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[12px] transition-all shadow-lg shadow-[#d1f107]/10 disabled:opacity-50"
            >
              {isLoading ? "Guardando..." : "Guardar PIN"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
