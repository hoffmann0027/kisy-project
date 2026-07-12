import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { IconButton } from "./IconButton";

interface Props {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function Modal({ open, title, onClose, children }: Props) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  // Portal to <body> so the fixed backdrop always covers the whole viewport
  // and is never contained/clipped by a panel with backdrop-filter or
  // overflow:hidden (e.g. modals opened from the chat list or conversation).
  return createPortal(
    <div className="ui-modal__backdrop" onMouseDown={onClose}>
      <div className="ui-modal" role="dialog" aria-modal="true" onMouseDown={(e) => e.stopPropagation()}>
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
