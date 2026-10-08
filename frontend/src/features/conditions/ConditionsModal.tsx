import { useEffect, useState } from "react";
import "./conditions.css";
import { Button, Modal, Spinner, toast } from "@shared/ui";
import type { LevelCondition } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useAllConditions, useNextCondition, useSetCondition } from "@entities/condition/queries";
import { t } from "@shared/i18n";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ConditionsModal({ open, onClose }: Props) {
  const me = useAuthStore((s) => s.user!);
  const isCEO = me.roleLevel === 1;

  return (
    <Modal open={open} title={t("account.conditions.title")} onClose={onClose}>
      {isCEO ? (
        <CeoEditor open={open} />
      ) : me.roleLevel === null ? (
        // Promotion is a movement inside the hierarchy; this account is not in
        // it. The backend refuses /conditions for the same reason, so there is
        // nothing to fetch here either.
        <p style={{ color: "var(--color-text-secondary)" }}>{t("account.conditions.outsider")}</p>
      ) : (
        <MemberView open={open} level={me.roleLevel} />
      )}
    </Modal>
  );
}

// CeoEditor lets the CEO write and edit the requirement for every target rank.
function CeoEditor({ open }: { open: boolean }) {
  const { data, isPending } = useAllConditions(open);

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
        <Spinner />
      </div>
    );
  }

  return (
    <div className="cond">
      <p className="cond__hint">{t("account.conditions.ceoHint")}</p>
      {(data ?? []).map((c) => (
        <CeoRow key={c.targetLevel} condition={c} />
      ))}
    </div>
  );
}

function CeoRow({ condition }: { condition: LevelCondition }) {
  const [body, setBody] = useState(condition.body);
  const set = useSetCondition();

  // Keep the field in sync if the list refetches with new server data.
  useEffect(() => setBody(condition.body), [condition.body]);

  const dirty = body !== condition.body;

  const save = () => {
    set.mutate(
      { level: condition.targetLevel, body: body.trim() },
      {
        onSuccess: () => toast.success(t("account.conditions.saved", { level: condition.targetLevel })),
        onError: () => toast.error(t("account.conditions.saveFailed")),
      },
    );
  };

  return (
    <div className="cond__row">
      <div className="cond__row-head">
        <span className="cond__level">{t("account.conditions.level", { level: condition.targetLevel })}</span>
        {dirty && <span className="cond__dirty">{t("account.conditions.unsaved")}</span>}
      </div>
      <textarea
        className="ui-input cond__input"
        rows={2}
        maxLength={4000}
        placeholder={t("account.conditions.placeholder", { level: condition.targetLevel })}
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <div className="cond__row-actions">
        <Button variant="primary" onClick={save} loading={set.isPending} disabled={!dirty}>
          {t("account.conditions.save")}
        </Button>
      </div>
    </div>
  );
}

// MemberView shows only the requirement for the member's next level.
function MemberView({ open, level }: { open: boolean; level: number }) {
  const { data, isPending } = useNextCondition(open);

  if (isPending) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
        <Spinner />
      </div>
    );
  }

  const nextLevel = level - 1;
  // The number is set in bold inside the sentence: split it where "{level}" stands.
  const [beforeLevel, afterLevel = ""] = t("account.conditions.nextLevel").split("{level}");

  return (
    <div className="cond">
      <div className="cond__next-badge">
        {beforeLevel}
        <strong>{nextLevel >= 1 ? nextLevel : "—"}</strong>
        {afterLevel}
      </div>
      {!data || !data.body.trim() ? (
        <div className="cond__empty">
          {nextLevel < 1 ? t("account.conditions.topLevel") : t("account.conditions.notSet")}
        </div>
      ) : (
        <div className="cond__member-card">
          <div className="cond__member-title">{t("account.conditions.memberTitle", { level: data.targetLevel })}</div>
          <div className="cond__member-body">{data.body}</div>
        </div>
      )}
    </div>
  );
}
