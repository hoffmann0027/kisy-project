import { useState } from "react";
import { Icon } from "@shared/ui/icons";
import type { ReportTargetKind } from "@shared/api/types";
import { ReportDialog } from "./ReportDialog";

// "Пожаловаться" as a button, for the places where the thing to report is the
// screen itself: a post, a community, the person on the other side of a chat.
// Messages have their own entry in the bubble's menu.
//
// Until now the dialog could report all four kinds and the server accepted
// all four, but only messages had a way in — so a post could never collect
// the reports that hide it, and a person or a community could not be reported
// at all (Google Play's policy for user-generated content wants all of them).

interface Props {
  targetKind: Exclude<ReportTargetKind, "message">;
  targetId: string;
  /** Spoken name of the button, e.g. "Пожаловаться на запись". */
  label: string;
  /** Class of the button, so it sits like its neighbours in each header. */
  className?: string;
  size?: number;
}

export function ReportButton({ targetKind, targetId, label, className, size = 18 }: Props) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className={className} title={label} aria-label={label} onClick={() => setOpen(true)}>
        <Icon.Flag size={size} />
      </button>
      {open && <ReportDialog targetKind={targetKind} targetId={targetId} onClose={() => setOpen(false)} />}
    </>
  );
}
