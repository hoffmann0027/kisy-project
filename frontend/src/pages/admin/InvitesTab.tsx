import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button, IconButton, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { invitesApi } from "@shared/api/endpoints";
import type { Invitation } from "@shared/api/types";

export function InvitesTab() {
  const [invite, setInvite] = useState<Invitation | null>(null);

  const create = useMutation({
    mutationFn: () => invitesApi.create(),
    onSuccess: (inv) => setInvite(inv),
    onError: () => toast.error("Не удалось создать приглашение"),
  });

  const registrationLink = invite ? `${window.location.origin}/register?token=${encodeURIComponent(invite.token)}` : "";

  const copy = (text: string) => {
    void navigator.clipboard.writeText(text);
    toast.success("Скопировано");
  };

  return (
    <div style={{ maxWidth: 620, display: "flex", flexDirection: "column", gap: 20, paddingTop: 12 }}>
      <div style={{ color: "var(--color-text-secondary)", fontSize: 14, lineHeight: 1.5 }}>
        Приглашения создаёт только CEO. Токен действует <strong>ровно 120 секунд</strong> и может быть использован
        один раз. Передайте его новому сотруднику для регистрации.
      </div>

      <Button onClick={() => create.mutate()} loading={create.isPending} style={{ alignSelf: "flex-start" }}>
        Создать приглашение
      </Button>

      {invite && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12, animation: "rise-in 0.2s ease" }}>
          <div>
            <div style={{ fontSize: 13, color: "var(--color-text-secondary)", marginBottom: 6 }}>Токен</div>
            <div className="invite-box">
              <span style={{ flex: 1 }}>{invite.token}</span>
              <IconButton label="Копировать токен" onClick={() => copy(invite.token)}>
                <Icon.Copy size={18} />
              </IconButton>
            </div>
          </div>
          <div>
            <div style={{ fontSize: 13, color: "var(--color-text-secondary)", marginBottom: 6 }}>Ссылка регистрации</div>
            <div className="invite-box">
              <span style={{ flex: 1 }}>{registrationLink}</span>
              <IconButton label="Копировать ссылку" onClick={() => copy(registrationLink)}>
                <Icon.Copy size={18} />
              </IconButton>
            </div>
          </div>
          <div style={{ fontSize: 13, color: "var(--color-warning)" }}>
            ⚠ Истекает: {new Date(invite.expiresAt).toLocaleTimeString("ru-RU")}
          </div>
        </div>
      )}
    </div>
  );
}
