import { useState, useEffect, useRef } from "react";
import { useAuthStore, type User } from "../../store/authStore";
import { useToastStore } from "../../store/toastStore";

const AVATAR_COLORS = [
  "#d1f107", // Electric Lime
  "#38bdf8", // Sky Blue
  "#a855f7", // Purple
  "#f43f5e", // Rose
  "#fb923c", // Orange
  "#34d399", // Emerald
  "#e879f9", // Pink
];

export default function ProfileSelectorModal() {
  const profiles = useAuthStore((s) => s.profiles);
  const fetchProfiles = useAuthStore((s) => s.fetchProfiles);
  const selectProfile = useAuthStore((s) => s.selectProfile);
  const createProfile = useAuthStore((s) => s.createProfile);
  const toast = useToastStore((s) => s.show);

  const [selectedProfile, setSelectedProfile] = useState<User | null>(null);
  const [pin, setPin] = useState("");
  const [isVerifying, setIsVerifying] = useState(false);
  const [pinError, setPinError] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);

  // New Profile Form State
  const [newName, setNewName] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [newRole, setNewRole] = useState("developer");
  const [newAvatarColor, setNewAvatarColor] = useState("#d1f107");
  const [newPin, setNewPin] = useState("");
  const [isCreating, setIsCreating] = useState(false);

  const pinInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    fetchProfiles();
  }, [fetchProfiles]);

  useEffect(() => {
    if (selectedProfile?.hasPin) {
      setTimeout(() => pinInputRef.current?.focus(), 100);
    }
  }, [selectedProfile]);

  const handleSelect = async (profile: User) => {
    if (!profile.hasPin) {
      setIsVerifying(true);
      const res = await selectProfile(profile);
      setIsVerifying(false);
      if (res.success) {
        toast(`Bienvenido, ${profile.name}`, "success");
      }
    } else {
      setSelectedProfile(profile);
      setPin("");
      setPinError(false);
    }
  };

  const handlePinSubmit = async (pinValue = pin) => {
    if (!selectedProfile || pinValue.length !== 6 || isVerifying) return;
    setIsVerifying(true);
    setPinError(false);

    const res = await selectProfile(selectedProfile, pinValue);
    setIsVerifying(false);

    if (res.success) {
      toast(`Sesión iniciada como ${selectedProfile.name}`, "success");
      setSelectedProfile(null);
      setPin("");
    } else {
      setPinError(true);
      setPin("");
      toast(res.error || "PIN de seguridad incorrecto", "error");
      setTimeout(() => setPinError(false), 800);
    }
  };

  const handleKeypadPress = (val: string) => {
    if (val === "clear") {
      setPin("");
    } else if (val === "backspace") {
      setPin((p) => p.slice(0, -1));
    } else if (pin.length < 6) {
      const nextPin = pin + val;
      setPin(nextPin);
      if (nextPin.length === 6) {
        handlePinSubmit(nextPin);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (/^[0-9]$/.test(e.key) && pin.length < 6) {
      const nextPin = pin + e.key;
      setPin(nextPin);
      if (nextPin.length === 6) {
        handlePinSubmit(nextPin);
      }
    } else if (e.key === "Backspace") {
      setPin((p) => p.slice(0, -1));
    } else if (e.key === "Enter" && pin.length === 6) {
      handlePinSubmit(pin);
    }
  };

  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim()) {
      toast("El nombre es requerido", "error");
      return;
    }
    if (newPin.trim() && newPin.trim().length !== 6) {
      toast("El PIN debe tener exactamente 6 dígitos", "error");
      return;
    }

    setIsCreating(true);
    const res = await createProfile({
      name: newName.trim(),
      email: newEmail.trim(),
      avatarColor: newAvatarColor,
      role: newRole,
      pin: newPin.trim(),
    });
    setIsCreating(false);

    if (res.success && res.user) {
      toast(`Perfil "${res.user.name}" creado con éxito`, "success");
      setShowCreateModal(false);
      setNewName("");
      setNewEmail("");
      setNewPin("");
    } else {
      toast(res.error || "Error al crear perfil", "error");
    }
  };

  return (
    <div className="fixed inset-0 z-[99999] bg-black/85 backdrop-blur-2xl flex items-center justify-center p-4 font-sans select-none animate-fadeIn">
      {/* Container Card */}
      <div className="w-full max-w-xl bg-[#181818] border border-white/10 rounded-3xl p-8 shadow-2xl shadow-black/80 flex flex-col items-center relative overflow-hidden">
        {/* Background glow accents */}
        <div className="absolute -top-24 -right-24 w-64 h-64 bg-[#d1f107]/5 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute -bottom-24 -left-24 w-64 h-64 bg-sky-500/5 rounded-full blur-3xl pointer-events-none" />

        {/* Top Logo */}
        <div className="w-12 h-12 rounded-2xl bg-[#222222] border border-white/10 flex items-center justify-center mb-4 shadow-md">
          <img src="/ozybaselogo.png" alt="OzyBase" className="w-7 h-7 object-contain" />
        </div>

        {/* Header Title */}
        <h1 className="text-[24px] font-bold text-white tracking-tight text-center mb-1">
          {selectedProfile ? selectedProfile.name : "¿Quién está usando OzyAssist?"}
        </h1>
        <p className="text-[13px] text-zinc-400 text-center mb-8 max-w-sm leading-relaxed">
          {selectedProfile
            ? "Ingresa el PIN de seguridad de 6 dígitos para acceder al perfil."
            : "Selecciona tu perfil de trabajo o crea uno nuevo con almacenamiento local seguro en SQLite."}
        </p>

        {/* PIN Entry View */}
        {selectedProfile ? (
          <div className="w-full max-w-xs flex flex-col items-center">
            {/* Hidden real input for physical keyboard focus */}
            <input
              ref={pinInputRef}
              type="password"
              inputMode="numeric"
              maxLength={6}
              value={pin}
              onChange={() => {}}
              onKeyDown={handleKeyDown}
              className="opacity-0 absolute -top-96 pointer-events-none"
              autoFocus
            />

            {/* Profile Avatar Badge */}
            <div
              style={{ backgroundColor: `${selectedProfile.avatarColor}20`, borderColor: selectedProfile.avatarColor }}
              className="w-16 h-16 rounded-2xl border-2 flex items-center justify-center mb-6 shadow-xl"
            >
              <span
                style={{ color: selectedProfile.avatarColor }}
                className="text-[20px] font-bold font-mono"
              >
                {selectedProfile.initials}
              </span>
            </div>

            {/* 6 Digit Visual Dots / Boxes */}
            <div
              className={`flex items-center gap-3 mb-8 transition-transform ${
                pinError ? "animate-bounce text-rose-400" : ""
              }`}
              onClick={() => pinInputRef.current?.focus()}
            >
              {[0, 1, 2, 3, 4, 5].map((idx) => {
                const filled = pin.length > idx;
                return (
                  <div
                    key={idx}
                    className={`w-10 h-12 rounded-xl border flex items-center justify-center font-mono text-[18px] font-bold transition-all shadow-inner ${
                      filled
                        ? "border-[#d1f107] bg-[#d1f107]/10 text-[#d1f107]"
                        : idx === pin.length
                        ? "border-white/40 bg-[#222222] ring-2 ring-[#d1f107]/30"
                        : "border-white/10 bg-[#202020] text-zinc-600"
                    }`}
                  >
                    {filled ? "•" : ""}
                  </div>
                );
              })}
            </div>

            {/* Virtual Keypad (0-9, Backspace) */}
            <div className="grid grid-cols-3 gap-2.5 w-full max-w-[240px] mb-6">
              {["1", "2", "3", "4", "5", "6", "7", "8", "9", "clear", "0", "backspace"].map(
                (key) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() => handleKeypadPress(key)}
                    disabled={isVerifying}
                    className="h-12 rounded-2xl bg-[#222222] hover:bg-[#2a2a2a] active:scale-95 border border-white/5 text-white font-mono text-[15px] font-semibold flex items-center justify-center transition-all shadow-sm disabled:opacity-50"
                  >
                    {key === "clear" ? (
                      <span className="text-[11px] font-sans text-zinc-400">C</span>
                    ) : key === "backspace" ? (
                      <span className="material-symbols-outlined text-[18px] text-zinc-400">
                        backspace
                      </span>
                    ) : (
                      key
                    )}
                  </button>
                )
              )}
            </div>

            {/* Back Button */}
            <button
              onClick={() => {
                setSelectedProfile(null);
                setPin("");
              }}
              className="text-[12px] text-zinc-400 hover:text-white flex items-center gap-1 transition-colors font-mono"
            >
              <span className="material-symbols-outlined text-[16px]">arrow_back</span>
              <span>Elegir otro perfil</span>
            </button>
          </div>
        ) : (
          /* Profile Cards List */
          <div className="w-full flex flex-col items-center">
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 w-full mb-8">
              {profiles.map((profile) => (
                <div
                  key={profile.id}
                  onClick={() => handleSelect(profile)}
                  className="group flex flex-col items-center p-5 rounded-2xl bg-[#202020] hover:bg-[#262626] border border-white/5 hover:border-[#d1f107]/40 cursor-pointer transition-all duration-200 shadow-md hover:shadow-xl hover:-translate-y-1 relative"
                >
                  {/* Avatar Icon */}
                  <div
                    style={{
                      backgroundColor: `${profile.avatarColor}20`,
                      borderColor: profile.avatarColor,
                    }}
                    className="w-14 h-14 rounded-2xl border-2 flex items-center justify-center mb-3 shadow-inner group-hover:scale-105 transition-transform"
                  >
                    <span
                      style={{ color: profile.avatarColor }}
                      className="text-[16px] font-bold font-mono"
                    >
                      {profile.initials}
                    </span>
                  </div>

                  {/* Profile Name */}
                  <span className="font-semibold text-white text-[13px] text-center truncate max-w-full group-hover:text-[#d1f107] transition-colors">
                    {profile.name}
                  </span>

                  {/* Role / Plan pill */}
                  <span className="text-[10px] text-zinc-400 font-mono mt-0.5 capitalize">
                    {profile.role || "developer"}
                  </span>

                  {/* Security Lock Badge */}
                  {profile.hasPin && (
                    <div
                      className="absolute top-2.5 right-2.5 p-1 rounded-md bg-white/5 border border-white/10 text-amber-400 flex items-center justify-center"
                      title="Protegido con PIN de 6 dígitos"
                    >
                      <span className="material-symbols-outlined text-[13px]">lock</span>
                    </div>
                  )}
                </div>
              ))}

              {/* + Create New Profile Card */}
              <button
                type="button"
                onClick={() => setShowCreateModal(true)}
                className="flex flex-col items-center justify-center p-5 rounded-2xl bg-white/[0.02] hover:bg-white/[0.05] border border-dashed border-white/15 hover:border-[#d1f107]/50 cursor-pointer transition-all group min-h-[140px]"
              >
                <div className="w-12 h-12 rounded-2xl bg-white/5 group-hover:bg-[#d1f107]/15 border border-white/10 group-hover:border-[#d1f107]/30 flex items-center justify-center text-zinc-400 group-hover:text-[#d1f107] mb-2 transition-all">
                  <span className="material-symbols-outlined text-[24px]">add</span>
                </div>
                <span className="font-semibold text-zinc-400 group-hover:text-white text-[12px] font-mono">
                  Nuevo Perfil
                </span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Modal: Create Profile */}
      {showCreateModal && (
        <div className="fixed inset-0 z-[100000] bg-black/90 backdrop-blur-xl flex items-center justify-center p-4">
          <div className="w-full max-w-md bg-[#1e1e1e] border border-white/10 rounded-3xl p-6 shadow-2xl flex flex-col animate-slideUp font-sans">
            <div className="flex items-center justify-between pb-3 mb-4 border-b border-white/10">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-[#d1f107] text-[20px]">
                  person_add
                </span>
                <h2 className="text-[16px] font-bold text-white">Crear Nuevo Perfil</h2>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="text-zinc-400 hover:text-white p-1 rounded"
              >
                <span className="material-symbols-outlined text-[18px]">close</span>
              </button>
            </div>

            <form onSubmit={handleCreateSubmit} className="flex flex-col gap-4">
              <div>
                <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
                  Nombre del Perfil *
                </label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="ej: Angel (Trabajo), Freelance, Personal..."
                  className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white text-[13px] outline-none focus:border-[#d1f107]/50"
                  required
                />
              </div>

              <div>
                <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
                  Rol / Especialidad
                </label>
                <input
                  type="text"
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value)}
                  placeholder="ej: Developer, Data Science, QA..."
                  className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white text-[13px] outline-none focus:border-[#d1f107]/50"
                />
              </div>

              <div>
                <label className="text-[11px] font-mono uppercase text-zinc-400 mb-1.5 block">
                  Color de Avatar
                </label>
                <div className="flex items-center gap-2.5">
                  {AVATAR_COLORS.map((c) => (
                    <button
                      key={c}
                      type="button"
                      onClick={() => setNewAvatarColor(c)}
                      style={{ backgroundColor: c }}
                      className={`w-7 h-7 rounded-xl transition-transform ${
                        newAvatarColor === c
                          ? "ring-2 ring-white scale-110 shadow-lg"
                          : "opacity-70 hover:opacity-100"
                      }`}
                    />
                  ))}
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-[11px] font-mono uppercase text-zinc-400 flex items-center gap-1">
                    <span className="material-symbols-outlined text-[14px] text-amber-400">
                      lock
                    </span>
                    <span>PIN de Seguridad (6 dígitos, opcional)</span>
                  </label>
                </div>
                <input
                  type="password"
                  inputMode="numeric"
                  maxLength={6}
                  value={newPin}
                  onChange={(e) => setNewPin(e.target.value.replace(/[^0-9]/g, ""))}
                  placeholder="6 dígitos numéricos (ej: 123456)"
                  className="w-full bg-[#2a2a2a] border border-white/10 rounded-xl px-3.5 py-2.5 text-white font-mono text-[13px] outline-none focus:border-[#d1f107]/50"
                />
                <span className="text-[10px] text-zinc-500 mt-1 block font-mono">
                  Si defines un PIN, se requerirá cada vez que inicies sesión en este perfil.
                </span>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-white/10 mt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-xl bg-white/5 hover:bg-white/10 text-white/70 text-[12px] font-mono transition-colors"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={isCreating}
                  className="px-5 py-2 rounded-xl bg-[#d1f107] hover:bg-[#b8d406] text-[#181e00] font-bold text-[12px] font-mono transition-all shadow-lg shadow-[#d1f107]/10 disabled:opacity-50"
                >
                  {isCreating ? "Creando..." : "Crear Perfil"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
