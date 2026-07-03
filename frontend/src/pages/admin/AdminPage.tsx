import { useState } from "react";
import { useNavigate } from "react-router-dom";
import "./admin.css";
import { IconButton } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { UsersTab } from "./UsersTab";
import { InvitesTab } from "./InvitesTab";
import { AuditTab } from "./AuditTab";

type Tab = "users" | "invites" | "audit";

export function AdminPage() {
  const [tab, setTab] = useState<Tab>("users");
  const navigate = useNavigate();

  return (
    <div className="admin">
      <div className="admin__header">
        <h1 className="admin__title">Администрирование</h1>
        <IconButton label="Назад к чатам" onClick={() => navigate("/")}>
          <Icon.Back />
        </IconButton>
      </div>

      <div className="admin__tabs">
        <button className={tabCn(tab === "users")} onClick={() => setTab("users")}>
          Пользователи
        </button>
        <button className={tabCn(tab === "invites")} onClick={() => setTab("invites")}>
          Приглашения
        </button>
        <button className={tabCn(tab === "audit")} onClick={() => setTab("audit")}>
          Аудит
        </button>
      </div>

      <div className="admin__content">
        {tab === "users" && <UsersTab />}
        {tab === "invites" && <InvitesTab />}
        {tab === "audit" && <AuditTab />}
      </div>
    </div>
  );
}

function tabCn(active: boolean): string {
  return active ? "admin__tab admin__tab--active" : "admin__tab";
}
