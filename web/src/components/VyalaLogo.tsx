// VYALA Logo components — using the official SVG assets from /public

/** The lion-shield icon (transparent background version for UI use) */
export function VyalaLogo({ className = 'h-8 w-8' }: { className?: string; monochrome?: boolean }) {
  return (
    <img
      src="/vyala-icon.svg"
      alt="VYALA Shield"
      className={className}
      draggable={false}
    />
  );
}

export const VyalaShieldLogo = VyalaLogo;

/** The VYALA wordmark — rendered inline using SVG text for crisp quality at any size */
export function VyalaWordmark({ className = '' }: { className?: string }) {
  return (
    <span
      className={`font-display font-extrabold tracking-[0.18em] text-white uppercase select-none ${className}`}
      style={{ fontFamily: "'Syne', sans-serif", letterSpacing: '0.18em' }}
    >
      <span style={{ color: '#FFFFFF' }}>VY</span>
      <span style={{ color: '#00F5D4' }}>A</span>
      <span style={{ color: '#FFFFFF' }}>L</span>
      <span style={{ color: '#00F5D4' }}>A</span>
    </span>
  );
}

/** Full logo: shield icon + VYALA wordmark side by side */
export function VyalaFullLogo({
  className = '',
  tagline = false,
}: {
  className?: string;
  tagline?: boolean;
}) {
  return (
    <div className={`flex flex-col items-start gap-0.5 ${className}`}>
      <div className="flex items-center gap-2.5">
        <VyalaLogo className="h-8 w-8 drop-shadow-[0_0_8px_rgba(0,245,212,0.5)]" />
        <VyalaWordmark className="text-xl" />
      </div>
      {tagline && (
        <p
          className="ml-10 font-mono text-[9px] tracking-[0.25em] text-[#00F5D4] uppercase opacity-75"
          style={{ fontFamily: "'JetBrains Mono', monospace" }}
        >
          Guarding the Future of Cryptography
        </p>
      )}
    </div>
  );
}