import { describe, expect, it } from 'vitest';
import { paths } from '@/config/paths';

describe('paths', () => {
  it('builds locale-less public and registration hrefs', () => {
    expect(paths.public.home.getHref()).toBe('/');
    expect(paths.public.terms.getHref()).toBe('/terms');
    expect(paths.registration.wizard.getHref('saku-no-ie')).toBe('/stays/saku-no-ie/registration');
    expect(paths.registration.complete.getHref()).toBe('/registration/complete');
    expect(paths.registration.manage.getHref()).toBe('/registration/manage');
  });

  it('encodes dynamic segments', () => {
    expect(paths.registration.wizard.getHref('a/b')).toBe('/stays/a%2Fb/registration');
    expect(paths.admin.registration.getHref('x y')).toBe('/admin/guest-registrations/x%20y');
  });

  it('puts the edit token only in the fragment of the manage URL', () => {
    const url = new URL(paths.registration.manage.getAbsoluteHref('https://example.test', 'en', 'abc_DEF-123'));
    expect(url.pathname).toBe('/en/registration/manage');
    expect(url.search).toBe('');
    expect(url.hash).toBe('#token=abc_DEF-123');
  });

  it('has admin hrefs without a locale', () => {
    expect(paths.admin.root.getHref()).toBe('/admin');
    expect(paths.admin.login.getHref()).toBe('/admin/login');
    expect(paths.admin.registrations.getHref()).toBe('/admin/guest-registrations');
    expect(paths.admin.surveys.getHref()).toBe('/admin/surveys');
  });
});
