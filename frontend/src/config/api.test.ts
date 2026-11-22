import { describe, expect, it } from 'vitest';
import { parseApiBaseUrl } from '@/config/api';

describe('parseApiBaseUrl', () => {
  it('is undefined when unset (relative /api/v1 through the dev rewrite)', () => {
    expect(parseApiBaseUrl(undefined)).toBeUndefined();
    expect(parseApiBaseUrl('  ')).toBeUndefined();
  });

  it('returns the origin', () => {
    expect(parseApiBaseUrl('https://api.sakunoie.example')).toBe('https://api.sakunoie.example');
    expect(parseApiBaseUrl('https://api.sakunoie.example/')).toBe('https://api.sakunoie.example');
    expect(parseApiBaseUrl('http://127.0.0.1:8180')).toBe('http://127.0.0.1:8180');
    expect(parseApiBaseUrl('HTTPS://API.Sakunoie.example:443')).toBe('https://api.sakunoie.example');
  });

  it('rejects anything but an http(s) origin', () => {
    for (const bad of [
      'api.sakunoie.example',
      '/api',
      'ftp://api.sakunoie.example',
      'https://api.sakunoie.example/api/v1',
      'https://api.sakunoie.example?x=1',
      'https://api.sakunoie.example#x',
      'https://user:pw@api.sakunoie.example'
    ]) {
      expect(() => parseApiBaseUrl(bad), bad).toThrow(/NEXT_PUBLIC_API_BASE_URL/);
    }
  });
});
