'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Copies by selecting a temporary textarea. `anchor` is where it is attached: inside an open modal
 * it must stay within the modal's focus trap, or the trap pulls focus back and the selection is lost.
 */
const copyWithSelection = (text: string, anchor: HTMLElement): boolean => {
  const area = document.createElement('textarea');
  area.value = text;
  area.setAttribute('readonly', '');
  area.setAttribute('aria-hidden', 'true');
  area.style.position = 'fixed';
  area.style.top = '0';
  area.style.left = '0';
  area.style.opacity = '0';
  anchor.appendChild(area);
  try {
    area.focus();
    area.select();
    area.setSelectionRange(0, text.length);
    return document.execCommand('copy');
  } catch {
    return false;
  } finally {
    area.remove();
  }
};

/**
 * Clipboard copy that also works outside secure contexts. `navigator.clipboard` exists only on HTTPS
 * or localhost, so a phone opening the dev server over plain http on the LAN falls back to
 * `execCommand('copy')`. `copied` is true for two seconds after a successful copy.
 */
export const useCopyText = (timeout = 2000) => {
  const [copied, setCopied] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(timer.current), []);

  const done = useCallback(
    (ok: boolean) => {
      window.clearTimeout(timer.current);
      setCopied(ok);
      if (ok) timer.current = window.setTimeout(() => setCopied(false), timeout);
    },
    [timeout]
  );

  const copy = useCallback(
    (text: string, anchor: HTMLElement = document.body) => {
      if (window.isSecureContext && navigator.clipboard?.writeText) {
        navigator.clipboard.writeText(text).then(
          () => done(true),
          () => done(copyWithSelection(text, anchor))
        );
        return;
      }
      done(copyWithSelection(text, anchor));
    },
    [done]
  );

  return { copied, copy };
};
