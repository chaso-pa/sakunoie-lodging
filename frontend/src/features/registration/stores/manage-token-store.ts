/**
 * Edit-token session (plan Decision C): URL fragment → sessionStorage (this tab only) →
 * `Authorization: Bearer`. Never localStorage, never a query string.
 */
const STORAGE_KEY = 'sakunoie.manage.token';
const TOKEN_PATTERN = /^[A-Za-z0-9_-]{16,256}$/;

export const manageTokenStore = {
  get(): string | null {
    try {
      return window.sessionStorage.getItem(STORAGE_KEY);
    } catch {
      return null;
    }
  },
  set(token: string) {
    try {
      window.sessionStorage.setItem(STORAGE_KEY, token);
    } catch {
      // Storage disabled: the session simply will not survive a reload.
    }
  },
  clear() {
    try {
      window.sessionStorage.removeItem(STORAGE_KEY);
    } catch {
      // ignore
    }
  }
};

/** Extracts `#token=…` from a URL fragment, or null if absent or malformed. */
export const parseTokenFragment = (hash: string): string | null => {
  const m = /^#token=([^&]+)$/.exec(hash);
  if (!m) return null;
  let token: string;
  try {
    token = decodeURIComponent(m[1]);
  } catch {
    return null;
  }
  return TOKEN_PATTERN.test(token) ? token : null;
};

/**
 * Moves a token from the fragment into sessionStorage and strips the fragment from the
 * address bar and history. Any fragment is removed, even a malformed one.
 * Returns whether a token was stored.
 */
export const consumeTokenFragment = (): boolean => {
  const { hash, pathname, search } = window.location;
  if (!hash) return false;
  const token = parseTokenFragment(hash);
  if (token) manageTokenStore.set(token);
  window.history.replaceState(window.history.state, '', `${pathname}${search}`);
  return token !== null;
};
