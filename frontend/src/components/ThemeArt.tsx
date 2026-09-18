import type { ThemeName } from '../api/types';

function ForestArt() {
  return (
    <>
      <defs>
        <linearGradient id="forestSun" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#f5f9f1" stopOpacity="0.5" />
          <stop offset="1" stopColor="#f5f9f1" stopOpacity="0" />
        </linearGradient>
      </defs>
      <circle cx="316" cy="96" r="44" fill="#f5f9f1" opacity="0.5" />
      <circle cx="316" cy="96" r="70" fill="url(#forestSun)" />
      <path d="M288 128 q7 -7 14 0 M308 120 q7 -7 14 0" stroke="#597b58" strokeWidth="3" fill="none" strokeLinecap="round" opacity="0.6" />
      <ellipse cx="200" cy="830" rx="330" ry="150" fill="#a9c4a5" opacity="0.45" />
      <ellipse cx="60" cy="700" rx="150" ry="80" fill="#8fb08b" opacity="0.35" />
      {/* дальние ёлки */}
      <g opacity="0.45">
        <path d="M52 560 L84 470 L116 560 Z" fill="#7fa17d" />
        <path d="M60 528 L84 460 L108 528 Z" fill="#7fa17d" />
        <rect x="79" y="560" width="10" height="30" fill="#6b5a44" />
        <path d="M300 580 L332 490 L364 580 Z" fill="#7fa17d" />
        <path d="M308 548 L332 480 L356 548 Z" fill="#7fa17d" />
        <rect x="327" y="580" width="10" height="30" fill="#6b5a44" />
      </g>
      {/* ближние ёлки */}
      <g>
        <rect x="120" y="600" width="14" height="52" fill="#6b5a44" opacity="0.85" />
        <path d="M70 610 L127 440 L184 610 Z" fill="#597b58" opacity="0.9" />
        <path d="M82 566 L127 430 L172 566 Z" fill="#6f9271" />
        <path d="M94 522 L127 420 L160 522 Z" fill="#7fa17d" />
        <rect x="238" y="620" width="16" height="60" fill="#6b5a44" opacity="0.85" />
        <path d="M176 632 L246 430 L316 632 Z" fill="#597b58" opacity="0.9" />
        <path d="M190 584 L246 420 L302 584 Z" fill="#6f9271" />
        <path d="M204 536 L246 410 L288 536 Z" fill="#7fa17d" />
      </g>
      <ellipse cx="200" cy="90" rx="230" ry="70" fill="#f5f9f1" opacity="0.25" />
    </>
  );
}

function CoffeeArt() {
  return (
    <>
      {/* газетный лист */}
      <g transform="rotate(-8 110 620)" opacity="0.5">
        <rect x="30" y="540" width="160" height="190" rx="6" fill="#f7f1e5" />
        <rect x="44" y="556" width="132" height="22" rx="3" fill="#6b4226" opacity="0.75" />
        <rect x="44" y="586" width="60" height="120" rx="3" fill="#a97c50" opacity="0.4" />
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <rect key={i} x={112} y={586 + i * 18} width={64} height={7} rx={3.5} fill="#6b4226" opacity="0.45" />
        ))}
        <rect x="44" y="688" width={132} height={7} rx={3.5} fill="#6b4226" opacity="0.3" />
      </g>
      {/* чашка */}
      <g>
        <ellipse cx="280" cy="660" rx="96" ry="20" fill="#6b4226" opacity="0.25" />
        <path d="M196 520 h150 v70 a75 55 0 0 1 -150 0 Z" fill="#a97c50" opacity="0.85" />
        <path d="M196 520 h150 v14 h-150 Z" fill="#6b4226" opacity="0.5" />
        <ellipse cx="271" cy="520" rx="75" ry="14" fill="#4e3018" opacity="0.8" />
        <ellipse cx="271" cy="520" rx="58" ry="9" fill="#2f1c0d" opacity="0.9" />
        <path d="M346 535 q44 8 30 52 q-12 36 -52 30" fill="none" stroke="#a97c50" strokeWidth="16" opacity="0.8" strokeLinecap="round" />
        <ellipse cx="271" cy="672" rx="104" ry="16" fill="none" stroke="#6b4226" strokeWidth="8" opacity="0.35" />
        <path d="M238 480 q10 -26 -6 -44 M266 482 q10 -30 -4 -52 M294 480 q10 -26 -6 -44" fill="none" stroke="#a97c50" strokeWidth="7" opacity="0.55" strokeLinecap="round" />
      </g>
      {/* зёрна */}
      <g fill="#6b4226" opacity="0.5">
        <ellipse cx="80" cy="300" rx="20" ry="14" transform="rotate(-24 80 300)" />
        <ellipse cx="330" cy="250" rx="16" ry="11" transform="rotate(18 330 250)" />
        <ellipse cx="120" cy="120" rx="13" ry="9" transform="rotate(-12 120 120)" opacity="0.7" />
      </g>
      <g stroke="#f3e3cd" strokeWidth="3" opacity="0.6">
        <path d="M64 292 q16 8 32 16" fill="none" />
        <path d="M316 244 q14 -6 28 -10" fill="none" />
      </g>
      <g opacity="0.14">
        {[0, 1, 2, 3, 4, 5, 6].map((i) => (
          <line key={i} x1="0" y1={i * 120 + 20} x2="400" y2={i * 120} stroke="#6b4226" strokeWidth="1.5" />
        ))}
      </g>
    </>
  );
}

