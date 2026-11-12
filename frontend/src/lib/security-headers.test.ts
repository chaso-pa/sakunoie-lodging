import { describe, expect, it } from 'vitest';
import { buildCsp, buildPageHeaders, createNonce, isSensitivePath } from '@/lib/security-headers';

const directive = (csp: string, name: string) =>
  csp
    .split('; ')
    .find((d) => d.startsWith(`${name} `))
    ?.slice(name.length + 1);

describe('buildCsp', () => {
  const publicCsp = buildCsp({ nonce: 'abc123==', isDev: false, isAdmin: false });

  it('locks down the public pages', () => {
    expect(directive(publicCsp, 'script-src')).toBe("'self' 'nonce-abc123==' 'strict-dynamic'");
    expect(directive(publicCsp, 'connect-src')).toBe("'self'");
    expect(directive(publicCsp, 'frame-ancestors')).toBe("'none'");
    expect(directive(publicCsp, 'object-src')).toBe("'none'");
    expect(directive(publicCsp, 'base-uri')).toBe("'none'");
    expect(publicCsp).not.toMatch(/googleapis|unsafe-eval/);
  });

  it("adds 'unsafe-eval' only in development", () => {
    expect(directive(buildCsp({ nonce: 'n', isDev: true, isAdmin: false }), 'script-src')).toContain("'unsafe-eval'");
  });

  it('adds Firebase origins to connect-src only for admin', () => {
    expect(directive(buildCsp({ nonce: 'n', isDev: false, isAdmin: true }), 'connect-src')).toBe(
      "'self' https://identitytoolkit.googleapis.com https://securetoken.googleapis.com"
    );
    expect(
      directive(
        buildCsp({ nonce: 'n', isDev: false, isAdmin: true, firebaseAuthEmulatorHost: '127.0.0.1:9099' }),
        'connect-src'
      )
    ).toBe("'self' http://127.0.0.1:9099");
    expect(
      directive(
        buildCsp({ nonce: 'n', isDev: false, isAdmin: false, firebaseAuthEmulatorHost: '127.0.0.1:9099' }),
        'connect-src'
      )
    ).toBe("'self'");
  });
});

describe('buildCsp with a cross-origin API (NEXT_PUBLIC_API_BASE_URL)', () => {
  const apiOrigin = 'https://api.sakunoie.example';

  it("adds the API origin to connect-src on public, registration, manage and survey pages, keeping 'self'", () => {
    const csp = buildCsp({ nonce: 'n', isDev: false, isAdmin: false, apiOrigin });
    expect(directive(csp, 'connect-src')).toBe("'self' https://api.sakunoie.example");
    expect(csp).not.toMatch(/googleapis/);
  });

  it('adds the API origin and the Firebase origins on admin pages', () => {
    expect(directive(buildCsp({ nonce: 'n', isDev: false, isAdmin: true, apiOrigin }), 'connect-src')).toBe(
      "'self' https://api.sakunoie.example https://identitytoolkit.googleapis.com https://securetoken.googleapis.com"
    );
    expect(
      directive(
        buildCsp({ nonce: 'n', isDev: false, isAdmin: true, apiOrigin, firebaseAuthEmulatorHost: '127.0.0.1:9099' }),
        'connect-src'
      )
    ).toBe("'self' https://api.sakunoie.example http://127.0.0.1:9099");
  });
});

describe('buildPageHeaders', () => {
  it('sends no referrer and noindex on registration, stays and admin pages', () => {
    for (const p of [
      '/ja/registration/manage',
      '/en/registration/complete',
      '/ko/stays/saku-no-ie/registration',
      '/zh-CN/registration/manage',
      '/admin',
      '/admin/login'
    ]) {
      expect(isSensitivePath(p), p).toBe(true);
      expect(buildPageHeaders(p)['Referrer-Policy']).toBe('no-referrer');
      expect(buildPageHeaders(p)['X-Robots-Tag']).toBe('noindex, nofollow');
    }
  });

  it('uses strict-origin-when-cross-origin elsewhere', () => {
    for (const p of ['/', '/ja', '/en/faq', '/administrator']) {
      expect(buildPageHeaders(p)['Referrer-Policy'], p).toBe('strict-origin-when-cross-origin');
      expect(buildPageHeaders(p)['X-Robots-Tag']).toBeUndefined();
    }
    expect(buildPageHeaders('/ja')['X-Content-Type-Options']).toBe('nosniff');
  });
});

describe('createNonce', () => {
  it('is unique and matches the nonce pattern Next accepts', () => {
    const a = createNonce();
    expect(a).toMatch(/^[A-Za-z0-9+/_-]+={0,2}$/);
    expect(createNonce()).not.toBe(a);
  });
});
