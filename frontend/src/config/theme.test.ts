import { describe, expect, it } from 'vitest';
import { tokens } from '@/config/theme';

const luminance = (hex: string) => {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const v = Number.parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

const contrast = (a: string, b: string) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};

const backgrounds = [tokens.canvas, tokens.surface, tokens.surfaceMuted];

describe('theme tokens meet WCAG AA (4.5:1) for normal text', () => {
  for (const name of ['ink', 'muted', 'terracottaText', 'warningText', 'error', 'success', 'forest'] as const) {
    it(`${name} on canvas, surface and surfaceMuted`, () => {
      for (const bg of backgrounds) expect(contrast(tokens[name], bg)).toBeGreaterThanOrEqual(4.5);
    });
  }

  it('white on the primary button color (terracotta) and on forest', () => {
    expect(contrast('#FFFFFF', tokens.terracotta)).toBeGreaterThanOrEqual(4.5);
    expect(contrast('#FFFFFF', tokens.terracottaHover)).toBeGreaterThanOrEqual(4.5);
    expect(contrast('#FFFFFF', tokens.forest)).toBeGreaterThanOrEqual(4.5);
  });

  it('home bands: (translucent) white text on forest and forestDeep', () => {
    /** `rgb(255 255 255 / alpha)` composited over `bg`. */
    const whiteOver = (alpha: number, bg: string) => {
      const ch = (i: number) => Math.round(alpha * 255 + (1 - alpha) * Number.parseInt(bg.slice(i, i + 2), 16));
      return `#${[1, 3, 5].map((i) => ch(i).toString(16).padStart(2, '0')).join('')}`;
    };
    expect(contrast('#FFFFFF', tokens.forestDeep)).toBeGreaterThanOrEqual(4.5);
    // Booking band intro (0.82–0.84) and the nearby featured card (0.8 local name, 0.88 description).
    expect(contrast(whiteOver(0.82, tokens.forestDeep), tokens.forestDeep)).toBeGreaterThanOrEqual(4.5);
    expect(contrast(whiteOver(0.8, tokens.forest), tokens.forest)).toBeGreaterThanOrEqual(4.5);
  });

  it('documents why the DESIGN warning/terracotta shades are not used as text', () => {
    expect(contrast(tokens.warning, tokens.canvas)).toBeLessThan(4.5);
    expect(contrast(tokens.terracotta, tokens.canvas)).toBeLessThan(4.5);
  });
});
