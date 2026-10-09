import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Avatar, Button, VerifiedName, toast } from "@shared/ui";
import { t } from "@shared/i18n";
import { usersApi } from "@shared/api/endpoints";
import { ApiError } from "@shared/api/envelope";
import { userSubtitle, type RatingProject } from "@shared/api/types";
import type { useRatingMutations } from "@entities/rating/queries";

interface Props {
  project: RatingProject;
  m: ReturnType<typeof useRatingMutations>;
  onDone: () => void;
}

/** Find a person in the directory and put them on the project. */
export function MemberPicker({ project, m, onDone }: Props) {
  const [query, setQuery] = useState("");
  const { data } = useQuery({
    queryKey: ["directory", "rating-member", query],
    queryFn: async () => (await usersApi.directory(query)).users,
  });
  const onIt = new Set(project.members.map((u) => u.id));
  const candidates = (data ?? []).filter((u) => !onIt.has(u.id));

  const pick = (userId: string, name: string) => {
    m.addMember.mutate(
      { projectId: project.id, userId },
      {
        onSuccess: () => {
          toast.success(t("work.rating.memberAdded", { name }));
          onDone();
        },
        onError: (e) =>
          toast.error(e instanceof ApiError && e.status === 409 ? t("work.rating.memberCannotSee") : t("work.rating.addMemberFailed")),
      },
    );
  };

  return (
    <div className="rmembers__picker">
      <input className="ui-input" placeholder={t("work.rating.searchUsers")} autoFocus value={query} onChange={(e) => setQuery(e.target.value)} />
      <div className="rmembers__results">
        {candidates.map((u) => (
          <button key={u.id} type="button" className="user-row" onClick={() => pick(u.id, u.displayName)} disabled={m.addMember.isPending}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={30} />
            <div>
              <div className="user-row__name">
                <VerifiedName name={u.displayName} verified={!!u.verifiedAt} />
              </div>
              <div className="user-row__role">{userSubtitle(u)}</div>
            </div>
          </button>
        ))}
      </div>
      <Button variant="ghost" onClick={onDone}>
        {t("work.rating.cancel")}
      </Button>
    </div>
  );
}
