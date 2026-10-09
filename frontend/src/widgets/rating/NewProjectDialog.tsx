import { useState } from "react";
import { Button, Modal, toast } from "@shared/ui";
import { t } from "@shared/i18n";
import type { useRatingMutations } from "@entities/rating/queries";
import { useAuthStore } from "@shared/store/auth";

interface Props {
  m: ReturnType<typeof useRatingMutations>;
  open: boolean;
  onClose: () => void;
}

/** A new project: title, description, and the level from which it is seen (CEO). */
export function NewProjectDialog({ m, open, onClose }: Props) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [minLevel, setMinLevel] = useState(10);
  // Not above the creator's own clearance — they could not see it.
  const lowest = useAuthStore((s) => s.user?.roleLevel ?? 1);

  const submit = () => {
    const name = title.trim();
    if (!name) return;
    m.createProject.mutate(
      { title: name, minLevel, description: description.trim() || undefined },
      {
        onSuccess: () => {
          setTitle("");
          setDescription("");
          setMinLevel(10);
          onClose();
        },
        onError: () => toast.error(t("work.rating.createProjectFailed")),
      },
    );
  };

  return (
    <Modal open={open} title={t("work.rating.newProjectTitle")} onClose={onClose}>
      <div className="rating-form">
        <input className="ui-input" placeholder={t("work.rating.projectTitlePlaceholder")} autoFocus value={title} onChange={(e) => setTitle(e.target.value)} onKeyDown={(e) => e.key === "Enter" && submit()} />
        <textarea className="ui-input" placeholder={t("work.rating.descriptionPlaceholder")} rows={3} value={description} onChange={(e) => setDescription(e.target.value)} />
        <label className="rating-form__field">
          <span>{t("work.rating.accessLevelHint")}</span>
          <select className="ui-input" value={minLevel} onChange={(e) => setMinLevel(Number(e.target.value))}>
            {Array.from({ length: 10 }, (_, i) => i + 1)
              .filter((lvl) => lvl >= lowest)
              .map((lvl) => (
                <option key={lvl} value={lvl}>
                  {t("work.rating.levelOption", { level: lvl })}
                </option>
              ))}
          </select>
        </label>
        <div className="rating-form__actions">
          <Button variant="ghost" onClick={onClose}>
            {t("work.rating.cancel")}
          </Button>
          <Button variant="primary" onClick={submit} loading={m.createProject.isPending} disabled={!title.trim()}>
            {t("work.rating.create")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
