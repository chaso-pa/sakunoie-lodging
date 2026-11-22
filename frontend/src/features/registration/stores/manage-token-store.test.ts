import { describe, expect, it } from 'vitest';
import { parseTokenFragment } from '@/features/registration/stores/manage-token-store';

describe('parseTokenFragment', () => {
  it('reads a base64url token from #token=', () => {
    const token = 'AbCdEfGhIjKlMnOpQrStUvWxYz_0123456789-abcde';
    expect(parseTokenFragment(`#token=${token}`)).toBe(token);
  });

  it('rejects missing, short or malformed tokens', () => {
    expect(parseTokenFragment('')).toBeNull();
    expect(parseTokenFragment('#other=abc')).toBeNull();
    expect(parseTokenFragment('#token=short')).toBeNull();
    expect(parseTokenFragment('#token=has spaces in it!!')).toBeNull();
    expect(parseTokenFragment('#token=%E0%A4%A')).toBeNull();
  });
});
