import { useEffect, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent, ReactNode, MouseEvent as ReactMouseEvent } from 'react';
import {
  Code2,
  Package,
  Boxes,
  Radio,
  CheckCircle2,
  Loader2,
  Menu,
  X,
  Sparkles,
  ArrowRight,
  Download,
  Copy,
  Check,
  Zap,
  Crown,
  Sliders,
  ShieldAlert,
  Ticket,
  GitBranch,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { supabase } from '../lib/supabase';
import { VyalaFullLogo } from '../components/VyalaLogo';

// ==========================================
// LUXURY NEUTRAL & CYBER-AESTHETIC STYLES
// ==========================================
function GlobalAestheticStyles() {
  return (
    <style>{`
      @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Plus+Jakarta+Sans:wght@300;400;500;600;700;800&family=Syne:wght@600;700;800&display=swap');

      :root {
        --void-bg: #07090E;
        --surface-1: #0D111A;
        --surface-2: #141A26;
        --surface-3: #1E2638;
        
        --border-slate: rgba(255, 255, 255, 0.08);
        --border-cyan: rgba(0, 245, 212, 0.25);
        --border-purple: rgba(123, 110, 246, 0.25);

        --cyber-cyan: #00F5D4;
        --cyber-blue: #00C8FF;
        --royal-purple: #7B6EF6;
        --risk-red: #FF4D4D;

        --gradient-cyber: linear-gradient(135deg, #00F5D4 0%, #00C8FF 50%, #7B6EF6 100%);
        --gradient-silver: linear-gradient(135deg, #FFFFFF 0%, #CBD5E1 50%, #94A3B8 100%);

        --font-display: 'Syne', sans-serif;
        --font-sans: 'Plus Jakarta Sans', sans-serif;
        --font-mono: 'JetBrains Mono', monospace;
      }

      html {
        scroll-behavior: smooth;
        background-color: var(--void-bg);
      }

      /* Custom modern scrollbar */
      ::-webkit-scrollbar {
        width: 8px;
      }
      ::-webkit-scrollbar-track {
        background: #07090E;
      }
      ::-webkit-scrollbar-thumb {
        background: rgba(0, 245, 212, 0.25);
        border-radius: 4px;
      }
      ::-webkit-scrollbar-thumb:hover {
        background: rgba(0, 245, 212, 0.5);
      }

      ::selection {
        background: #00F5D4;
        color: #07090E;
      }

      /* Shimmer text and metallic effects */
      .cyber-text-shimmer {
        background: linear-gradient(90deg, #FFFFFF 0%, #00F5D4 30%, #00C8FF 60%, #7B6EF6 90%, #FFFFFF 100%);
        background-size: 200% auto;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: cyberShimmer 6s linear infinite;
      }

      .cyan-border-glow {
        box-shadow: 0 0 25px -5px rgba(0, 245, 212, 0.25);
      }

      .glass-card {
        background: rgba(13, 17, 26, 0.75);
        backdrop-filter: blur(16px);
        -webkit-backdrop-filter: blur(16px);
        border: 1px solid rgba(255, 255, 255, 0.08);
      }
      .glass-card:hover {
        border-color: rgba(0, 245, 212, 0.35);
        box-shadow: 0 12px 40px -10px rgba(0, 245, 212, 0.15);
      }

      @keyframes cyberShimmer {
        0% { background-position: 0% center; }
        100% { background-position: 200% center; }
      }

      @keyframes floatSlow {
        0%, 100% { transform: translateY(0px) rotate(0deg); }
        50% { transform: translateY(-10px) rotate(0.4deg); }
      }

      .animate-float {
        animation: floatSlow 6s ease-in-out infinite;
      }

      /* Custom interactive cursor ring sits as background aura without suppressing system pointer */
      .custom-cursor-active {
        cursor: auto;
      }
    `}</style>
  );
}

// ==========================================
// INTERACTIVE HTML5 CANVAS (BACKGROUND PHYSICS)
// ==========================================
function LuxuryInteractiveCanvas() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let width = (canvas.width = window.innerWidth);
    let height = (canvas.height = window.innerHeight);

    const handleResize = () => {
      if (!canvas) return;
      width = canvas.width = window.innerWidth;
      height = canvas.height = window.innerHeight;
    };
    window.addEventListener('resize', handleResize);

    const mouse = {
      x: width / 2,
      y: height / 2,
      targetX: width / 2,
      targetY: height / 2,
      active: false,
    };

    const handleMouseMove = (e: MouseEvent) => {
      mouse.targetX = e.clientX;
      mouse.targetY = e.clientY;
      mouse.active = true;
    };
    window.addEventListener('mousemove', handleMouseMove);

    const count = Math.min(Math.floor((width * height) / 18000), 70);
    const nodes: Array<{
      x: number;
      y: number;
      vx: number;
      vy: number;
      radius: number;
      baseAlpha: number;
      cyan: boolean;
    }> = [];

    for (let i = 0; i < count; i++) {
      nodes.push({
        x: Math.random() * width,
        y: Math.random() * height,
        vx: (Math.random() - 0.5) * 0.4,
        vy: (Math.random() - 0.5) * 0.4,
        radius: Math.random() * 2 + 1,
        baseAlpha: Math.random() * 0.4 + 0.2,
        cyan: Math.random() > 0.35,
      });
    }

    const render = () => {
      mouse.x += (mouse.targetX - mouse.x) * 0.08;
      mouse.y += (mouse.targetY - mouse.y) * 0.08;

      ctx.clearRect(0, 0, width, height);

      if (mouse.active) {
        const radGrad = ctx.createRadialGradient(
          mouse.x,
          mouse.y,
          0,
          mouse.x,
          mouse.y,
          350
        );
        radGrad.addColorStop(0, 'rgba(0, 245, 212, 0.09)');
        radGrad.addColorStop(0.5, 'rgba(123, 110, 246, 0.04)');
        radGrad.addColorStop(1, 'rgba(7, 9, 14, 0)');
        ctx.fillStyle = radGrad;
        ctx.fillRect(0, 0, width, height);
      }

      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i];
        n.x += n.vx;
        n.y += n.vy;

        if (n.x < 0) n.x = width;
        if (n.x > width) n.x = 0;
        if (n.y < 0) n.y = height;
        if (n.y > height) n.y = 0;

        const dx = mouse.x - n.x;
        const dy = mouse.y - n.y;
        const dist = Math.sqrt(dx * dx + dy * dy);

        if (dist < 140) {
          const force = (140 - dist) / 140;
          n.x -= (dx / dist) * force * 1.5;
          n.y -= (dy / dist) * force * 1.5;
        }

        ctx.beginPath();
        ctx.arc(n.x, n.y, n.radius, 0, Math.PI * 2);
        ctx.fillStyle = n.cyan
          ? `rgba(0, 245, 212, ${n.baseAlpha})`
          : `rgba(123, 110, 246, ${n.baseAlpha})`;
        ctx.fill();

        for (let j = i + 1; j < nodes.length; j++) {
          const n2 = nodes[j];
          const ldx = n.x - n2.x;
          const ldy = n.y - n2.y;
          const ldist = Math.sqrt(ldx * ldx + ldy * ldy);

          if (ldist < 130) {
            const lineAlpha = (1 - ldist / 130) * 0.22;
            ctx.beginPath();
            ctx.moveTo(n.x, n.y);
            ctx.lineTo(n2.x, n2.y);

            const midX = (n.x + n2.x) / 2;
            const midY = (n.y + n2.y) / 2;
            const mouseDistToLine = Math.sqrt(
              Math.pow(mouse.x - midX, 2) + Math.pow(mouse.y - midY, 2)
            );

            if (mouseDistToLine < 160) {
              ctx.strokeStyle = `rgba(0, 245, 212, ${lineAlpha * 2.5})`;
              ctx.lineWidth = 1;
            } else {
              ctx.strokeStyle = `rgba(148, 163, 184, ${lineAlpha})`;
              ctx.lineWidth = 0.5;
            }
            ctx.stroke();
          }
        }
      }

      animationFrameId = requestAnimationFrame(render);
    };

    render();

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('mousemove', handleMouseMove);
      cancelAnimationFrame(animationFrameId);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="pointer-events-none fixed inset-0 z-0 h-full w-full opacity-80"
    />
  );
}

