import React, { useState, useRef, useEffect } from 'react';
import { ShieldAlert, X } from 'lucide-react';
import { useTaskStore } from '../../store/taskStore';

interface HUDPinModalProps {
  onAuthorize: (pin: string) => Promise<void>;
  onReject: () => void;
}

export const HUDPinModal: React.FC<HUDPinModalProps> = ({ onAuthorize, onReject }) => {
  const pinRequest = useTaskStore((s) => s.pinRequest);
  const [pin, setPin] = useState(['', '', '', '', '', '']);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputsRef = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    inputsRef.current[0]?.focus();
  }, []);

  if (!pinRequest) return null;

  const handleChange = (index: number, val: string) => {
    if (!/^\d*$/.test(val)) return;
    const newPin = [...pin];
    newPin[index] = val.slice(-1);
    setPin(newPin);
    setError(null);

    if (val && index < 5) {
      inputsRef.current[index + 1]?.focus();
    }

    // Auto-envío en el 6to dígito
    if (index === 5 && val) {
      const fullPin = newPin.join('');
      if (fullPin.length === 6) {
        submitPIN(fullPin);
      }
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !pin[index] && index > 0) {
      inputsRef.current[index - 1]?.focus();
    }
  };

  const submitPIN = async (fullPin: string) => {
    setLoading(true);
    try {
      await onAuthorize(fullPin);
    } catch (err: any) {
      setError(err?.message || 'PIN incorrecto');
      setPin(['', '', '', '', '', '']);
      inputsRef.current[0]?.focus();
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-fadeIn">
      <div className="w-full max-w-md rounded-2xl border border-red-500/30 bg-[#131313] p-6 shadow-2xl">
        <div className="flex items-center justify-between border-b border-surface-border pb-4">
          <div className="flex items-center gap-3 text-red-400">
            <ShieldAlert className="h-6 w-6" />
            <h3 className="font-semibold text-white">Autorización Requerida</h3>
          </div>
          <button onClick={onReject} className="text-slate-400 hover:text-white transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="my-4 space-y-3">
          <p className="text-sm text-slate-300">{pinRequest.reason}</p>
          {pinRequest.command && (
            <pre className="rounded-lg bg-[#0a0a0a] p-3 font-mono text-xs text-amber-300 border border-slate-800 overflow-x-auto">
              {pinRequest.command}
            </pre>
          )}
        </div>

        <div className="my-6">
          <label className="mb-2 block text-center text-xs font-medium text-slate-400">
            Ingresa tu PIN de seguridad (6 dígitos)
          </label>
          <div className="flex justify-center gap-2">
            {pin.map((digit, idx) => (
              <input
                key={idx}
                ref={(el) => {
                  inputsRef.current[idx] = el;
                }}
                type="password"
                inputMode="numeric"
                maxLength={1}
                value={digit}
                disabled={loading}
                onChange={(e) => handleChange(idx, e.target.value)}
                onKeyDown={(e) => handleKeyDown(idx, e)}
                className="h-12 w-10 rounded-lg border border-slate-700 bg-slate-800/80 text-center font-mono text-lg font-bold text-white focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500 disabled:opacity-50 transition-all"
              />
            ))}
          </div>
          {error && <p className="mt-2 text-center text-xs text-red-400 font-medium">{error}</p>}
        </div>

        <div className="flex justify-end gap-2 border-t border-surface-border pt-4">
          <button
            onClick={onReject}
            disabled={loading}
            className="rounded-lg px-4 py-2 text-sm text-slate-300 hover:bg-slate-800 transition-colors"
          >
            Rechazar Tarea
          </button>
        </div>
      </div>
    </div>
  );
};