function LavenderArt() {
  const sprig = (x: number, y: number, s: number, tilt: number, key: string) => (
    <g key={key} transform={`translate(${x} ${y}) rotate(${tilt}) scale(${s})`} opacity="0.8">
      <path d="M0 0 C -6 -50 6 -110 0 -160" stroke="#5d7d5a" strokeWidth="7" fill="none" strokeLinecap="round" />
      <path d="M0 -40 C -22 -52 -34 -58 -46 -60 M0 -80 C 22 -92 34 -98 46 -100" stroke="#5d7d5a" strokeWidth="5" fill="none" strokeLinecap="round" />
      {[-150, -138, -126, -114, -102, -90].map((yy, i) => (
        <g key={i}>
          <ellipse cx={-13} cy={yy} rx="11" ry="16" fill={i % 2 ? '#8a5f9e' : '#a37bb8'} transform={`rotate(-24 -13 ${yy})`} />
          <ellipse cx={13} cy={yy - 6} rx="11" ry="16" fill={i % 2 ? '#a37bb8' : '#7c5291'} transform={`rotate(24 13 ${yy - 6})`} />
        </g>
      ))}
      <ellipse cx="0" cy={-168} rx="10" ry="15" fill="#8a5f9e" />
    </g>
  );
  return (
    <>
      <defs>
        <linearGradient id="lavSun" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#fff" stopOpacity="0.55" />
          <stop offset="1" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
      </defs>
      <circle cx="322" cy="104" r="42" fill="#fff" opacity="0.55" />
      <circle cx="322" cy="104" r="72" fill="url(#lavSun)" />
      <ellipse cx="80" cy="200" rx="86" ry="26" fill="#fff" opacity="0.35" />
      <ellipse cx="270" cy="260" rx="104" ry="30" fill="#fff" opacity="0.28" />
      <ellipse cx="200" cy="840" rx="330" ry="150" fill="#b48ec4" opacity="0.4" />
      <ellipse cx="40" cy="720" rx="150" ry="80" fill="#c9a8d6" opacity="0.35" />
      {sprig(90, 700, 1.05, -6, 'a')}
      {sprig(200, 730, 1.3, 4, 'b')}
      {sprig(310, 700, 1.0, 10, 'c')}
      {sprig(30, 640, 0.8, -14, 'd')}
      {sprig(365, 640, 0.85, 14, 'e')}
      <g fill="#8a5f9e" opacity="0.5">
        <ellipse cx="150" cy="330" rx="9" ry="13" transform="rotate(24 150 330)" />
        <ellipse cx="250" cy="380" rx="8" ry="12" transform="rotate(-18 250 380)" />
        <ellipse cx="110" cy="450" rx="7" ry="10" transform="rotate(30 110 450)" />
      </g>
    </>
  );
}

const ART: Record<Exclude<ThemeName, 'custom'>, () => React.ReactNode> = {
  forest: ForestArt,
  coffee: CoffeeArt,
  lavender: LavenderArt,
};

export default function ThemeArt({ theme, artOn }: { theme: ThemeName; artOn: boolean }) {
  if (!artOn || theme === 'custom') return null;
  const Scene = ART[theme];
  return (
    <svg
      className="bg-illustration"
      viewBox="0 0 400 800"
      preserveAspectRatio="xMidYMid slice"
      aria-hidden="true"
      focusable="false"
    >
      <Scene />
    </svg>
  );
}
