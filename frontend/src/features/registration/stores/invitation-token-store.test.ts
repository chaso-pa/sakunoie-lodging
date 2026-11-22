import { describe, expect, it } from 'vitest';
import { parseInviteFragment } from '@/features/registration/stores/invitation-token-store';
import { parseTokenFragment } from '@/features/registration/stores/manage-token-store';

describe('parseInviteFragment', () => {
  const token = 'AbCdEfGhIjKlMnOpQrStUvWxYz_0123456789-abcde';

  it('reads a base64url invitation from #invite=', () => {
    expect(parseInviteFragment(`#invite=${token}`)).toBe(token);
  });

  it('rejects other fragments, short or malformed values', () => {
    expect(parseInviteFragment('')).toBeNull();
    expect(parseInviteFragment(`#token=${token}`)).toBeNull();
    expect(parseInviteFragment('#invite=short')).toBeNull();
    expect(parseInviteFragment('#invite=has spaces in it!!')).toBeNull();
    expect(parseInviteFragment(`#invite=${token}&token=${token}`)).toBeNull();
  });

  it('is never read as an edit token', () => {
    expect(parseTokenFragment(`#invite=${token}`)).toBeNull();
  });
});
