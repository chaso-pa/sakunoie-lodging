import type { Page } from '@playwright/test';

/**
 * Makes the current document behave like plain http on the LAN (no secure context, so no
 * `navigator.clipboard`) and records what `execCommand('copy')` would copy. E2E runs on 127.0.0.1,
 * which browsers treat as secure, so the fallback is otherwise never exercised.
 */
export const simulateInsecureClipboard = (page: Page) =>
  page.evaluate(() => {
    Object.defineProperty(window, 'isSecureContext', { value: false, configurable: true });
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
    const w = window as unknown as { __copied: string[] };
    w.__copied = [];
    document.execCommand = (command: string) => {
      const el = document.activeElement;
      if (command !== 'copy' || !(el instanceof HTMLTextAreaElement)) return false;
      w.__copied.push(el.value.slice(el.selectionStart, el.selectionEnd));
      return true;
    };
  });

export const copiedTexts = (page: Page) => page.evaluate(() => (window as unknown as { __copied: string[] }).__copied);
