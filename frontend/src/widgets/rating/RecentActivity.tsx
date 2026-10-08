import { t } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import { formatRelative } from "@shared/lib/format";
import type { RatingLedgerEntry } from "@shared/api/types";
import { Card, CardEmpty } from "./Card";

/** The latest money recorded, newest first. An entry may carry both an income and an expense. */
export function RecentActivity({ entries }: { entries: RatingLedgerEntry[] }) {
  return (
    <Card title={t("work.rating.activityTitle")}>
      {entries.length === 0 ? (
        <CardEmpty>{t("work.rating.activityEmpty")}</CardEmpty>
      ) : (
        <ul className="ract">
          {entries.map((e) => {
            const gain = e.incomeKopecks - e.expenseKopecks >= 0;
            return (
              <li key={e.id} className="ract__row">
                <span className={"ract__icon " + (gain ? "ract__icon--in" : "ract__icon--out")} aria-hidden="true">
                  {gain ? "+" : "−"}
                </span>
                <span className="ract__body">
                  <span className="ract__line">
                    {e.incomeKopecks > 0 && <b className="ract__sum--in">+{formatKopecks(e.incomeKopecks)}</b>}
                    {e.incomeKopecks > 0 && e.expenseKopecks > 0 && <span className="ract__project"> · </span>}
                    {e.expenseKopecks > 0 && <b className="ract__sum--out">−{formatKopecks(e.expenseKopecks)}</b>}
                    <span className="ract__project"> · {e.projectTitle}</span>
                  </span>
                  <span className="ract__meta">
                    {e.note ? `${e.note} · ` : ""}
                    {e.authorName}
                  </span>
                </span>
                <span className="ract__when">{formatRelative(e.createdAt)}</span>
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
}
