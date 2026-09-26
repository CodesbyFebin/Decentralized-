import React from 'react';

/** Hexagonal isometric cube mark: cyan → blue → violet faces with a bright inner core. */
export const BrandMark: React.FC<{ size?: number; className?: string }> = ({ size = 40, className = '' }) => (
  <svg width={size} height={size} viewBox="0 0 64 64" className={className} aria-hidden="true">
    <defs>
      <linearGradient id="bm-top" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stopColor="#7DF3FF" />
        <stop offset="1" stopColor="#20DDF7" />
      </linearGradient>
      <linearGradient id="bm-left" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#248BFF" />
        <stop offset="1" stopColor="#3B2BD9" />
      </linearGradient>
      <linearGradient id="bm-right" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#A855F7" />
        <stop offset="1" stopColor="#6D28D9" />
      </linearGradient>
      <filter id="bm-glow" x="-30%" y="-30%" width="160%" height="160%">
        <feGaussianBlur stdDeviation="2.2" result="b" />
        <feMerge>
          <feMergeNode in="b" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>
    <g filter="url(#bm-glow)">
      <path d="M32 4 56 18 32 32 8 18Z" fill="url(#bm-top)" />
      <path d="M8 18 32 32V60L8 46Z" fill="url(#bm-left)" />
      <path d="M56 18 32 32V60L56 46Z" fill="url(#bm-right)" />
      <path d="M32 18 44 25 32 32 20 25Z" fill="#0B1734" opacity="0.85" />
      <path d="M20 25 32 32V45L20 38Z" fill="#0B1734" opacity="0.55" />
      <path d="M44 25 32 32V45L44 38Z" fill="#0B1734" opacity="0.7" />
      <path d="M32 4 56 18V46L32 60 8 46V18Z" fill="none" stroke="#E0FBFF" strokeOpacity="0.7" strokeWidth="1.2" />
    </g>
  </svg>
);

/**
 * Glowing isometric "server cube" on a holographic grid, used in the sidebar hero card.
 */
export const CubeIllustration: React.FC<{ className?: string }> = ({ className = '' }) => (
  <svg viewBox="0 0 200 150" className={className} aria-hidden="true">
    <defs>
      <radialGradient id="ci-floor" cx="0.5" cy="0.5" r="0.5">
        <stop offset="0" stopColor="#248BFF" stopOpacity="0.55" />
        <stop offset="1" stopColor="#248BFF" stopOpacity="0" />
      </radialGradient>
      <linearGradient id="ci-top" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stopColor="#5FEAFF" stopOpacity="0.95" />
        <stop offset="1" stopColor="#248BFF" stopOpacity="0.85" />
      </linearGradient>
      <linearGradient id="ci-left" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#1E3A8A" />
        <stop offset="1" stopColor="#0B1340" />
      </linearGradient>
      <linearGradient id="ci-right" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#4C1D95" />
        <stop offset="1" stopColor="#170B40" />
      </linearGradient>
      <filter id="ci-glow" x="-50%" y="-50%" width="200%" height="200%">
        <feGaussianBlur stdDeviation="3" result="b" />
        <feMerge>
          <feMergeNode in="b" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>
    <ellipse cx="100" cy="118" rx="90" ry="26" fill="url(#ci-floor)" />
    {/* holographic floor grid */}
    <g stroke="#60A5FA" strokeOpacity="0.35" strokeWidth="0.6">
      {[-60, -40, -20, 0, 20, 40, 60].map((d) => (
        <line key={`a${d}`} x1={100 + d - 40} y1={138} x2={100 + d + 40} y2={98} />
      ))}
      {[-60, -40, -20, 0, 20, 40, 60].map((d) => (
        <line key={`b${d}`} x1={100 + d + 40} y1={138} x2={100 + d - 40} y2={98} />
      ))}
    </g>
    <g filter="url(#ci-glow)">
      <path d="M100 22 146 46 100 70 54 46Z" fill="url(#ci-top)" />
      <path d="M54 46 100 70V118L54 94Z" fill="url(#ci-left)" />
      <path d="M146 46 100 70V118L146 94Z" fill="url(#ci-right)" />
      {/* inner cells */}
      <g stroke="#7DD3FC" strokeOpacity="0.55" strokeWidth="0.8" fill="none">
        <path d="M77 34 123 58M123 34 77 58M100 22V70" />
        <path d="M54 62 100 86M54 78 100 102M77 58V106" />
        <path d="M146 62 100 86M146 78 100 102M123 58V106" />
      </g>
      <path d="M100 22 146 46V94L100 118 54 94V46Z" fill="none" stroke="#A5F3FC" strokeWidth="1.4" />
      <circle cx="100" cy="70" r="3" fill="#E0FBFF" />
    </g>
    {/* floating nodes */}
    {[
      [30, 40, '#20DDF7'],
      [172, 34, '#A855F7'],
      [18, 92, '#248BFF'],
      [182, 88, '#20DDF7']
    ].map(([x, y, c], i) => (
      <g key={i} filter="url(#ci-glow)">
        <circle cx={x as number} cy={y as number} r="2.4" fill={c as string} />
        <line x1={x as number} y1={y as number} x2="100" y2="70" stroke={c as string} strokeOpacity="0.25" strokeWidth="0.6" />
      </g>
    ))}
  </svg>
);