// ==========================================
// DUAL-RING CUSTOM MOUSE CURSOR
// ==========================================
function CustomLuxuryCursor() {
  const dotRef = useRef<HTMLDivElement | null>(null);
  const ringRef = useRef<HTMLDivElement | null>(null);
  const [cursorText, setCursorText] = useState('');
  const [isHovered, setIsHovered] = useState(false);

  useEffect(() => {
    let mouseX = -100;
    let mouseY = -100;
    let ringX = -100;
    let ringY = -100;
    let animId: number;

    const onMouseMove = (e: MouseEvent) => {
      mouseX = e.clientX;
      mouseY = e.clientY;

      if (dotRef.current) {
        dotRef.current.style.transform = `translate3d(${mouseX}px, ${mouseY}px, 0)`;
      }

      const target = e.target as HTMLElement | null;
      if (target) {
        const interactive = target.closest('[data-cursor]');
        if (interactive) {
          const attr = interactive.getAttribute('data-cursor');
          setCursorText(attr || '');
          setIsHovered(true);
        } else if (
          target.closest('a, button, input, textarea, select, [role="button"]')
        ) {
          setCursorText('');
          setIsHovered(true);
        } else {
          setIsHovered(false);
          setCursorText('');
        }
      }
    };

    const render = () => {
      ringX += (mouseX - ringX) * 0.15;
      ringY += (mouseY - ringY) * 0.15;

      if (ringRef.current) {
        ringRef.current.style.transform = `translate3d(${ringX}px, ${ringY}px, 0)`;
      }
      animId = requestAnimationFrame(render);
    };

    window.addEventListener('mousemove', onMouseMove);
    render();

    return () => {
      window.removeEventListener('mousemove', onMouseMove);
      cancelAnimationFrame(animId);
    };
  }, []);

  return (
    <>
      <div
        ref={dotRef}
        className="pointer-events-none fixed top-0 left-0 z-50 hidden -translate-x-1/2 -translate-y-1/2 rounded-full bg-[#00F5D4] shadow-[0_0_12px_#00F5D4] transition-opacity duration-300 lg:block"
        style={{
          width: isHovered ? '6px' : '8px',
          height: isHovered ? '6px' : '8px',
          opacity: 0.9,
        }}
      />

      <div
        ref={ringRef}
        className="pointer-events-none fixed top-0 left-0 z-50 hidden -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border border-[#00F5D4]/50 bg-[#00F5D4]/5 backdrop-blur-[1px] transition-all duration-200 ease-out lg:flex"
        style={{
          width: isHovered ? (cursorText ? '80px' : '48px') : '32px',
          height: isHovered ? (cursorText ? '80px' : '48px') : '32px',
          borderColor: isHovered ? 'rgba(0, 245, 212, 0.9)' : 'rgba(0, 245, 212, 0.35)',
        }}
      >
        {cursorText && (
          <span className="font-mono text-[9px] font-bold tracking-widest text-[#00F5D4] uppercase">
            {cursorText}
          </span>
        )}
      </div>
    </>
  );
}

// ==========================================
// 3D CARD TILT & GLARE EFFECT COMPONENT
// ==========================================
function TiltCard({
  children,
  className = '',
  dataCursor = 'explore',
}: {
  children: ReactNode;
  className?: string;
  dataCursor?: string;
}) {
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [transform, setTransform] = useState('perspective(1000px) rotateX(0deg) rotateY(0deg)');
  const [glarePos, setGlarePos] = useState({ x: 50, y: 50, opacity: 0 });

  const handleMouseMove = (e: ReactMouseEvent<HTMLDivElement>) => {
    const card = cardRef.current;
    if (!card) return;
    const rect = card.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;

    const rotateX = ((y - centerY) / centerY) * -6;
    const rotateY = ((x - centerX) / centerX) * 6;

    setTransform(`perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale3d(1.012, 1.012, 1.012)`);
    setGlarePos({
      x: (x / rect.width) * 100,
      y: (y / rect.height) * 100,
      opacity: 0.2,
    });
  };

  const handleMouseLeave = () => {
    setTransform('perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)');
    setGlarePos((prev) => ({ ...prev, opacity: 0 }));
  };

  return (
    <div
      ref={cardRef}
      data-cursor={dataCursor}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={`relative overflow-hidden rounded-2xl border border-white/10 bg-[#0D111A]/80 transition-transform duration-200 ease-out ${className}`}
      style={{ transform, transformStyle: 'preserve-3d' }}
    >
      <div
        className="pointer-events-none absolute inset-0 transition-opacity duration-300"
        style={{
          background: `radial-gradient(circle at ${glarePos.x}% ${glarePos.y}%, rgba(0, 245, 212, ${glarePos.opacity}), transparent 60%)`,
          opacity: glarePos.opacity,
        }}
      />
      {children}
    </div>
  );
}

// ==========================================
// REVEAL ANIMATION WRAPPER
// ==========================================
function Reveal({
  children,
  delay = 0,
  className = '',
}: {
  children: ReactNode;
  delay?: number;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement | null>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.15 }
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={ref}
      className={`transition-all duration-1000 ease-out ${visible ? 'translate-y-0 opacity-100' : 'translate-y-8 opacity-0'
        } ${className}`}
      style={{ transitionDelay: `${delay}ms` }}
    >
      {children}
    </div>
  );
}

