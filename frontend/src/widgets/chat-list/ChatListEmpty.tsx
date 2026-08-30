import { Button } from "@shared/ui";

interface Props {
  title: string;
  hint: string;
  /** Optional action shown under the hint (hidden when omitted). */
  actionLabel?: string;
  onAction?: () => void;
  /**
   * Final 3D render for the illustration. While the artwork is missing the
   * component falls back to the CSS composition below (handoff §2), which is
   * drawn inside the same 200×180 box.
   */
  art?: string;
}

/**
 * Empty state of the chat list: illustration + title + hint + action
 * (design_handoff_kisy_mobile §2). The illustration is a phone-only element —
 * on desktop CSS hides it and only the text block remains.
 */
export function ChatListEmpty({ title, hint, actionLabel, onAction, art }: Props) {
  return (
    <div className="chatlist__empty">
      <div className="empty-art" aria-hidden="true">
        {art ? (
          <img className="empty-art__img" src={art} alt="" />
        ) : (
          <>
            <span className="empty-art__orbit" />
            <span className="empty-art__ball empty-art__ball--lg" />
            <span className="empty-art__ball empty-art__ball--md" />
            <span className="empty-art__ball empty-art__ball--sm" />
            <span className="empty-art__shadow" />
          </>
        )}
      </div>
      <h2 className="chatlist__empty-title">{title}</h2>
      <p className="chatlist__empty-hint">{hint}</p>
      {actionLabel && onAction && (
        <Button className="chatlist__empty-cta" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
