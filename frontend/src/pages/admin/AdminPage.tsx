import { useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import "./admin.css";
import { Button, IconButton, Logo } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { adminApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { AnnouncementsModal } from "@features/announcements/AnnouncementsModal";
import { UsersTab } from "./UsersTab";
import { InvitesTab } from "./InvitesTab";
import { AuditTab } from "./AuditTab";
import { VerificationTab } from "./VerificationTab";
import { CommunitiesTab } from "./CommunitiesTab";
import { DeletedTab } from "./DeletedTab";
import { ReportsTab } from "./ReportsTab";
import { OverviewTab, type AdminSection } from "./OverviewTab";
import { UpdatesTab } from "./UpdatesTab";

// The CEO's panel (October 2026 redesign): a menu of sections on the left — a
// strip across the top on a phone — with "Обзор" first, and the three things
// done most often as buttons in the header.

const SECTIONS: { id: AdminSection; label: string; icon: ReactNode }[] = [
  { id: "overview", label: "Обзор", icon: <Icon.Grid /> },
  { id: "users", label: "Пользователи", icon: <Icon.Users /> },
  { id: "invites", label: "Приглашения", icon: <Icon.Plus /> },
  { id: "verification", label: "Верификация", icon: <Icon.Check /> },
  { id: "communities", label: "Сообщества", icon: <Icon.Community /> },
  { id: "reports", label: "Жалобы", icon: <Icon.Flag /> },
  { id: "updates", label: "Обновления", icon: <Icon.Send /> },
  { id: "deleted", label: "Удалённые", icon: <Icon.Trash /> },
  { id: "audit", label: "Аудит", icon: <Icon.Shield /> },
];

export function AdminPage() {
  const [section, setSection] = useState<AdminSection>("overview");
  const [broadcast, setBroadcast] = useState(false);
  const navigate = useNavigate();
  const me = useAuthStore((s) => s.user!);
  // Shares the overview's query: the reports badge costs no extra request.
  const { data: overview } = useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: () => adminApi.dashboard(),
    refetchInterval: 60_000,
  });
  const openReports = overview?.inbox.openReports ?? 0;

  const showStatus = () => {
    setSection("overview");
    requestAnimationFrame(() => document.getElementById("dash-system")?.scrollIntoView({ behavior: "smooth", block: "center" }));
  };

  return (
    <div className="admin admin--panel">
      <aside className="admin-nav" aria-label="Разделы администрирования">
        <div className="admin-nav__brand">
          <Logo size={36} />
          <div>
            <div className="admin-nav__name">KISY</div>
            <div className="admin-nav__role">Панель CEO</div>
          </div>
        </div>
        <nav className="admin-nav__list">
          {SECTIONS.map((s) => (
            <button
              key={s.id}
              className={"admin-nav__item" + (s.id === section ? " is-active" : "")}
              aria-current={s.id === section ? "page" : undefined}
              onClick={() => setSection(s.id)}
            >
              <span className="admin-nav__icon" aria-hidden="true">
                {s.icon}
              </span>
              <span>{s.label}</span>
              {s.id === "reports" && openReports > 0 && <span className="admin-nav__badge">{openReports}</span>}
            </button>
          ))}
        </nav>
      </aside>

      <main className="admin-main">
        <header className="admin-main__head">
          <div>
            <h1 className="admin__title">С возвращением, {me.displayName}</h1>
            <p className="admin-main__sub">Что происходит в KISY сегодня</p>
          </div>
          <div className="admin-main__actions">
            <Button variant="secondary" onClick={() => setBroadcast(true)}>
              Рассылка
            </Button>
            <Button variant="secondary" onClick={() => setSection("updates")}>
              Новая версия
            </Button>
            <Button variant="secondary" onClick={showStatus}>
              Состояние систем
            </Button>
            <IconButton label="Назад к чатам" onClick={() => navigate("/")}>
              <Icon.Back />
            </IconButton>
          </div>
        </header>

        <div className="admin__content">
          {section === "overview" && <OverviewTab onNavigate={setSection} />}
          {section === "users" && <UsersTab />}
          {section === "invites" && <InvitesTab />}
          {section === "verification" && <VerificationTab />}
          {section === "communities" && <CommunitiesTab />}
          {section === "reports" && <ReportsTab />}
          {section === "updates" && <UpdatesTab />}
          {section === "deleted" && <DeletedTab />}
          {section === "audit" && <AuditTab />}
        </div>
      </main>

      <AnnouncementsModal open={broadcast} onClose={() => setBroadcast(false)} authorLevel={me.roleLevel ?? 1} />
    </div>
  );
}
