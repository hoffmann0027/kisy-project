import { useState } from "react";
import "./rating.css";
import { Rail } from "@widgets/rail/Rail";
import { Button, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { t } from "@shared/i18n";
import { useAuthStore } from "@shared/store/auth";
import { ProfileModal } from "@features/profile/ProfileModal";
import { useRatingAnalytics, useRatingBoard, useRatingMutations } from "@entities/rating/queries";
import { RatingKpis } from "@widgets/rating/RatingKpis";
import { FinanceChart } from "@widgets/rating/FinanceChart";
import { StatusDonut } from "@widgets/rating/StatusDonut";
import { Workload } from "@widgets/rating/Workload";
import { RecentActivity } from "@widgets/rating/RecentActivity";
import { TopProjects } from "@widgets/rating/TopProjects";
import { ProfitShare } from "@widgets/rating/ProfitShare";
import { ProjectsTable } from "@widgets/rating/ProjectsTable";
import { NewProjectDialog } from "@widgets/rating/NewProjectDialog";
import { canCreateProject } from "@entities/rating/model";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

// The project dashboard (October 2026 redesign): the headline figures, the
// money over time, where every project stands and who is on it, and the
// projects themselves — each row opening into its tasks and controls. The
// kanban it replaces is folded into the rows.
export function RatingPage() {
  const { data: board, isPending } = useRatingBoard();
  const { data: analytics } = useRatingAnalytics();
  const m = useRatingMutations();
  const user = useAuthStore((s) => s.user);
  // Levels 1–4 run projects of their own; the CEO runs them all.
  const creator = canCreateProject({ id: user?.id ?? "", roleLevel: user?.roleLevel ?? null });
  const [profile, setProfile] = useState(false);
  const [creating, setCreating] = useState(false);

  const projects = board?.projects ?? [];
  const monthly = analytics?.monthly ?? [];
  const perProject = analytics?.perProject ?? [];

  return (
    <div className="rating-shell">
      <Rail onProfile={() => setProfile(true)} />

      <main className="rating">
        <div className="rating__scroll">
          <header className="rating__top">
            <div className="rating__titles">
              <h1 className="rating__heading">{t("work.rating.heading")}</h1>
              <p className="rating__sub">{t("work.rating.subtitle")}</p>
            </div>
            <div className="rating__actions">
              {creator && (
                <Button variant="primary" onClick={() => setCreating(true)}>
                  <Icon.Plus size={18} /> {t("work.rating.newProject")}
                </Button>
              )}
              <a className="ui-btn ui-btn--secondary rating__export" href={`${API_BASE}/rating/export.csv`}>
                <Icon.Archive size={18} /> {t("work.rating.exportCsv")}
              </a>
            </div>
          </header>

          {isPending || !board ? (
            <div className="rating__loading">
              <Spinner size={28} />
            </div>
          ) : (
            <div className="rating-grid">
              <div className="rating-grid__main">
                <RatingKpis projects={projects} monthly={monthly} />
                <FinanceChart monthly={monthly} />
                <ProjectsTable projects={projects} m={m} />
                <div className="rating-grid__pair">
                  <TopProjects per={perProject} />
                  <ProfitShare per={perProject} />
                </div>
              </div>
              <aside className="rating-grid__side">
                <StatusDonut projects={projects} />
                <Workload projects={projects} />
                <RecentActivity entries={analytics?.recent ?? []} />
              </aside>
            </div>
          )}
        </div>
      </main>

      <ProfileModal open={profile} onClose={() => setProfile(false)} />
      {creator && <NewProjectDialog m={m} open={creating} onClose={() => setCreating(false)} />}
    </div>
  );
}
