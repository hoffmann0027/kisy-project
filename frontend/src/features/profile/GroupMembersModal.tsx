import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Avatar, Button, Modal, Spinner, VerifiedName, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { ROLE_LABELS, roleLabel, userSubtitle, type Group, type GroupRole, type JoinPolicy, type PostPolicy } from "@shared/api/types";
import { groupsApi, usersApi } from "@shared/api/endpoints";
import {
  groupKeys,
  useAddMember,
  useDecideRequest,
  useDeleteGroup,
  useGroupBans,
  useGroupMembers,
  useGroupRequests,
  useLeaveGroup,
  useRemoveMember,
  useSetBan,
  useSetMemberRole,
  useUpdateGroupLevel,
  useUpdateGroupSettings,
} from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { ApiError, userFacingError } from "@shared/api/envelope";
import { AvatarCropper } from "./AvatarCropper";
import { ReportButton } from "@features/reports/ReportButton";
import { t, type Key } from "@shared/i18n";

interface Props {
  group: Group;
  canAdd: boolean;
  open: boolean;
  onClose: () => void;
}

const EDITOR_TIER: GroupRole[] = ["owner", "editor", "moderator"];

// Who may show whom the door, as the server decides it (groups.mayDiscipline):
// the founder and owners over editors, editors over moderators, moderators
// over members. Equals do not remove each other.
const RANK: Record<GroupRole, number> = { owner: 3, editor: 2, moderator: 1, member: 0 };
function rankOf(group: Group, userId: string, role: GroupRole | undefined): number {
  if (userId === group.createdBy) return RANK.owner;
  return role === undefined ? -1 : RANK[role];
}

// Human labels for the in-group roles (keys, resolved at render).
const GROUP_ROLE_LABEL: Record<GroupRole, Key> = {
  owner: "account.groupMembers.roleOwner",
  editor: "account.groupMembers.roleEditor",
  moderator: "account.groupMembers.roleModerator",
  member: "account.groupMembers.roleMember",
};

