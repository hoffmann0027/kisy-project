import { useState } from "react";
import { t } from "@shared/i18n";
import { Button, Input, Modal, toast } from "@shared/ui";
import { ROLE_LABELS, type Group, type GroupKind } from "@shared/api/types";
import { useCreateGroup } from "@entities/group/queries";
import { useAuthStore } from "@shared/store/auth";
import { useCapabilities } from "@shared/lib/useCapabilities";
import { heldBackNotice } from "@shared/lib/quarantine";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: (group: Group) => void;
}

export function NewGroupModal({ open, onClose, onCreated }: Props) {
  const myLevel = useAuthStore((s) => s.user?.roleLevel ?? null);
  const caps = useCapabilities();
  const [kind, setKind] = useState<GroupKind>("group");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isPublic, setIsPublic] = useState(true);
  // Only ever consulted for an account that has a level (see below).
  const [minRoleLevel, setMinRoleLevel] = useState(myLevel ?? 10);
  const create = useCreateGroup();
  // A brand-new account may not found a community yet (ordinary groups are
  // unaffected — see internal/quarantine).
  const heldCommunity = heldBackNotice(useAuthStore((s) => s.quarantine), t("chat.newGroup.communityHeldBack"));

  // A user may only create a group whose minimum clearance is their own
  // level or weaker (numerically >= their level).
  const levelOptions = Object.entries(ROLE_LABELS).filter(([lvl]) => Number(lvl) >= (myLevel ?? 1));

  const submit = () => {
    if (kind === "community" && heldCommunity) {
      toast.error(heldCommunity);
      return;
    }
    if (name.trim().length < 1) {
      toast.error(kind === "community" ? t("chat.newGroup.enterCommunityName") : t("chat.newGroup.enterGroupName"));
      return;
    }
    create.mutate(
      {
        name: name.trim(),
        // An account outside the hierarchy has no threshold to choose, so it
        // sends none: the group is open to everyone.
        minRoleLevel: caps.canSeeLevels ? minRoleLevel : null,
        description: description.trim() || undefined,
        kind,
        isPublic: kind === "community" && isPublic,
      },
      {
        onSuccess: ({ group }) => {
          toast.success(kind === "community" ? t("chat.newGroup.communityCreated") : t("chat.newGroup.groupCreated"));
          setName("");
          setDescription("");
          onCreated(group);
          onClose();
        },
        onError: () => toast.error(t("chat.newGroup.createFailed")),
      },
    );
  };

  return (
    <Modal open={open} title={kind === "community" ? t("chat.newGroup.newCommunity") : t("chat.list.newGroup")} onClose={onClose}>
      <div className="ui-field">
        <label className="ui-field__label">{t("chat.newGroup.kindLabel")}</label>
        <div style={{ display: "flex", gap: 8 }}>
          <Button variant={kind === "group" ? "primary" : "secondary"} onClick={() => setKind("group")}>
            {t("chat.group.group")}
          </Button>
          <Button variant={kind === "community" ? "primary" : "secondary"} onClick={() => setKind("community")}>
            {t("chat.group.community")}
          </Button>
        </div>
        <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
          {kind === "group"
            ? t("chat.newGroup.groupExplain")
            : t("chat.newGroup.communityExplain")}
        </span>
        {kind === "community" && heldCommunity && (
          <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>{heldCommunity}</span>
        )}
      </div>

      <Input
        label={t("chat.newGroup.name")}
        placeholder={kind === "community" ? t("chat.newGroup.communityNamePlaceholder") : t("chat.newGroup.groupNamePlaceholder")}
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      <Input label={t("chat.newGroup.description")} value={description} onChange={(e) => setDescription(e.target.value)} />

      {/* No level, no selector. An empty or disabled dropdown would be asking a
          question this account cannot answer; its groups are simply open. */}
      {caps.canSeeLevels && (
        <div className="ui-field">
          <label className="ui-field__label">{t("chat.newGroup.minLevel")}</label>
          <select className="ui-input" value={minRoleLevel} onChange={(e) => setMinRoleLevel(Number(e.target.value))}>
            {levelOptions.map(([lvl, label]) => (
              <option key={lvl} value={lvl}>
                {lvl}. {label}
              </option>
            ))}
          </select>
          <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
            {t("chat.newGroup.minLevelHint")}
          </span>
        </div>
      )}

      {kind === "community" && (
        <label className="ui-field" style={{ flexDirection: "row", alignItems: "center", gap: 10 }}>
          <input type="checkbox" checked={isPublic} onChange={(e) => setIsPublic(e.target.checked)} />
          <span>
            <span style={{ display: "block" }}>{t("chat.group.publicCommunity")}</span>
            <span style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}>
              {t("chat.newGroup.publicHint")}
            </span>
          </span>
        </label>
      )}

      <Button block loading={create.isPending} onClick={submit}>
        {kind === "community" ? t("chat.newGroup.createCommunity") : t("chat.newGroup.createGroup")}
      </Button>
    </Modal>
  );
}
