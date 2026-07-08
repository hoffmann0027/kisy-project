import { useEffect, useState } from "react";
import "./conditions.css";
import { Button, Modal, Spinner, toast } from "@shared/ui";
import type { LevelCondition } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useAllConditions, useNextCondition, useSetCondition } from "@entities/condition/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ConditionsModal({ open, onClose }: Props) {
  const me = useAuthStore((s) => s.user!);
  const isCEO = me.roleLevel === 1;

  return (
    <Modal open={open} title="Условия повышения уровня" onClose={onClose}>
      {isCEO ? <CeoEditor open={open} /> : <MemberView open={open} level={me.roleLevel} />}
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
      <p className="cond__hint">
        Каждый участник видит только одно условие — для своего следующего уровня. Уровень 1 — высший.
      </p>
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
        onSuccess: () => toast.success(`Условие для уровня ${condition.targetLevel} сохранено`),
        onError: () => toast.error("Не удалось сохранить"),
      },
    );
  };

  return (
    <div className="cond__row">
      <div className="cond__row-head">
        <span className="cond__level">Уровень {condition.targetLevel}</span>
        {dirty && <span className="cond__dirty">не сохранено</span>}
      </div>
      <textarea
        className="ui-input cond__input"
        rows={2}
        maxLength={4000}
        placeholder={`Что нужно, чтобы получить уровень ${condition.targetLevel}…`}
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <div className="cond__row-actions">
        <Button variant="primary" onClick={save} loading={set.isPending} disabled={!dirty}>
          Сохранить
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

  return (
    <div className="cond">
      <div className="cond__next-badge">
        Ваш следующий уровень: <strong>{nextLevel >= 1 ? nextLevel : "—"}</strong>
      </div>
      {!data || !data.body.trim() ? (
        <div className="cond__empty">
          {nextLevel < 1
            ? "Вы уже на высшем уровне."
            : "Условие для следующего уровня пока не задано."}
        </div>
      ) : (
        <div className="cond__member-card">
          <div className="cond__member-title">Чтобы получить уровень {data.targetLevel}:</div>
          <div className="cond__member-body">{data.body}</div>
        </div>
      )}
    </div>
  );
}
