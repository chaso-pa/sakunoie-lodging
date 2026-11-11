import { type CSSVariablesResolver, createTheme, type MantineColorsTuple } from '@mantine/core';

/**
 * DESIGN.md §3 tokens. Two text tokens are darkened for WCAG AA (≥ 4.5:1 on canvas/surface/surfaceMuted):
 * - `warningText` #8A5712 (DESIGN `warning` #A86B17 is 4.00 on canvas)
 * - `terracottaText` #934B32 (DESIGN `terracotta` #B05D3F is 4.27 as text on canvas)
 * `src/config/theme.test.ts` re-checks every text/background pair.
 */
export const tokens = {
  canvas: '#F7F4EE',
  surface: '#FFFFFF',
  surfaceMuted: '#F0EDE6',
  terracotta: '#B05D3F',
  terracottaHover: '#934B32',
  terracottaText: '#934B32',
  forest: '#214A3B',
  forestDeep: '#17382D',
  ink: '#292D2A',
  muted: '#626B65',
  border: '#DDDCD5',
  success: '#34734E',
  warning: '#A86B17',
  warningText: '#8A5712',
  error: '#B53B36'
} as const;

const terracotta: MantineColorsTuple = [
  '#FBF1EC',
  '#F3DED4',
  '#E8BFAE',
  '#DC9E85',
  '#D08262',
  '#C2704F',
  '#B05D3F',
  '#934B32',
  '#763C28',
  '#5A2D1E'
];

const forest: MantineColorsTuple = [
  '#EAF3EF',
  '#D2E3DB',
  '#A6C7B8',
  '#78A994',
  '#518F77',
  '#356B57',
  '#214A3B',
  '#17382D',
  '#10291F',
  '#091A13'
];

export const theme = createTheme({
  colors: { terracotta, forest },
  primaryColor: 'terracotta',
  primaryShade: 6,
  black: tokens.ink,
  fontFamily: 'var(--font-inter), var(--font-noto-sans-jp), system-ui, sans-serif',
  headings: {
    fontFamily: 'var(--font-inter), var(--font-noto-sans-jp), system-ui, sans-serif',
    fontWeight: '700'
  },
  defaultRadius: 'md',
  radius: { md: '10px', lg: '14px' },
  components: {
    Button: { defaultProps: { size: 'md' } },
    TextInput: { defaultProps: { size: 'md' } },
    NumberInput: { defaultProps: { size: 'md' } },
    Textarea: { defaultProps: { size: 'md' } },
    NativeSelect: { defaultProps: { size: 'md' } },
    Card: { defaultProps: { radius: 'lg', withBorder: true } }
  },
  other: { tokens }
});

/** Page/surface/text variables; `--mantine-color-anchor` uses the AA-safe terracotta text shade. */
export const cssVariablesResolver: CSSVariablesResolver = () => ({
  variables: {
    '--sakunoie-canvas': tokens.canvas,
    '--sakunoie-surface': tokens.surface,
    '--sakunoie-surface-muted': tokens.surfaceMuted,
    '--sakunoie-border': tokens.border,
    '--sakunoie-forest': tokens.forest,
    '--sakunoie-success': tokens.success,
    '--sakunoie-warning-text': tokens.warningText,
    '--sakunoie-error': tokens.error
  },
  light: {
    '--mantine-color-body': tokens.canvas,
    '--mantine-color-text': tokens.ink,
    '--mantine-color-dimmed': tokens.muted,
    '--mantine-color-anchor': tokens.terracottaText,
    '--mantine-color-error': tokens.error,
    '--mantine-color-default-border': tokens.border
  },
  dark: {}
});
