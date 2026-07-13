import { useState } from "react";
import { useAuthStore } from "../../store/authStore";
import { useUIStore } from "../../store/uiStore";

export default function WelcomeScreen() {
  const setUser = useAuthStore((s) => s.setUser);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const setOnboardingStep = useUIStore((s) => s.setOnboardingStep);
  const [name, setName] = useState("");
  const [role, setRole] = useState("");
  const [hasUsedAI, setHasUsedAI] = useState(false);

  const handleStart = () => {
    if (!name.trim()) return;
    const initials = name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w[0])
      .join("")
      .toUpperCase();
    setUser({
      id: crypto.randomUUID(),
      name: name.trim(),
      initials,
      role: role || "Desarrollador",
      plan: "free",
      hasUsedAI,
    });
    setOnboardingStep(0);
    setActiveView("onboarding");
  };

  return (
    <div className="h-screen w-screen flex items-center justify-center bg-background">
      <div className="max-w-md w-full mx-4">
        <div className="flex flex-col items-center mb-10">
          <div className="w-24 h-24 rounded-2xl flex items-center justify-center mb-6 overflow-hidden">
            <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
          </div>
          <h1 className="text-headline-lg text-on-surface font-headline-lg text-center">
            Bienvenido a Ozy
          </h1>
          <p className="text-text-muted text-body-md text-center mt-2">
            Tu asistente de IA para desarrollo y productividad.
          </p>
        </div>

        <div className="bg-surface-elevated rounded-2xl border border-border-subtle p-6 flex flex-col gap-5">
          <div className="flex flex-col gap-1.5">
            <label className="text-label-caps text-text-muted">Tu nombre</label>
            <input
              className="w-full bg-surface-container-high border border-border-subtle rounded-lg px-4 py-2.5 text-on-surface text-body-md placeholder:text-text-muted outline-none focus:border-primary-container focus:ring-1 focus:ring-primary-container transition-all"
              placeholder="Ej: Juan Pérez"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-label-caps text-text-muted">Tu rol</label>
            <select
              className="w-full bg-surface-container-high border border-border-subtle rounded-lg px-4 py-2.5 text-on-surface text-body-md outline-none focus:border-primary-container focus:ring-1 focus:ring-primary-container transition-all appearance-none"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <option value="">Selecciona tu rol</option>
              <option value="Desarrollador">Desarrollador</option>
              <option value="Arquitecto">Arquitecto</option>
              <option value="DevOps">DevOps</option>
              <option value="Product Manager">Product Manager</option>
              <option value="Diseñador">Diseñador</option>
              <option value="Estudiante">Estudiante</option>
              <option value="Otro">Otro</option>
            </select>
          </div>

          <div className="flex items-center justify-between">
            <span className="text-body-md text-on-surface">
              ¿Has usado otra IA antes?
            </span>
            <button
              className={`w-12 h-6 rounded-full transition-colors relative ${
                hasUsedAI ? "bg-primary-container" : "bg-surface-variant"
              }`}
              onClick={() => setHasUsedAI(!hasUsedAI)}
            >
              <div
                className={`w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all ${
                  hasUsedAI ? "left-[26px]" : "left-0.5"
                }`}
              />
            </button>
          </div>

          <button
            className="w-full mt-2 px-4 py-2.5 bg-primary-container text-on-primary rounded-lg hover:bg-primary-fixed-dim transition-colors text-body-md font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={!name.trim()}
            onClick={handleStart}
          >
            Comenzar
          </button>
        </div>
      </div>
    </div>
  );
}
