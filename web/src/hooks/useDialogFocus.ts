import { useEffect, useRef } from "react";
import type { KeyboardEvent, RefObject } from "react";

const focusableSelector = [
  "button:not(:disabled)",
  "a[href]",
  "input:not(:disabled)",
  "select:not(:disabled)",
  "textarea:not(:disabled)",
  "summary",
  '[tabindex]:not([tabindex="-1"])'
].join(",");

type DialogFocus = {
  panelRef: RefObject<HTMLElement | null>;
  initialFocusRef: RefObject<HTMLButtonElement | null>;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
};

export function useDialogFocus(open: boolean, onClose: () => void): DialogFocus {
  const panelRef = useRef<HTMLElement>(null);
  const initialFocusRef = useRef<HTMLButtonElement>(null);
  const returnFocus = useRef<HTMLElement | null>(null);
  const wasOpen = useRef(false);

  useEffect(() => {
    if (open && !wasOpen.current) {
      returnFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      initialFocusRef.current?.focus();
    } else if (!open && wasOpen.current) {
      returnFocus.current?.focus();
    }
    wasOpen.current = open;
  }, [open]);

  function onKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key === "Escape") {
      event.stopPropagation();
      onClose();
      return;
    }
    if (event.key !== "Tab" || !panelRef.current) return;
    const focusable = Array.from(panelRef.current.querySelectorAll<HTMLElement>(focusableSelector))
      .filter(isKeyboardVisible);
    if (focusable.length === 0) return;

    const first = focusable[0];
    const last = focusable.at(-1)!;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return { panelRef, initialFocusRef, onKeyDown };
}

function isKeyboardVisible(element: HTMLElement): boolean {
  if (element.closest('[inert], [aria-hidden="true"]')) return false;
  const closedDetails = element.closest("details:not([open])");
  if (closedDetails && element.tagName !== "SUMMARY") return false;
  const style = window.getComputedStyle(element);
  return style.display !== "none" && style.visibility !== "hidden";
}
