// Typography system following DESIGN.md specification
export const typography = {
  // Headings
  h1: {
    fontSize: '36px',
    lineHeight: '40px',
    fontWeight: 700,
    letterSpacing: '-0.5px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  h2: {
    fontSize: '30px',
    lineHeight: '36px',
    fontWeight: 700,
    letterSpacing: '-0.25px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  h3: {
    fontSize: '24px',
    lineHeight: '32px',
    fontWeight: 600,
    letterSpacing: '0px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  h4: {
    fontSize: '20px',
    lineHeight: '28px',
    fontWeight: 600,
    letterSpacing: '0px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },

  // Body text
  bodyLarge: {
    fontSize: '18px',
    lineHeight: '28px',
    fontWeight: 400,
    letterSpacing: '0px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  body: {
    fontSize: '16px',
    lineHeight: '24px',
    fontWeight: 400,
    letterSpacing: '0px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  bodySmall: {
    fontSize: '14px',
    lineHeight: '20px',
    fontWeight: 400,
    letterSpacing: '0px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },

  // Labels and captions
  label: {
    fontSize: '14px',
    lineHeight: '20px',
    fontWeight: 500,
    letterSpacing: '0.5px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  caption: {
    fontSize: '12px',
    lineHeight: '16px',
    fontWeight: 500,
    letterSpacing: '0.25px',
    fontFamily: 'Inter, system-ui, sans-serif',
  },

  // Monospace (for code, addresses, hashes)
  mono: {
    fontSize: '14px',
    lineHeight: '20px',
    fontWeight: 400,
    fontFamily: 'Monaco, monospace',
  },
  monoSmall: {
    fontSize: '12px',
    lineHeight: '16px',
    fontWeight: 400,
    fontFamily: 'Monaco, monospace',
  },
}

// CSS class helpers
export const typographyClasses = {
  h1: 'text-4xl font-bold leading-tight tracking-tight',
  h2: 'text-3xl font-bold leading-9 tracking-tight',
  h3: 'text-2xl font-semibold leading-8',
  h4: 'text-xl font-semibold leading-7',
  bodyLarge: 'text-lg leading-7',
  body: 'text-base leading-6',
  bodySmall: 'text-sm leading-5',
  label: 'text-sm font-medium tracking-wider',
  caption: 'text-xs font-medium tracking-wider',
  mono: 'font-mono text-sm leading-5',
  monoSmall: 'font-mono text-xs leading-4',
}
