import { afterEach, describe, expect, it, vi } from 'vitest';
import { randomUUID } from './uuid';

const V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe('randomUUID', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('uses crypto.randomUUID when available', () => {
    expect(randomUUID()).toMatch(V4);
  });

  it('falls back to getRandomValues outside a secure context', () => {
    const real = globalThis.crypto;
    const getRandomValues = vi.fn(<T extends ArrayBufferView>(array: T) => real.getRandomValues(array));
    vi.stubGlobal('crypto', { getRandomValues });
    const a = randomUUID();
    const b = randomUUID();
    expect(a).toMatch(V4);
    expect(b).toMatch(V4);
    expect(a).not.toBe(b);
    expect(getRandomValues).toHaveBeenCalledTimes(2);
  });
});
