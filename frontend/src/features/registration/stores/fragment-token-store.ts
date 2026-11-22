/**
 * A secret handed over in a URL fragment (plan Decision C): fragment → sessionStorage (this tab
 * only) → `Authorization: Bearer`. Never localStorage, never a query string. Shared by the edit
 * token (`#token=`) and the registration invitation (`#invite=`).
 */
const TOKEN_PATTERN = /^[A-Za-z0-9_-]{16,256}$/;

export type FragmentTokenStore = {
  get: () => string | null;
  set: (token: string) => void;
  clear: () => void;
};

export const createFragmentTokenStore = (storageKey: string, fragmentName: string) => {
  const store: FragmentTokenStore = {
    get() {
      try {
        return window.sessionStorage.getItem(storageKey);
      } catch {
        return null;
      }
    },
    set(token) {
      try {
        window.sessionStorage.setItem(storageKey, token);
      } catch {
        // Storage disabled: the session simply will not survive a reload.
      }
    },
    clear() {
      try {
        window.sessionStorage.removeItem(storageKey);
      } catch {
        // ignore
      }
    }
  };

  const pattern = new RegExp(`^#${fragmentName}=([^&]+)$`);

  /** Extracts `#<name>=…` from a URL fragment, or null if absent or malformed. */
  const parse = (hash: string): string | null => {
    const m = pattern.exec(hash);
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
  const consume = (): boolean => {
    const { hash, pathname, search } = window.location;
    if (!hash) return false;
    const token = parse(hash);
    if (token) store.set(token);
    window.history.replaceState(window.history.state, '', `${pathname}${search}`);
    return token !== null;
  };

  return { store, parse, consume };
};
