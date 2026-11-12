import { describe, expect, it } from 'vitest';
import { firebaseSettingsFromEnv } from '@/lib/firebase';

describe('firebaseSettingsFromEnv', () => {
  it('uses the emulator with demo defaults when the emulator host is set', () => {
    expect(firebaseSettingsFromEnv({ authEmulatorHost: '127.0.0.1:9099' })).toEqual({
      config: { apiKey: 'demo-api-key', authDomain: 'demo-sakunoie.firebaseapp.com', projectId: 'demo-sakunoie' },
      emulatorUrl: 'http://127.0.0.1:9099'
    });
  });

  it('requires the full config without the emulator', () => {
    expect(() => firebaseSettingsFromEnv({})).toThrow(/not configured/);
    expect(firebaseSettingsFromEnv({ apiKey: 'k', authDomain: 'd', projectId: 'p' })).toEqual({
      config: { apiKey: 'k', authDomain: 'd', projectId: 'p' },
      emulatorUrl: null
    });
  });
});
