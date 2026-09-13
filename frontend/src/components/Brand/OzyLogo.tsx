import React from 'react';

interface OzyLogoProps {
  size?: number;
  className?: string;
  isThinking?: boolean;
}

export const OzyLogo: React.FC<OzyLogoProps> = ({
  size = 28,
  className = '',
  isThinking = false,
}) => {
  return (
    <div
      style={{ width: size, height: size }}
      className={`relative flex items-center justify-center flex-shrink-0 ${className}`}
    >
      <img
        src="/ozybaselogo.png"
        alt="Ozy"
        className={`w-full h-full object-contain select-none transition-all duration-300 ${
          isThinking ? 'animate-pulse scale-105 drop-shadow-[0_0_10px_rgba(209,241,7,0.5)]' : ''
        }`}
        draggable={false}
      />
      {isThinking && (
        <span className="absolute inset-0 rounded-full bg-[#d1f107]/20 blur-md animate-pulse pointer-events-none" />
      )}
    </div>
  );
};

export default OzyLogo;
