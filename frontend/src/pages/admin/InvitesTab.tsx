import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button, IconButton, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { invitesApi } from "@shared/api/endpoints";
import type { Invitation } from "@shared/api/types";
import { intlLocale, t } from "@shared/i18n";
import { interpolate } from "./interpolate";

export function InvitesTab() {
  const [invite, setInvite] = useState<Invitation | null>(null);

  const create = useMutation({
    mutationFn: () => invitesApi.create(),
    onSuccess: (inv) => setInvite(inv),
    onError: () => toast.error(t("admin.invites.createFailed")),
  });

  const registrationLink = invite ? `${window.location.origin}/register?token=${encodeURIComponent(invite.token)}` : "";

  const copy = (text: string) => {
    void navigator.clipboard.writeText(text);
    toast.success(t("admin.invites.copied"));
  };

  return (
    <div style={{ maxWidth: 620, display: "flex", flexDirection: "column", gap: 20, paddingTop: 12 }}>
      <div style={{ color: "var(--color-text-secondary)", fontSize: 14, lineHeight: 1.5 }}>
        {interpolate(t("admin.invites.intro"), { duration: <strong>{t("admin.invites.introDuration")}</strong> })}
      </div>

      <Button onClick={() => create.mutate()} loading={create.isPending} style={{ alignSelf: "flex-start" }}>
        {t("admin.invites.create")}
      </Button>

      {invite && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12, animation: "rise-in 0.2s ease" }}>
          <div>
            <div style={{ fontSize: 13, color: "var(--color-text-secondary)", marginBottom: 6 }}>{t("admin.invites.token")}</div>
            <div className="invite-box">
              <span style={{ flex: 1 }}>{invite.token}</span>
              <IconButton label={t("admin.invites.copyToken")} onClick={() => copy(invite.token)}>
                <Icon.Copy size={18} />
              </IconButton>
            </div>
          </div>
          <div>
            <div style={{ fontSize: 13, color: "var(--color-text-secondary)", marginBottom: 6 }}>{t("admin.invites.link")}</div>
            <div className="invite-box">
              <span style={{ flex: 1 }}>{registrationLink}</span>
              <IconButton label={t("admin.invites.copyLink")} onClick={() => copy(registrationLink)}>
                <Icon.Copy size={18} />
              </IconButton>
            </div>
          </div>
          <div style={{ fontSize: 13, color: "var(--color-warning)" }}>
            {t("admin.invites.expires", { time: new Date(invite.expiresAt).toLocaleTimeString(intlLocale()) })}
          </div>
        </div>
      )}
    </div>
  );
}
