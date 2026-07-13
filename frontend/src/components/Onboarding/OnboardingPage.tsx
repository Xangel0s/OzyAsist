import { useUIStore } from "../../store/uiStore";
import { useAuthStore } from "../../store/authStore";
import MemoryImportStep from "./MemoryImportStep";
import FeedbackStep from "./FeedbackStep";
import InstructionsStep from "./InstructionsStep";

export default function OnboardingPage() {
  const onboardingStep = useUIStore((s) => s.onboardingStep);
  const setOnboardingStep = useUIStore((s) => s.setOnboardingStep);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const completeOnboarding = useAuthStore((s) => s.completeOnboarding);

  const handleComplete = () => {
    setOnboardingStep(3);
    completeOnboarding();
    setActiveView("home");
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-surface-deep overflow-y-auto">
      <div className="max-w-content-max-width mx-auto w-full px-gutter py-12">
        <div className="flex items-center justify-center gap-3 mb-4">
          <div className="w-14 h-14 rounded-xl flex items-center justify-center overflow-hidden">
            <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
          </div>
          <h1 className="text-headline-md font-headline-md text-on-surface">
            Configuración inicial
          </h1>
        </div>

        <div className="flex items-center justify-center gap-2 mb-12">
          {[0, 1, 2].map((step) => (
            <div key={step} className="flex items-center gap-2">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-label-caps transition-colors ${
                  onboardingStep > step
                    ? "bg-primary-container text-on-primary"
                    : onboardingStep === step
                      ? "bg-surface-container-high text-on-surface border border-primary-container"
                      : "bg-surface-variant text-text-muted"
                }`}
              >
                {onboardingStep > step ? (
                  <span className="material-symbols-outlined text-[16px] fill">
                    check
                  </span>
                ) : (
                  step + 1
                )}
              </div>
              <span className="text-label-caps text-text-muted hidden sm:inline">
                {step === 0
                  ? "Memoria"
                  : step === 1
                    ? "Feedback"
                    : "Instrucciones"}
              </span>
              {step < 2 && (
                <div className="w-8 h-px bg-border-subtle mx-1" />
              )}
            </div>
          ))}
        </div>

        {onboardingStep === 0 && (
          <MemoryImportStep onNext={() => setOnboardingStep(1)} />
        )}
        {onboardingStep === 1 && (
          <FeedbackStep
            onNext={() => setOnboardingStep(2)}
            onBack={() => setOnboardingStep(0)}
          />
        )}
        {onboardingStep === 2 && (
          <InstructionsStep
            onComplete={handleComplete}
            onBack={() => setOnboardingStep(1)}
          />
        )}
      </div>
    </div>
  );
}