// ==========================================
// NAVBAR COMPONENT
// ==========================================
function LuxuryNavbar({ onOpenVIP }: { onOpenVIP: () => void }) {
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const isAuthenticated = useAuth();

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 20);
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const navLinks = [
    { label: 'Platform Topics', href: '#topics' },
    { label: 'Quantum Threat Radar', href: '#threat-radar' },
    { label: 'CBOM Standard', href: '#cbom' },
  ];

  return (
    <header
      className={`fixed inset-x-0 top-0 z-40 transition-all duration-300 ${scrolled || menuOpen
        ? 'border-b border-white/10 bg-[#07090E]/90 shadow-[0_4px_30px_rgba(0,0,0,0.8)] backdrop-blur-xl'
        : 'border-b border-transparent bg-transparent'
        }`}
    >
      <div className="mx-auto flex h-20 max-w-7xl items-center justify-between px-6">
        {/* VYALA Logo — links to top of page */}
        <a href="#top" className="flex-shrink-0 transition-opacity hover:opacity-85" aria-label="VYALA Home">
          <VyalaFullLogo className="" />
        </a>

        {/* Center Links */}
        <nav className="hidden items-center gap-8 md:flex">
          {navLinks.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="text-xs font-medium tracking-wide text-slate-300 transition-colors hover:text-[#00F5D4]"
            >
              {link.label}
            </a>
          ))}
        </nav>

        {/* Right CTA */}
        <div className="flex items-center gap-4">
          <div className="hidden items-center gap-2 rounded-full border border-[#00F5D4]/30 bg-[#00F5D4]/10 px-3.5 py-1 text-[11px] font-mono text-[#00F5D4] lg:flex">
            <span className="h-1.5 w-1.5 rounded-full bg-[#00F5D4] animate-pulse" />
            NIST FIPS 203 READY
          </div>

          <a
            href="#wishlist"
            data-cursor="ticket"
            className="relative inline-flex items-center gap-2 overflow-hidden rounded-xl border border-[#00F5D4]/60 bg-gradient-to-r from-[#00F5D4] via-[#00C8FF] to-[#7B6EF6] px-4 py-2.5 text-xs font-bold tracking-wider text-black shadow-[0_0_20px_rgba(0,245,212,0.3)] transition-all hover:scale-105 active:scale-95"
          >
            <Crown className="h-3.5 w-3.5" />
            <span> EARLY ACCESS</span>
          </a>

          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="p-2 text-slate-300 md:hidden"
            aria-label="Toggle menu"
          >
            {menuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </button>
        </div>
      </div>

      {menuOpen && (
        <div className="border-b border-slate-800 bg-[#07090E]/95 px-6 py-6 backdrop-blur-xl md:hidden">
          <div className="flex flex-col gap-4">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                onClick={() => setMenuOpen(false)}
                className="text-sm font-medium text-slate-200 hover:text-[#00F5D4]"
              >
                {link.label}
              </a>
            ))}
          </div>
        </div>
      )}
    </header>
  );
}

// ==========================================
// INTERACTIVE SCANNER TERMINAL MOCKUP
// ==========================================
function InteractiveScannerMockup() {
  const [activeTab, setActiveTab] = useState<'code' | 'deps' | 'iac' | 'tls'>('code');
  const [isScanning, setIsScanning] = useState(false);
  const [scanProgress, setScanProgress] = useState(100);

  const handleRunScan = () => {
    setIsScanning(true);
    setScanProgress(0);
    let p = 0;
    const interval = setInterval(() => {
      p += 20;
      setScanProgress(p);
      if (p >= 100) {
        clearInterval(interval);
        setIsScanning(false);
      }
    }, 180);
  };

  const sampleResults = {
    code: [
      {
        type: 'VULNERABLE',
        badge: 'RSA-2048',
        file: 'src/auth/vault_signer.py:84',
        desc: 'Harvest-Now Decrypt-Later Risk: RSA signature found',
        suggested: 'Upgrade to ML-KEM-768 (FIPS 203)',
        severity: 'CRITICAL',
      },
      {
        type: 'MIGRATED',
        badge: 'ML-DSA-65',
        file: 'src/crypto/quantum_sig.ts:12',
        desc: 'Lattice-based digital signature validated',
        suggested: 'NIST FIPS 204 Compliant',
        severity: 'SAFE',
      },
    ],
    deps: [
      {
        type: 'VULNERABLE',
        badge: 'PyJWT v2.4',
        file: 'requirements.txt:14',
        desc: 'Third-party package relies on classical ECDSA P-256',
        suggested: 'Update to v2.9+ with Post-Quantum Hybrid extension',
        severity: 'HIGH',
      },
    ],
    iac: [
      {
        type: 'VULNERABLE',
        badge: 'TLS 1.2 Suite',
        file: 'infra/k8s/ingress.yaml:32',
        desc: 'Ingress controller permits classical ECDHE key exchange',
        suggested: 'Configure X25519_MLKEM768 in cipher policy',
        severity: 'MEDIUM',
      },
    ],
    tls: [
      {
        type: 'MIGRATED',
        badge: 'X25519+ML-KEM',
        file: 'api.vyala.dev:443',
        desc: 'Active TLS handshake verified quantum-safe hybrid key exchange',
        suggested: 'Passes NIST FIPS 203 verification',
        severity: 'SAFE',
      },
    ],
  };

  return (
    <TiltCard dataCursor="scan" className="p-1">
      <div className="flex items-center justify-between border-b border-slate-800 bg-[#0D111A] px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs text-slate-300">
            vyala-cli v2.4.0 — Quantum Engine
          </span>
        </div>
        <button
          onClick={handleRunScan}
          disabled={isScanning}
          className="flex items-center gap-1.5 rounded-lg border border-[#00F5D4]/40 bg-[#00F5D4]/10 px-3 py-1 font-mono text-xs font-semibold text-[#00F5D4] transition-all hover:bg-[#00F5D4]/20"
        >
          {isScanning ? (
            <>
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              <span>Scanning... {scanProgress}%</span>
            </>
          ) : (
            <>
              <Zap className="h-3.5 w-3.5 text-[#00F5D4]" />
              <span>Re-Run Audit</span>
            </>
          )}
        </button>
      </div>

      <div className="flex border-b border-slate-800 bg-[#07090E]/60 p-1 font-mono text-xs">
        {(['code', 'deps', 'iac', 'tls'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 rounded-lg py-2 transition-all ${activeTab === tab
              ? 'bg-[#141A26] font-bold text-[#00F5D4] shadow-inner'
              : 'text-slate-400 hover:text-slate-200'
              }`}
          >
            {tab.toUpperCase()} AUDIT
          </button>
        ))}
      </div>

      <div className="min-h-[260px] p-5 font-mono text-xs text-slate-300">
        {isScanning ? (
          <div className="flex h-48 flex-col items-center justify-center gap-3">
            <Loader2 className="h-8 w-8 text-[#00F5D4] animate-spin" />
            <p className="font-mono text-xs text-[#00F5D4]">
              Parsing AST & validating against NIST FIPS 203/204/205 specs...
            </p>
            <div className="h-1.5 w-64 overflow-hidden rounded-full bg-slate-800">
              <div
                className="h-full bg-gradient-to-r from-[#00F5D4] via-[#00C8FF] to-[#7B6EF6] transition-all duration-200"
                style={{ width: `${scanProgress}%` }}
              />
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="flex items-center justify-between text-[11px] text-slate-400">
              <span>Surface: {activeTab.toUpperCase()}</span>
              <span className="text-[#00F5D4]">Status: Scan Completed in 28ms</span>
            </div>

            {sampleResults[activeTab].map((item, idx) => (
              <div
                key={idx}
                className={`rounded-xl border p-3 transition-all ${item.severity === 'SAFE'
                  ? 'border-[#00F5D4]/30 bg-[#00F5D4]/5'
                  : 'border-[#FF4D4D]/30 bg-[#FF4D4D]/5'
                  }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={`rounded px-2 py-0.5 font-mono text-[10px] font-bold ${item.severity === 'SAFE'
                        ? 'bg-[#00F5D4]/20 text-[#00F5D4]'
                        : 'bg-[#FF4D4D]/20 text-[#FF4D4D]'
                        }`}
                    >
                      {item.badge}
                    </span>
                    <span className="font-mono text-slate-300">{item.file}</span>
                  </div>
                  <span className="font-mono text-[10px] text-slate-400">
                    {item.severity}
                  </span>
                </div>
                <p className="mt-2 text-slate-200">{item.desc}</p>
                <div className="mt-2 flex items-center gap-1.5 text-[11px] font-medium text-[#00F5D4]">
                  <ArrowRight className="h-3 w-3" />
                  <span>{item.suggested}</span>
                </div>
              </div>
            ))}

            <div className="mt-4 flex items-center justify-between rounded-lg border border-slate-800 bg-[#07090E] p-2.5 text-[11px] text-slate-400">
              <span className="flex items-center gap-1.5">
                <CheckCircle2 className="h-3.5 w-3.5 text-[#00F5D4]" />
                Auto-Posted PR Comment & CBOM Spec
              </span>
              <span className="text-[#00F5D4]">NIST FIPS 203/204 Validated</span>
            </div>
          </div>
        )}
      </div>
    </TiltCard>
  );
}

