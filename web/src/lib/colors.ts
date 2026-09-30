// Design token colors following DESIGN.md spatial glass language
export const colors = {
  // Primary - Cyan
  primary: {
    50: '#f0f9fc',
    100: '#e0f2f9',
    200: '#b3e5f5',
    300: '#81d4f4',
    400: '#4fc3f7',
    500: '#00d9ff',
    600: '#00b8cc',
    700: '#0097a7',
    800: '#007a8a',
    900: '#004e6b',
  },

  // Secondary - Violet
  secondary: {
    50: '#faf5ff',
    100: '#f3e5ff',
    200: '#e1bee7',
    300: '#ce93d8',
    400: '#ba68c8',
    500: '#7c3aed',
    600: '#7028d8',
    700: '#6f00d2',
    800: '#5e00c3',
    900: '#4a00b0',
  },

  // Success - Emerald
  success: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#10b981',
    600: '#059669',
    700: '#047857',
    800: '#065f46',
    900: '#064e3b',
  },

  // Warning - Amber
  warning: {
    50: '#fefce8',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
  },

  // Error - Red
  error: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444',
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
  },

  // Neutral - Gray
  neutral: {
    0: '#ffffff',
    50: '#fafafa',
    100: '#f5f5f5',
    200: '#e5e5e5',
    300: '#d4d4d4',
    400: '#a3a3a3',
    500: '#737373',
    600: '#525252',
    700: '#404040',
    800: '#262626',
    900: '#171717',
    950: '#0a0a0a',
  },
}

// Semantic color mappings
export const semanticColors = {
  background: colors.neutral[950],
  surface: colors.neutral[900],
  surfaceHover: colors.neutral[800],
  border: colors.neutral[700],
  text: {
    primary: colors.neutral[50],
    secondary: colors.neutral[300],
    tertiary: colors.neutral[500],
    disabled: colors.neutral[600],
  },
  focus: colors.primary[500],
  interactive: colors.primary[500],
  success: colors.success[500],
  warning: colors.warning[500],
  error: colors.error[500],
}

// Glass effect colors (semi-transparent overlays)
export const glassColors = {
  primary: 'rgba(0, 217, 255, 0.1)',
  secondary: 'rgba(124, 58, 237, 0.1)',
  success: 'rgba(16, 185, 129, 0.1)',
  warning: 'rgba(245, 158, 11, 0.1)',
  error: 'rgba(239, 68, 68, 0.1)',
}
