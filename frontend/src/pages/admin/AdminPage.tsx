import { useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import "./admin.css";
import { Button, IconButton, Logo } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { adminApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { t, type Key } from "@shared/i18n";
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

const SECTIONS: { id: AdminSection; label: Key; icon: ReactNode }[] = [
  { id: "overview", label: "admin.nav.overview", icon: <Icon.Grid /> },
  { id: "users", label: "admin.nav.users", icon: <Icon.Users /> },
  { id: "invites", label: "admin.nav.invites", icon: <Icon.Plus /> },
  { id: "verification", label: "admin.nav.verification", icon: <Icon.Check /> },
  { id: "communities", label: "admin.nav.communities", icon: <Icon.Community /> },
  { id: "reports", label: "admin.nav.reports", icon: <Icon.Flag /> },
  { id: "updates", label: "admin.nav.updates", icon: <Icon.Send /> },
  { id: "deleted", label: "admin.nav.deleted", icon: <Icon.Trash /> },
  { id: "audit", label: "admin.nav.audit", icon: <Icon.Shield /> },
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
      <aside className="admin-nav" aria-label={t("admin.nav.ariaLabel")}>
        <div className="admin-nav__brand">
          <Logo size={36} />
          <div>
            <div className="admin-nav__name">KISY</div>
            <div className="admin-nav__role">{t("admin.nav.role")}</div>
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
              <span>{t(s.label)}</span>
              {s.id === "reports" && openReports > 0 && <span className="admin-nav__badge">{openReports}</span>}
            </button>
          ))}
        </nav>
      </aside>

      <main className="admin-main">
        <header className="admin-main__head">
          <div>
            <h1 className="admin__title">{t("admin.header.welcome", { name: me.displayName })}</h1>
            <p className="admin-main__sub">{t("admin.header.subtitle")}</p>
          </div>
          <div className="admin-main__actions">
            <Button variant="secondary" onClick={() => setBroadcast(true)}>
              {t("admin.header.broadcast")}
            </Button>
            <Button variant="secondary" onClick={() => setSection("updates")}>
              {t("admin.header.newVersion")}
            </Button>
            <Button variant="secondary" onClick={showStatus}>
              {t("admin.header.systemStatus")}
            </Button>
            <IconButton label={t("admin.header.backToChats")} onClick={() => navigate("/")}>
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