// ==========================================
// HERO SECTION
// ==========================================
function LuxuryHero() {
  return (
    <section id="top" className="relative pt-36 pb-24 lg:pt-44 lg:pb-32">
      <div className="relative mx-auto max-w-7xl px-6">
        <div className="grid gap-12 lg:grid-cols-12 lg:items-center">
          <div className="lg:col-span-6">
            <Reveal>
              <div className="inline-flex items-center gap-2 rounded-full border border-[#00F5D4]/30 bg-[#00F5D4]/10 px-4 py-1.5 font-mono text-xs font-semibold text-[#00F5D4] shadow-[0_0_20px_rgba(0,245,212,0.15)]">
                <Sparkles className="h-3.5 w-3.5 text-[#00F5D4]" />
                <span>NIST POST-QUANTUM CRYPTOGRAPHY GOVERNANCE</span>
              </div>
            </Reveal>

            <Reveal delay={1}>
              <h1 className="mt-6 font-display text-5xl font-extrabold leading-[1.2] tracking-tight text-white sm:text-6xl lg:text-7xl">
                Guarding the Future of <br />
                <span className="cyber-text-shimmer">Cryptography</span>
              </h1>
            </Reveal>

            <Reveal delay={200}>
              <p className="mt-6 max-w-xl text-lg leading-relaxed text-slate-300">
                Enterprise audits take 12 months and $100K+. <strong className="text-white">VYALA</strong> detects quantum-vulnerable cryptography (RSA, ECDSA) in seconds on every pull request, mapping NIST FIPS 203, 204, and 205 replacements directly inline.
              </p>
            </Reveal>

            <Reveal delay={300}>
              <div className="mt-8 flex flex-wrap items-center gap-4">
                <a
                  href="#wishlist"
                  data-cursor="ticket"
                  className="flex items-center gap-2 rounded-xl border border-[#00F5D4] bg-gradient-to-r from-[#00F5D4] via-[#00C8FF] to-[#7B6EF6] px-6 py-3.5 font-mono text-xs font-bold tracking-wider text-black shadow-[0_0_25px_rgba(0,245,212,0.35)] transition-all hover:scale-105 active:scale-95"
                >
                  <Crown className="h-4 w-4" />
                  <span>CLAIM EARLY ACCESS PASS</span>
                </a>

                <a
                  href="https://github.com/vyala/vyala"
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-2 rounded-xl border border-slate-700 bg-slate-900/80 px-6 py-3.5 font-mono text-xs font-medium text-slate-200 transition-colors hover:border-[#00F5D4]/50 hover:bg-slate-800"
                >
                  <GitBranch className="h-4 w-4 text-[#00F5D4]" />
                  <span>View on GitHub</span>
                </a>
              </div>
            </Reveal>

            <Reveal delay={400}>
              <div className="mt-12 grid grid-cols-3 gap-6 border-t border-slate-800/80 pt-6">
                <div>
                  <p className="font-display text-2xl font-bold text-[#00F5D4]">
                    &lt; 30s
                  </p>
                  <p className="mt-0.5 font-mono text-[11px] text-slate-400">
                    PR Scan Speed
                  </p>
                </div>
                <div>
                  <p className="font-display text-2xl font-bold text-[#00C8FF]">
                    FIPS 203/204
                  </p>
                  <p className="mt-0.5 font-mono text-[11px] text-slate-400">
                    NIST Aligned
                  </p>
                </div>
                <div>
                  <p className="font-display text-2xl font-bold text-white">
                    100% CBOM
                  </p>
                  <p className="mt-0.5 font-mono text-[11px] text-slate-400">
                    Automated Spec
                  </p>
                </div>
              </div>
            </Reveal>
          </div>

          <div className="lg:col-span-6">
            <Reveal delay={200}>
              <InteractiveScannerMockup />
            </Reveal>
          </div>
        </div>
      </div>
    </section>
  );
}

// ==========================================
// TOPIC 1: QUANTUM THREAT RADAR & CALCULATOR
// ==========================================
function QuantumThreatRadarSection() {
  const [keyCount, setKeyCount] = useState(450);
  const [algoTab, setAlgoTab] = useState<'kem' | 'dsa'>('kem');

  const estimatedHndlRisk = Math.min(Math.round((keyCount / 1000) * 100), 99);
  const migrationWeeks = Math.ceil(keyCount / 40);

  return (
    <section id="threat-radar" className="relative border-t border-slate-800/60 bg-[#0D111A]/40 py-24">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal>
          <div className="mx-auto max-w-3xl text-center">
            <span className="font-mono text-xs tracking-widest text-[#00F5D4] uppercase">
              TOPIC I: QUANTUM THREAT RADAR & ALGORITHM MIGRATION
            </span>
            <h2 className="mt-3 font-display text-4xl font-extrabold text-white sm:text-5xl">
              Harvest-Now-Decrypt-Later Threat Engine
            </h2>
            <p className="mt-4 text-slate-300">
              Quantum computers will break classical RSA and ECC. Adversaries are intercepting encrypted data today to decrypt tomorrow. Calculate your enterprise risk exposure below.
            </p>
          </div>
        </Reveal>

        <div className="mt-16 grid gap-8 lg:grid-cols-12">
          <div className="lg:col-span-6">
            <TiltCard className="h-full p-6 sm:p-8">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#00F5D4]/15 text-[#00F5D4]">
                  <Sliders className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-display text-lg font-bold text-white">
                    Cryptographic Key Exposure Calculator
                  </h3>
                  <p className="font-mono text-xs text-slate-400">
                    Adjust key count to model enterprise quantum exposure
                  </p>
                </div>
              </div>

              <div className="mt-8 space-y-6">
                <div>
                  <div className="flex justify-between font-mono text-xs text-slate-300">
                    <span>Active RSA / ECC Key Surfaces</span>
                    <span className="font-bold text-[#00F5D4]">{keyCount} Keys</span>
                  </div>
                  <input
                    type="range"
                    min={10}
                    max={2000}
                    step={10}
                    value={keyCount}
                    onChange={(e) => setKeyCount(Number(e.target.value))}
                    className="mt-3 h-2 w-full cursor-pointer appearance-none rounded-lg bg-slate-800 accent-[#00F5D4]"
                  />
                  <div className="mt-1 flex justify-between font-mono text-[10px] text-slate-400">
                    <span>10 (Startup)</span>
                    <span>1,000 (Scaleup)</span>
                    <span>2,000+ (Global Enterprise)</span>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-800 bg-[#07090E] p-4">
                  <div>
                    <span className="font-mono text-[11px] text-slate-400">
                      HNDL Vulnerability Score
                    </span>
                    <p className="mt-1 font-display text-2xl font-bold text-[#FF4D4D]">
                      {estimatedHndlRisk}% CRITICAL
                    </p>
                  </div>
                  <div>
                    <span className="font-mono text-[11px] text-slate-400">
                      Manual Migration Effort
                    </span>
                    <p className="mt-1 font-display text-2xl font-bold text-[#00C8FF]">
                      ~{migrationWeeks} Weeks
                    </p>
                  </div>
                </div>

                <div className="rounded-xl border border-[#00F5D4]/30 bg-[#00F5D4]/5 p-4">
                  <div className="flex items-center gap-2 text-xs font-bold text-[#00F5D4]">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>VYALA Automated Migration Advantage</span>
                  </div>
                  <p className="mt-1 font-mono text-xs text-slate-300">
                    Reduces migration time from {migrationWeeks} weeks down to under 2 days with automated AST rule matching and CI/CD PR injection.
                  </p>
                </div>
              </div>
            </TiltCard>
          </div>

          <div className="lg:col-span-6">
            <TiltCard className="h-full p-6 sm:p-8">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-display text-lg font-bold text-white">
                    NIST FIPS 203 / 204 Standard Mapping
                  </h3>
                  <p className="font-mono text-xs text-slate-400">
                    Classical ciphers vs Post-Quantum replacements
                  </p>
                </div>
                <div className="flex rounded-lg border border-slate-800 bg-[#07090E] p-1 font-mono text-xs">
                  <button
                    onClick={() => setAlgoTab('kem')}
                    className={`rounded px-3 py-1 ${algoTab === 'kem'
                      ? 'bg-[#00F5D4] font-bold text-black'
                      : 'text-slate-400'
                      }`}
                  >
                    Encryption (KEM)
                  </button>
                  <button
                    onClick={() => setAlgoTab('dsa')}
                    className={`rounded px-3 py-1 ${algoTab === 'dsa'
                      ? 'bg-[#00F5D4] font-bold text-black'
                      : 'text-slate-400'
                      }`}
                  >
                    Signatures (DSA)
                  </button>
                </div>
              </div>

              {algoTab === 'kem' ? (
                <div className="mt-6 space-y-4 font-mono text-xs">
                  <div className="rounded-xl border border-red-500/30 bg-red-950/20 p-4">
                    <span className="text-red-400 font-bold">VULNERABLE (CLASSICAL)</span>
                    <p className="mt-1 text-slate-200">RSA-2048 / RSA-4096 & ECDH P-256</p>
                    <p className="mt-1 text-[11px] text-slate-400">
                      Broken by Shor's Algorithm running on ~4,000 logical qubits.
                    </p>
                  </div>
                  <div className="rounded-xl border border-[#00F5D4]/40 bg-[#00F5D4]/10 p-4">
                    <span className="text-[#00F5D4] font-bold">NIST FIPS 203 REPLACEMENT</span>
                    <p className="mt-1 text-white font-bold">ML-KEM-768 / ML-KEM-1024 (Kyber)</p>
                    <p className="mt-1 text-[11px] text-slate-300">
                      Module-Lattice-Based Key-Encapsulation Mechanism resistant to both quantum and classical cryptanalysis.
                    </p>
                  </div>
                </div>
              ) : (
                <div className="mt-6 space-y-4 font-mono text-xs">
                  <div className="rounded-xl border border-red-500/30 bg-red-950/20 p-4">
                    <span className="text-red-400 font-bold">VULNERABLE (CLASSICAL)</span>
                    <p className="mt-1 text-slate-200">ECDSA P-256 / Ed25519 / RSA Signatures</p>
                    <p className="mt-1 text-[11px] text-slate-400">
                      Vulnerable to forgery under quantum discrete log attacks.
                    </p>
                  </div>
                  <div className="rounded-xl border border-[#00F5D4]/40 bg-[#00F5D4]/10 p-4">
                    <span className="text-[#00F5D4] font-bold">NIST FIPS 204 / 205 REPLACEMENT</span>
                    <p className="mt-1 text-white font-bold">ML-DSA-65 (Dilithium) & SLH-DSA (Sphincs+)</p>
                    <p className="mt-1 text-[11px] text-slate-300">
                      Lattice-based and hash-based digital signature schemes providing quantum authentication.
                    </p>
                  </div>
                </div>
              )}
            </TiltCard>
          </div>
        </div>
      </div>
    </section>
  );
}

// ==========================================
// TOPIC 2: FOUR ATTACK SURFACES
// ==========================================
function FourAttackSurfacesSection() {
  const surfaces = [
    {
      code: 'CODE',
      icon: Code2,
      title: 'AST Source Code Analysis',
      desc: 'Parses Python, JavaScript, TypeScript, and Go abstract syntax trees to locate hardcoded RSA keys, ECDSA signers, and legacy ciphers in application code.',
      metric: 'Zero false positives via semantic AST matching',
    },
    {
      code: 'DEPS',
      icon: Package,
      title: 'Third-Party Dependency Graph',
      desc: 'Deeply scans npm, PyPI, Cargo, and Go module trees to uncover hidden classical cryptographic primitives buried deep within nested libraries.',
      metric: 'Traces down 12+ dependency levels',
    },
    {
      code: 'IAC',
      icon: Boxes,
      title: 'Infrastructure & Cloud Configs',
      desc: 'Scans Terraform, Kubernetes ingress definitions, Dockerfiles, and AWS/GCP TLS policies for legacy cipher suites and non-quantum key exchanges.',
      metric: 'Terraform & Helm native scanning',
    },
    {
      code: 'TLS',
      icon: Radio,
      title: 'Live Endpoint TLS Probing',
      desc: 'Actively probes production endpoints to verify active X25519_MLKEM768 hybrid handshakes are functioning in real-world network traffic.',
      metric: 'Real-time TLS handshake verification',
    },
  ];

  return (
    <section id="topics" className="relative py-24">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal>
          <div className="mx-auto max-w-3xl text-center">
            <span className="font-mono text-xs tracking-widest text-[#00F5D4] uppercase">
              TOPIC II: COMPLETE COVERAGE
            </span>
            <h2 className="mt-3 font-display text-4xl font-extrabold text-white sm:text-5xl">
              Four Attack Surfaces. One Vault.
            </h2>
            <p className="mt-4 text-slate-300">
              Quantum vulnerabilities don't hide only in your application logic. VYALA protects your entire software supply chain in one unified engine.
            </p>
          </div>
        </Reveal>

        <div className="mt-16 grid gap-6 md:grid-cols-2">
          {surfaces.map((s, i) => (
            <Reveal key={s.code} delay={i * 120}>
              <TiltCard className="group h-full p-8 transition-colors hover:border-[#00F5D4]/40">
                <div className="flex items-center justify-between">
                  <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-white/10 bg-[#141A26] text-[#00F5D4] transition-all group-hover:scale-110 group-hover:border-[#00F5D4] group-hover:bg-[#00F5D4]/15">
                    <s.icon className="h-6 w-6" />
                  </div>
                  <span className="font-mono text-xs font-bold tracking-widest text-[#00F5D4]">
                    [{s.code}]
                  </span>
                </div>
                <h3 className="mt-6 font-display text-xl font-bold text-white group-hover:text-[#00F5D4]">
                  {s.title}
                </h3>
                <p className="mt-3 text-sm leading-relaxed text-slate-300">
                  {s.desc}
                </p>
                <div className="mt-6 flex items-center gap-2 border-t border-slate-800 pt-4 font-mono text-xs text-[#00F5D4]">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  <span>{s.metric}</span>
                </div>
              </TiltCard>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}

// ==========================================
// TOPIC 3: AUTOMATED CBOM GOVERNANCE
// ==========================================
function CBOMSection() {
  const [copied, setCopied] = useState(false);

  const sampleCBOMJson = `{
  "bomFormat": "CBOM",
  "specVersion": "1.0",
  "version": 1,
  "metadata": {
    "timestamp": "${new Date().toISOString()}",
    "scanner": "VYALA v2.4.0",
    "fipsAlignment": ["FIPS 203", "FIPS 204", "FIPS 205"]
  },
  "findings": [
    {
      "id": "VYALA-PQC-2026-0042",
      "surface": "CODE",
      "location": "src/auth/session.py:42",
      "classicalCipher": "RSA-2048",
      "quantumVulnerability": "Harvest-Now Decrypt-Later",
      "recommendedReplacement": {
        "nistStandard": "FIPS 203",
        "algorithm": "ML-KEM-768",
        "hybridConstruction": "X25519 + ML-KEM-768"
      }
    }
  ]
}`;

  const handleCopy = () => {
    navigator.clipboard.writeText(sampleCBOMJson);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section id="cbom" className="relative border-t border-slate-800/60 bg-[#0D111A]/60 py-24">
      <div className="mx-auto max-w-7xl px-6">
        <div className="grid gap-12 lg:grid-cols-12 lg:items-center">
          <div className="lg:col-span-6">
            <Reveal>
              <span className="font-mono text-xs tracking-widest text-[#00F5D4] uppercase">
                TOPIC III: CRYPTOGRAPHIC BILL OF MATERIALS
              </span>
              <h2 className="mt-3 font-display text-4xl font-extrabold text-white sm:text-5xl">
                Automated CBOM Spec Generation
              </h2>
              <p className="mt-4 text-lg leading-relaxed text-slate-300">
                Compliance frameworks and Executive Orders now require a structured Cryptographic Bill of Materials (CBOM). VYALA automatically generates standardized JSON CBOM artifacts on every build, giving enterprise CISOs complete auditability.
              </p>

              <div className="mt-8 space-y-4 font-mono text-xs">
                <div className="flex items-center gap-3 rounded-xl border border-slate-800 bg-[#07090E] p-4 text-slate-300">
                  <CheckCircle2 className="h-5 w-5 text-[#00F5D4]" />
                  <span>Stable Finding IDs prevent duplicate PR comments</span>
                </div>
                <div className="flex items-center gap-3 rounded-xl border border-slate-800 bg-[#07090E] p-4 text-slate-300">
                  <CheckCircle2 className="h-5 w-5 text-[#00F5D4]" />
                  <span>Direct mapping to NIST FIPS 203/204/205 standards</span>
                </div>
                <div className="flex items-center gap-3 rounded-xl border border-slate-800 bg-[#07090E] p-4 text-slate-300">
                  <CheckCircle2 className="h-5 w-5 text-[#00F5D4]" />
                  <span>CI-Native workflow execution under 30 seconds</span>
                </div>
              </div>
            </Reveal>
          </div>

          <div className="lg:col-span-6">
            <Reveal delay={200}>
              <TiltCard className="p-1">
                <div className="flex items-center justify-between border-b border-slate-800 bg-[#07090E] px-4 py-3 font-mono text-xs text-slate-400">
                  <span>report.cbom.json</span>
                  <button
                    onClick={handleCopy}
                    className="flex items-center gap-1.5 rounded-lg border border-[#00F5D4]/30 bg-[#00F5D4]/10 px-3 py-1 font-mono text-xs text-[#00F5D4] transition-colors hover:bg-[#00F5D4]/20"
                  >
                    {copied ? (
                      <>
                        <Check className="h-3.5 w-3.5 text-[#00F5D4]" />
                        <span>COPIED!</span>
                      </>
                    ) : (
                      <>
                        <Copy className="h-3.5 w-3.5 text-[#00F5D4]" />
                        <span>COPY JSON SPEC</span>
                      </>
                    )}
                  </button>
                </div>
                <pre className="max-h-[380px] overflow-x-auto p-5 font-mono text-xs leading-relaxed text-[#00F5D4]">
                  <code>{sampleCBOMJson}</code>
                </pre>
              </TiltCard>
            </Reveal>
          </div>
        </div>
      </div>
    </section>
  );
}

// ==========================================
// SIMPLE WELCOME MODAL
// ==========================================
function WelcomeModal({ onClose }: { onClose: () => void }) {
  const backdropRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [onClose]);

  const handleBackdropClick = (e: ReactMouseEvent<HTMLDivElement>) => {
    if (e.target === backdropRef.current) onClose();
  };

  return (
    <div
      ref={backdropRef}
      onClick={handleBackdropClick}
      className="fixed inset-0 z-[999] flex items-center justify-center bg-black/80 p-4 backdrop-blur-xl"
    >
      <div className="relative w-full max-w-md overflow-hidden rounded-3xl border border-[#00F5D4]/40 bg-gradient-to-b from-[#141A26] via-[#0D111A] to-[#07090E] p-10 shadow-[0_0_60px_rgba(0,245,212,0.2)] text-center">
        {/* Glow orbs */}
        <div className="pointer-events-none absolute -left-24 -top-24 h-48 w-48 rounded-full bg-[#00F5D4]/15 blur-3xl" />
        <div className="pointer-events-none absolute -right-24 -bottom-24 h-48 w-48 rounded-full bg-[#7B6EF6]/10 blur-3xl" />

        {/* Close button */}
        <button
          onClick={onClose}
          aria-label="Close"
          className="absolute top-4 right-4 z-10 rounded-full p-2 text-slate-400 transition-colors hover:bg-white/10 hover:text-white"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Logo icon */}
        <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center">
          <img src="/vyala-icon.svg" alt="VYALA" className="h-16 w-16 drop-shadow-[0_0_12px_rgba(0,245,212,0.6)]" />
        </div>

        {/* Eyebrow */}
        <span className="font-mono text-[11px] font-bold tracking-[0.25em] text-[#00F5D4] uppercase">
          Early Access
        </span>

        {/* Headline */}
        <h3 className="mt-2 font-display text-4xl font-extrabold text-white">
          Welcome to{' '}
          <span className="cyber-text-shimmer">VYALA</span>
        </h3>

        {/* Body */}
        <p className="mt-4 text-sm leading-relaxed text-slate-300">
          You're among the first to guard the future of cryptography.
          We'll be in touch as we roll out early access spots.
        </p>

        {/* Divider */}
        <div className="my-6 h-px w-full bg-gradient-to-r from-transparent via-[#00F5D4]/30 to-transparent" />

        {/* Close CTA */}
        <button
          onClick={onClose}
          className="w-full rounded-xl border border-[#00F5D4]/60 bg-gradient-to-r from-[#00F5D4] via-[#00C8FF] to-[#7B6EF6] py-3 font-mono text-xs font-bold tracking-wider text-black shadow-[0_0_20px_rgba(0,245,212,0.25)] transition-all hover:brightness-110 active:scale-95"
        >
          LET'S GO
        </button>
      </div>
    </div>
  );
}

// ==========================================
// WISHLIST & EARLY ACCESS SECTION
// ==========================================
function LuxuryWishlistSection({ onGrantVIP }: { onGrantVIP: (name: string, email: string, code: string) => void }) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('Developer');
  const [stack, setStack] = useState('Python');
  const [concern, setConcern] = useState('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!email || !concern) return;

    setStatus('loading');

    const { error } = await supabase.from('wishlist').insert({
      name: name || null,
      email,
      role,
      stack,
      system_need: concern,
    });

    if (error) {
      console.error('Supabase Insert Error:', error);
      setStatus('error');
      return;
    }

    setStatus('success');
    setName('');
    setEmail('');
    setRole('Developer');
    setStack('Python');
    setConcern('');

    const generatedPassCode = `VYALA-PASS-${Math.floor(1000 + Math.random() * 9000)}`;
    onGrantVIP(name || 'Enterprise VIP', email, generatedPassCode);
  };

  return (
    <section id="wishlist" className="relative border-t border-slate-800/60 py-24">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal>
          <div className="mx-auto max-w-3xl text-center">
            <span className="font-mono text-xs tracking-widest text-[#00F5D4] uppercase">
              TOPIC IV: EARLY ACCESS WAITLIST
            </span>
            <h2 className="mt-3 font-display text-4xl font-extrabold text-white sm:text-5xl">
              The Quantum Threat is a "When," Not an "If."
            </h2>
            <p className="mt-4 text-slate-300">
              Harvest Now, Decrypt Later attacks are happening today. Don't wait for the 2026 federal mandates to find out your codebase is vulnerable.
            </p>
            <p className="mt-4 text-xl font-semibold text-[#00C8FF]">
              Join the early access waitlist to secure your stack.
            </p>
          </div>
        </Reveal>

        <div className="mt-16 grid gap-12 lg:grid-cols-12 lg:items-start">
          <div className="lg:col-span-7">
            <TiltCard className="p-8 sm:p-10">
              <form onSubmit={handleSubmit} className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <input
                    type="text"
                    placeholder="Full Name"
                    value={name}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
                    className="w-full rounded-xl border border-slate-800 bg-[#07090E] px-4 py-3 text-sm text-slate-100 outline-none transition-colors focus:border-[#00F5D4]"
                  />
                  <input
                    type="email"
                    placeholder="Work Email *"
                    value={email}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setEmail(e.target.value)}
                    required
                    className="w-full rounded-xl border border-slate-800 bg-[#07090E] px-4 py-3 text-sm text-slate-100 outline-none transition-colors focus:border-[#00F5D4]"
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">I am a...</label>
                    <select
                      value={role}
                      onChange={(e: ChangeEvent<HTMLSelectElement>) => setRole(e.target.value)}
                      className="w-full rounded-xl border border-slate-800 bg-[#07090E] px-4 py-3 text-sm text-slate-100 outline-none transition-colors focus:border-[#00F5D4]"
                    >
                      <option value="Developer">Developer / Engineer</option>
                      <option value="Security Lead">Security Lead / CISO</option>
                      <option value="Founder">Founder / CTO</option>
                      <option value="DevOps">DevOps / SRE</option>
                      <option value="Product">Product Manager</option>
                      <option value="Other">Other</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Primary Stack</label>
                    <select
                      value={stack}
                      onChange={(e: ChangeEvent<HTMLSelectElement>) => setStack(e.target.value)}
                      className="w-full rounded-xl border border-slate-800 bg-[#07090E] px-4 py-3 text-sm text-slate-100 outline-none transition-colors focus:border-[#00F5D4]"
                    >
                      <option value="Python">Python</option>
                      <option value="JS/TS">JavaScript / TypeScript</option>
                      <option value="Go">Go</option>
                      <option value="Java">Java</option>
                      <option value="Rust">Rust</option>
                      <option value="C/C++">C/C++</option>
                      <option value="PHP">PHP</option>
                      <option value="Ruby">Ruby</option>
                      <option value="Swift">Swift</option>
                      <option value="Kotlin">Kotlin</option>
                      <option value="R">R</option>
                      <option value="Multi">Multi-language</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    What is your biggest immediate cryptography concern? *
                  </label>
                  <textarea
                    placeholder="e.g., We process payments and need to know if our TLS is quantum-safe."
                    value={concern}
                    onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setConcern(e.target.value)}
                    required
                    rows={3}
                    className="w-full rounded-xl border border-slate-800 bg-[#07090E] px-4 py-3 text-sm text-slate-100 outline-none transition-colors focus:border-[#00F5D4]"
                  />
                </div>

                <button
                  type="submit"
                  disabled={status === 'loading'}
                  className="w-full rounded-xl border border-[#00F5D4] bg-gradient-to-r from-[#00F5D4] via-[#00C8FF] to-[#7B6EF6] px-6 py-4 font-mono text-sm font-semibold tracking-wider text-black transition-colors hover:brightness-110 disabled:opacity-50"
                >
                  {status === 'loading' ? 'Securing your spot...' : 'Get Early Access'}
                </button>

                {status === 'success' && (
                  <p className="mt-4 text-emerald-400 font-medium text-center">
                    You're on the waitlist. We'll be in touch soon to help secure your codebase.
                  </p>
                )}
                {status === 'error' && (
                  <p className="mt-4 text-rose-400 font-medium text-center">
                    Something went wrong. Please try again.
                  </p>
                )}
              </form>
            </TiltCard>
          </div>

          <div className="lg:col-span-5 space-y-6">
            <Reveal delay={150}>
              <TiltCard className="p-8">
                <div className="flex items-center gap-3">
                  <Crown className="h-6 w-6 text-[#00F5D4]" />
                  <h3 className="font-display text-lg font-bold text-white">
                    Early Access Benefits
                  </h3>
                </div>
                <div className="mt-6 space-y-4 font-mono text-xs">
                  <div className="flex items-start gap-3">
                    <CheckCircle2 className="h-4 w-4 text-[#00F5D4] shrink-0 mt-0.5" />
                    <span className="text-slate-300">
                      <strong>Priority Alpha Access</strong>: Deploy VYALA CLI & GitHub Action before public launch.
                    </span>
                  </div>
                  <div className="flex items-start gap-3">
                    <CheckCircle2 className="h-4 w-4 text-[#00F5D4] shrink-0 mt-0.5" />
                    <span className="text-slate-300">
                      <strong>Free CBOM Cryptographic Inventory</strong>: Receive a full CBOM report for up to 10 repositories.
                    </span>
                  </div>
                  <div className="flex items-start gap-3">
                    <CheckCircle2 className="h-4 w-4 text-[#00F5D4] shrink-0 mt-0.5" />
                    <span className="text-slate-300">
                      <strong>Direct PQC Architect Advisory</strong>: 1-on-1 session to map your RSA/ECC migration path to NIST FIPS 203/204.
                    </span>
                  </div>
                </div>
              </TiltCard>
            </Reveal>

            <Reveal delay={250}>
              <div className="rounded-2xl border border-[#00F5D4]/30 bg-[#141A26]/80 p-6 text-center">
                <ShieldAlert className="mx-auto h-8 w-8 text-[#00F5D4]" />
                <p className="mt-3 font-display text-lg font-bold text-white">
                  Limited Early Access
                </p>
              </div>
            </Reveal>
          </div>
        </div>
      </div>
    </section>
  );
}

// ==========================================
// FOOTER
// ==========================================
function LuxuryFooter() {
  return (
    <footer className="border-t border-slate-800/80 bg-[#07090E] py-12">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-6 px-6 sm:flex-row">
        <div className="w-24" />

        <p className="font-mono text-xs text-slate-400">
          Mapped to NIST FIPS 203 (ML-KEM) · FIPS 204 (ML-DSA) · FIPS 205 (SLH-DSA)
        </p>

        <div className="flex items-center gap-6 font-mono text-xs text-slate-400">
          <a href="https://github.com/vyala/vyala" className="hover:text-[#00F5D4]">
            GitHub
          </a>
          <a href="#topics" className="hover:text-[#00F5D4]">
            Topics
          </a>
          <a href="#wishlist" className="hover:text-[#00F5D4]">
            Claim Early Access Pass
          </a>
        </div>
      </div>
    </footer>
  );
}

// ==========================================
// MAIN LANDING PAGE CONTAINER
// ==========================================
export default function LandingPage() {
  const [welcomeOpen, setWelcomeOpen] = useState(false);

  const handleGrantVIP = (_name: string, _email: string, _code: string) => {
    setWelcomeOpen(true);
  };

  const handleOpenQuickVIP = () => {
    setWelcomeOpen(true);
  };

  return (
    <div className="custom-cursor-active min-h-screen bg-[#07090E] font-sans text-slate-100 selection:bg-[#00F5D4] selection:text-black">
      <GlobalAestheticStyles />
      <LuxuryInteractiveCanvas />
      <CustomLuxuryCursor />

      <LuxuryNavbar onOpenVIP={handleOpenQuickVIP} />

      <main className="relative z-10">
        <LuxuryHero />
        <QuantumThreatRadarSection />
        <FourAttackSurfacesSection />
        <CBOMSection />
        <LuxuryWishlistSection onGrantVIP={handleGrantVIP} />
      </main>

      <LuxuryFooter />

      {welcomeOpen && (
        <WelcomeModal onClose={() => setWelcomeOpen(false)} />
      )}
    </div>
  );
}