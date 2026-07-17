import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { IconButton } from "./IconButton";
import { useVisualViewport } from "@shared/lib/useVisualViewport";

interface Props {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function Modal({ open, title, onClose, children }: Props) {
  const modalRef = useRef<HTMLDivElement>(null);
  const focusedRef = useRef<HTMLElement | null>(null);
  const keyboardInset = useVisualViewport();

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  // Remember whichever field inside the modal has focus, so the effect below
  // can chase it once the keyboard has finished resizing the viewport.
  useEffect(() => {
    const el = modalRef.current;
    if (!open || !el) return;
    const onFocusIn = (e: FocusEvent) => {
      const target = e.target as HTMLElement | null;
      focusedRef.current =
        target && target.matches("input, textarea, select, [contenteditable]") ? target : null;
    };
    const onFocusOut = () => {
      focusedRef.current = null;
    };
    el.addEventListener("focusin", onFocusIn);
    el.addEventListener("focusout", onFocusOut);
    return () => {
      el.removeEventListener("focusin", onFocusIn);
      el.removeEventListener("focusout", onFocusOut);
    };
  }, [open]);

  // The modal shrinks by --kb-inset as the keyboard opens; pull the focused
  // field back into what is left. Runs on inset changes only, so a field
  // focused with the keyboard already up is handled by the browser itself.
  useEffect(() => {
    if (!open || keyboardInset === 0) return;
    focusedRef.current?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [open, keyboardInset]);

  if (!open) return null;

  // Portal to <body> so the fixed backdrop always covers the whole viewport
  // and is never contained/clipped by a panel with backdrop-filter or
  // overflow:hidden (e.g. modals opened from the chat list or conversation).
  return createPortal(
    <div className="ui-modal__backdrop" onMouseDown={onClose}>
      <div
        ref={modalRef}
        className="ui-modal"
        role="dialog"
        aria-modal="true"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="ui-modal__header">
          <h2 className="ui-modal__title">{title}</h2>
          <IconButton label="Закрыть" onClick={onClose}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12" strokeLinecap="round" />
            </svg>
          </IconButton>
        </div>
        <div className="ui-modal__body">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
