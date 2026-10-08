import { useState } from "react";
import "./voting.css";
import { Avatar, Button, Modal, Spinner, toast } from "@shared/ui";
import { cn } from "@shared/lib/cn";
import { t } from "@shared/i18n";
import type { Poll, PollOption } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { usePollMutations, usePolls } from "@entities/poll/queries";

type Mutations = ReturnType<typeof usePollMutations>;

interface Props {
  open: boolean;
  onClose: () => void;
}

export function VotingModal({ open, onClose }: Props) {
  const me = useAuthStore((s) => s.user!);
  const isCEO = me.roleLevel === 1;
  const { data: polls, isPending } = usePolls(open);
  const m = usePollMutations();

  return (
    <Modal open={open} title={t("work.poll.title")} onClose={onClose}>
      <div className="poll">
        {isCEO && <CreatePollForm m={m} />}

        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
            <Spinner />
          </div>
        )}
        {!isPending && (polls?.length ?? 0) === 0 && (
          <div className="poll__empty">
            {isCEO ? t("work.poll.emptyCeo") : t("work.poll.empty")}
          </div>
        )}
        {polls?.map((p) => (
          <PollCard key={p.id} poll={p} m={m} isCEO={isCEO} />
        ))}
      </div>
    </Modal>
  );
}

function CreatePollForm({ m }: { m: Mutations }) {
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState<string[]>(["", ""]);

  const reset = () => {
    setQuestion("");
    setOptions(["", ""]);
    setOpen(false);
  };

  const setOption = (i: number, v: string) =>
    setOptions((prev) => prev.map((o, idx) => (idx === i ? v : o)));
  const addOption = () => setOptions((prev) => (prev.length >= 10 ? prev : [...prev, ""]));
  const removeOption = (i: number) =>
    setOptions((prev) => (prev.length <= 2 ? prev : prev.filter((_, idx) => idx !== i)));

  const submit = () => {
    const q = question.trim();
    const opts = options.map((o) => o.trim()).filter(Boolean);
    if (!q) {
      toast.error(t("work.poll.enterQuestion"));
      return;
    }
    if (opts.length < 2) {
      toast.error(t("work.poll.minTwoOptions"));
      return;
    }
    m.create.mutate(
      { question: q, options: opts },
      { onSuccess: reset, onError: () => toast.error(t("work.poll.createFailed")) },
    );
  };

  if (!open) {
    return (
      <button className="poll__add" onClick={() => setOpen(true)}>
        {t("work.poll.new")}
      </button>
    );
  }

  return (
    <div className="poll__create">
      <input
        className="ui-input"
        placeholder={t("work.poll.questionPlaceholder")}
        autoFocus
        maxLength={500}
        value={question}
        onChange={(e) => setQuestion(e.target.value)}
      />
      <div className="poll__options-edit">
        {options.map((o, i) => (
          <div key={i} className="poll__option-edit">
            <input
              className="ui-input"
              placeholder={t("work.poll.optionPlaceholder", { n: i + 1 })}
              maxLength={200}
              value={o}
              onChange={(e) => setOption(i, e.target.value)}
            />
            {options.length > 2 && (
              <button className="poll__opt-remove" title={t("work.poll.removeOption")} onClick={() => removeOption(i)}>
                ✕
              </button>
            )}
          </div>
        ))}
      </div>
      <div className="poll__create-actions">
        {options.length < 10 && (
          <button className="poll__link" onClick={addOption}>
            {t("work.poll.addOption")}
          </button>
        )}
        <div className="poll__spacer" />
        <Button variant="ghost" onClick={reset}>
          {t("work.poll.cancel")}
        </Button>
        <Button variant="primary" onClick={submit} loading={m.create.isPending}>
          {t("work.poll.create")}
        </Button>
      </div>
    </div>
  );
}

function PollCard({ poll, m, isCEO }: { poll: Poll; m: Mutations; isCEO: boolean }) {
  const closed = poll.status === "closed";

  const vote = (optionId: string) => {
    if (closed || optionId === poll.myOptionId) return;
    m.vote.mutate(optionId, { onError: () => toast.error(t("work.poll.voteFailed")) });
  };

  const close = () => {
    if (!window.confirm(t("work.poll.confirmClose"))) return;
    m.close.mutate(poll.id, { onError: () => toast.error(t("work.poll.closeFailed")) });
  };

  const remove = () => {
    if (!window.confirm(t("work.poll.confirmDelete"))) return;
    m.del.mutate(poll.id, { onError: () => toast.error(t("work.poll.deleteFailed")) });
  };

  return (
    <div className="poll__card">
      <div className="poll__card-head">
        <div className="poll__question">{poll.question}</div>
        <span className={cn("poll__status", closed ? "poll__status--closed" : "poll__status--open")}>
          {closed ? t("work.poll.statusClosed") : t("work.poll.statusOpen")}
        </span>
      </div>

      <div className="poll__options">
        {poll.options.map((o) => (
          <OptionRow
            key={o.id}
            option={o}
            total={poll.totalVotes}
            mine={o.id === poll.myOptionId}
            closed={closed}
            onVote={() => vote(o.id)}
          />
        ))}
      </div>

      <div className="poll__card-foot">
        <span className="poll__total">{t("work.poll.totalVotes", { total: poll.totalVotes })}</span>
        {isCEO && (
          <div className="poll__admin">
            {!closed && (
              <button className="poll__link" onClick={close}>
                {t("work.poll.close")}
              </button>
            )}
            <button className="poll__link poll__link--danger" onClick={remove}>
              {t("work.poll.delete")}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function OptionRow({
  option,
  total,
  mine,
  closed,
  onVote,
}: {
  option: PollOption;
  total: number;
  mine: boolean;
  closed: boolean;
  onVote: () => void;
}) {
  const pct = total > 0 ? Math.round((option.votes / total) * 100) : 0;

  return (
    <div
      className={cn("poll__option", mine && "poll__option--mine", closed && "poll__option--closed")}
      onClick={onVote}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && onVote()}
    >
      <div className="poll__option-bar" style={{ width: `${pct}%` }} />
      <div className="poll__option-content">
        <div className="poll__option-top">
          <span className="poll__option-body">
            {mine && <span className="poll__check">✓</span>}
            {option.body}
          </span>
          <span className="poll__option-pct">
            {option.votes} · {pct}%
          </span>
        </div>
        {option.voters.length > 0 && (
          <div className="poll__voters">
            {option.voters.map((v) => (
              <span key={v.id} className="poll__voter" title={v.displayName}>
                <Avatar name={v.displayName} url={v.avatarUrl} size={22} />
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