export function GroupMembersModal({ group, canAdd, open, onClose }: Props) {
  const { data: members, isPending } = useGroupMembers(open ? group.id : null);
  const [adding, setAdding] = useState(false);
  const navigate = useNavigate();
  const del = useDeleteGroup();
  const qc = useQueryClient();
  const me = useAuthStore((s) => s.user!);
  const updateLevel = useUpdateGroupLevel();
  const updateSettings = useUpdateGroupSettings();
  const setRole = useSetMemberRole();
  const caps = useCapabilities();
  const isCommunity = group.kind === "community";
  // The CEO may manage any group; the founder their own.
  const isCEO = caps.canAdmin;
  const isOwner = isCEO || me.id === group.createdBy;
  const canManage = isOwner;
  const canDelete = canManage;
  // My own in-group role (for the editor tier → may approve requests).
  const myRole = members?.find((m) => m.user.id === me.id)?.role;
  const canApprove = isOwner || (myRole !== undefined && EDITOR_TIER.includes(myRole));

  const { data: requests } = useGroupRequests(group.id, open && canApprove);
  const decide = useDecideRequest();
  // Leaving, removal and bans.
  const leave = useLeaveGroup();
  const removeMember = useRemoveMember();
  const setBan = useSetBan();
  const { data: bans } = useGroupBans(group.id, open && canApprove);
  const isFounder = me.id === group.createdBy;
  const amMember = myRole !== undefined;
  const myRank = rankOf(group, me.id, myRole);
  const mayDiscipline = (userId: string, role: GroupRole) =>
    userId !== me.id && userId !== group.createdBy && (isCEO || (myRank >= 1 && myRank > rankOf(group, userId, role)));

  const leaveGroup = () => {
    const question = isCommunity ? "account.groupMembers.confirmLeaveCommunity" : "account.groupMembers.confirmLeaveGroup";
    if (!window.confirm(t(question, { name: group.name }))) return;
    leave.mutate(group.id, {
      onSuccess: () => {
        toast.success(isCommunity ? t("account.groupMembers.leftCommunity") : t("account.groupMembers.leftGroup"));
        onClose();
        navigate("/communities", { replace: true });
      },
      onError: (e) =>
        toast.error(
          e instanceof ApiError && e.status === 409 ? t("account.groupMembers.founderStays") : t("account.groupMembers.actionFailed"),
        ),
    });
  };

  const disciplineError = (e: unknown) =>
    toast.error(e instanceof ApiError && e.status === 403 ? t("account.groupMembers.notAllowed") : t("account.groupMembers.actionFailed"));

  const kick = (userId: string, name: string) => {
    if (!window.confirm(t("account.groupMembers.confirmRemove", { name }))) return;
    removeMember.mutate(
      { groupId: group.id, userId },
      { onSuccess: () => toast.success(t("account.groupMembers.memberRemoved", { name })), onError: disciplineError },
    );
  };

  const ban = (userId: string, name: string) => {
    if (!window.confirm(t("account.groupMembers.confirmBan", { name }))) return;
    setBan.mutate(
      { groupId: group.id, userId, banned: true },
      { onSuccess: () => toast.success(t("account.groupMembers.memberBanned", { name })), onError: disciplineError },
    );
  };

  const unban = (userId: string, name: string) => {
    setBan.mutate(
      { groupId: group.id, userId, banned: false },
      { onSuccess: () => toast.success(t("account.groupMembers.memberUnbanned", { name })), onError: disciplineError },
    );
  };

  const changeLevel = (level: number) => {
    if (level === group.minRoleLevel) return;
    updateLevel.mutate(
      { groupId: group.id, minRoleLevel: level },
      {
        onSuccess: () => toast.success(t("account.groupMembers.levelChanged")),
        onError: () => toast.error(t("account.groupMembers.levelChangeFailed")),
      },
    );
  };

  const changeAccess = (value: string) => {
    const [joinPolicy, postPolicy] = value.split(":") as [JoinPolicy, PostPolicy];
    if (joinPolicy === group.joinPolicy && postPolicy === group.postPolicy) return;
    updateSettings.mutate(
      { groupId: group.id, joinPolicy, postPolicy },
      {
        onSuccess: () => toast.success(t("account.groupMembers.accessUpdated")),
        onError: () => toast.error(t("account.groupMembers.accessUpdateFailed")),
      },
    );
  };

  const toggleEditor = (userId: string, current: GroupRole) => {
    const role: GroupRole = current === "editor" ? "member" : "editor";
    setRole.mutate(
      { groupId: group.id, userId, role },
      { onError: () => toast.error(t("account.groupMembers.roleChangeFailed")) },
    );
  };

  const uploadGroupAvatar = async (blob: Blob) => {
    await groupsApi.uploadAvatar(group.id, blob);
    qc.invalidateQueries({ queryKey: groupKeys.list });
  };

  const removeGroup = () => {
    const question = isCommunity ? "account.groupMembers.confirmDeleteCommunity" : "account.groupMembers.confirmDeleteGroup";
    if (!window.confirm(t(question, { name: group.name }))) return;
    del.mutate(group.id, {
      onSuccess: () => {
        toast.success(isCommunity ? t("account.groupMembers.communityDeleted") : t("account.groupMembers.groupDeleted"));
        onClose();
        navigate("/", { replace: true });
      },
      onError: (e) => toast.error(userFacingError(e, t("account.groupMembers.deleteFailed"))),
    });
  };

  const pendingCount = requests?.length ?? 0;

  return (
    <Modal open={open} title={t("account.groupMembers.title", { name: group.name })} onClose={onClose}>
      <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
        {canManage ? (
          <AvatarCropper name={group.name} url={group.avatarUrl} size={56} onUpload={uploadGroupAvatar} />
        ) : (
          <Avatar name={group.name} url={group.avatarUrl} size={56} />
        )}
        <div>
          <div style={{ fontWeight: 640, fontSize: 17 }}>
            <VerifiedName name={group.name} verified={!!group.verifiedAt} size={20} subject="group" />
          </div>
          <div style={{ color: "var(--color-text-secondary)", fontSize: 14 }}>
            {canManage
              ? t("account.groupMembers.tapAvatar")
              : isCommunity
                ? t("account.groupMembers.kindCommunity")
                : t("account.groupMembers.kindGroup")}
          </div>
        </div>
      </div>

      {/* Levels are a fact of the hierarchy. An account outside it has none,
          cannot change this field and learns nothing from it — every group it
          can see is one without a threshold — so it is not shown at all. */}
      {caps.canSeeLevels && (
      <div className="ui-field">
        <label className="ui-field__label">{t("account.groupMembers.accessLevel")}</label>
        {group.minRoleLevel === null ? (
          // No threshold at all. Not shown as a level, because it is not one:
          // this group is open to everyone, accounts outside the hierarchy
          // included.
          <div className="ui-input" style={{ display: "flex", alignItems: "center" }}>
            {t("account.groupMembers.noLevel")}
          </div>
        ) : isCEO ? (
          <select className="ui-input" value={group.minRoleLevel} disabled={updateLevel.isPending} onChange={(e) => changeLevel(Number(e.target.value))}>
            {Object.entries(ROLE_LABELS).map(([lvl, label]) => (
              <option key={lvl} value={lvl}>
                {lvl}. {label}
              </option>
            ))}
          </select>
        ) : (
          <div className="ui-input" style={{ display: "flex", alignItems: "center" }}>
            {group.minRoleLevel}. {roleLabel(group.minRoleLevel)}
          </div>
        )}
        <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
          {group.minRoleLevel === null
            ? t("account.groupMembers.openToAll")
            : isCEO
              ? t("account.groupMembers.ceoLevelHint")
              : t("account.groupMembers.levelHint")}
        </span>
      </div>
      )}

      {canManage && (
        <div className="ui-field">
          <label className="ui-field__label">{t("account.groupMembers.access")}</label>
          <select
            className="ui-input"
            value={`${group.joinPolicy}:${group.postPolicy}`}
            disabled={updateSettings.isPending}
            onChange={(e) => changeAccess(e.target.value)}
          >
            <option value="open:all">{t("account.groupMembers.accessOpenAll")}</option>
            <option value="open:editors">{t("account.groupMembers.accessOpenEditors")}</option>
            <option value="request:all">{t("account.groupMembers.accessRequestAll")}</option>
            <option value="request:editors">{t("account.groupMembers.accessRequestEditors")}</option>
          </select>
          <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
            {t("account.groupMembers.accessHint")}
            {caps.canSeeLevels && ` ${t("account.groupMembers.clearanceHint")}`}
          </span>
        </div>
      )}

      {canApprove && pendingCount > 0 && (
        <div className="ui-field">
          <label className="ui-field__label">{t("account.groupMembers.requests", { count: pendingCount })}</label>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {requests?.map((u) => (
              <div key={u.id} className="user-row" style={{ cursor: "default" }}>
                <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
                <div style={{ flex: 1 }}>
                  <div className="user-row__name">
                    <VerifiedName name={u.displayName} verified={!!u.verifiedAt} />
                  </div>
                  <div className="user-row__role">{userSubtitle(u)}</div>
                </div>
                <div style={{ display: "flex", gap: 6 }}>
                  <Button variant="secondary" loading={decide.isPending} onClick={() => decide.mutate({ groupId: group.id, userId: u.id, approve: true })}>
                    {t("account.groupMembers.accept")}
                  </Button>
                  <Button variant="ghost" onClick={() => decide.mutate({ groupId: group.id, userId: u.id, approve: false })}>
                    {t("account.groupMembers.decline")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {canAdd && !adding && (
        <Button variant="secondary" onClick={() => setAdding(true)}>
          {t("account.groupMembers.addMember")}
        </Button>
      )}
      {adding && <AddMemberPicker group={group} onDone={() => setAdding(false)} />}

      <div style={{ maxHeight: 320, overflowY: "auto", display: "flex", flexDirection: "column", gap: 2 }}>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
            <Spinner />
          </div>
        )}
        {members?.map((m) => {
          const founder = m.user.id === group.createdBy;
          return (
            <div key={m.user.id} className="user-row" style={{ cursor: "default" }}>
              <Avatar name={m.user.displayName} url={m.user.avatarUrl} size={38} />
              <div style={{ flex: 1 }}>
                <div className="user-row__name">
                  <VerifiedName name={m.user.displayName} verified={!!m.user.verifiedAt} />
                </div>
                <div className="user-row__role">
                  {userSubtitle(m.user)}
                  {founder
                    ? ` · ${t("account.groupMembers.founder")}`
                    : m.role !== "member"
                      ? ` · ${t(GROUP_ROLE_LABEL[m.role])}`
                      : ""}
                </div>
              </div>
              {canManage && !founder && (
                <Button variant="ghost" loading={setRole.isPending} onClick={() => toggleEditor(m.user.id, m.role)}>
                  {m.role === "editor" ? t("account.groupMembers.removeEditor") : t("account.groupMembers.makeEditor")}
                </Button>
              )}
              {mayDiscipline(m.user.id, m.role) && (
                <>
                  <button
                    type="button"
                    className="ui-icon-btn"
                    title={t("account.groupMembers.removeMember")}
                    aria-label={`${t("account.groupMembers.removeMember")}: ${m.user.displayName}`}
                    disabled={removeMember.isPending}
                    onClick={() => kick(m.user.id, m.user.displayName)}
                  >
                    <Icon.X size={16} />
                  </button>
                  <button
                    type="button"
                    className="ui-icon-btn"
                    title={t("account.groupMembers.banMember")}
                    aria-label={`${t("account.groupMembers.banMember")}: ${m.user.displayName}`}
                    disabled={setBan.isPending}
                    onClick={() => ban(m.user.id, m.user.displayName)}
                  >
                    <Icon.Ban size={16} />
                  </button>
                </>
              )}
              {/* The one place someone you have never written to is visible:
                  a community's members. Without this, reporting a person
                  required opening a private chat with them first. */}
              {m.user.id !== me.id && (
                <ReportButton
                  targetKind="user"
                  targetId={m.user.id}
                  label={t("account.groupMembers.report", { name: m.user.displayName })}
                  className="ui-icon-btn"
                  size={16}
                />
              )}
            </div>
          );
        })}
      </div>

      {canApprove && bans && bans.length > 0 && (
        <div className="ui-field">
          <label className="ui-field__label">{t("account.groupMembers.bannedList")}</label>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {bans.map((b) => (
              <div key={b.user.id} className="user-row" style={{ cursor: "default" }}>
                <Avatar name={b.user.displayName} url={b.user.avatarUrl} size={34} />
                <div style={{ flex: 1 }}>
                  <div className="user-row__name">
                    <VerifiedName name={b.user.displayName} verified={!!b.user.verifiedAt} />
                  </div>
                  <div className="user-row__role">{userSubtitle(b.user)}</div>
                </div>
                <Button variant="ghost" loading={setBan.isPending} onClick={() => unban(b.user.id, b.user.displayName)}>
                  {t("account.groupMembers.unbanMember")}
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* The founder deletes; everyone else leaves. */}
      {amMember && !isFounder && (
        <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 14 }}>
          <Button variant="ghost" block loading={leave.isPending} onClick={leaveGroup}>
            {isCommunity ? t("account.groupMembers.leaveCommunity") : t("account.groupMembers.leaveGroup")}
          </Button>
        </div>
      )}
      {canDelete && (
        <div style={{ borderTop: "1px solid var(--color-border)", paddingTop: 14 }}>
          <Button variant="danger" block loading={del.isPending} onClick={removeGroup}>
            {isCommunity ? t("account.groupMembers.deleteCommunity") : t("account.groupMembers.deleteGroup")}
          </Button>
        </div>
      )}
    </Modal>
  );
}

function AddMemberPicker({ group, onDone }: { group: Group; onDone: () => void }) {
  const [query, setQuery] = useState("");
  const add = useAddMember();
  const { data } = useQuery({
    queryKey: ["directory", "group-add", query],
    queryFn: async () => (await usersApi.directory(query)).users,
  });

  const pick = (userId: string) => {
    add.mutate(
      { groupId: group.id, userId },
      {
        onSuccess: () => {
          toast.success(t("account.groupMembers.memberAdded"));
          onDone();
        },
        onError: (e) =>
          toast.error(
            e instanceof ApiError && e.status === 409
              ? t("account.groupMembers.alreadyMember")
              : e instanceof ApiError && e.status === 403
                ? t("account.groupMembers.ownerOnly")
                : t("account.groupMembers.addFailed"),
          ),
      },
    );
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      <input className="ui-input" placeholder={t("account.groupMembers.searchPlaceholder")} autoFocus value={query} onChange={(e) => setQuery(e.target.value)} />
      <div style={{ maxHeight: 200, overflowY: "auto" }}>
        {data?.map((u) => (
          <button key={u.id} className="user-row" onClick={() => pick(u.id)} disabled={add.isPending}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={34} />
            <div>
              <div className="user-row__name">
                <VerifiedName name={u.displayName} verified={!!u.verifiedAt} />
              </div>
              <div className="user-row__role">
                {userSubtitle(u)}
              </div>
            </div>
          </button>
        ))}
      </div>
      <Button variant="ghost" onClick={onDone}>
        {t("account.groupMembers.done")}
      </Button>
    </div>
  );
}
