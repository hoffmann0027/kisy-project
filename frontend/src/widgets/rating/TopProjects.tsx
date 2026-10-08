import { useMemo, useState } from "react";
import { t, type Key } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import { colorFromString } from "@shared/lib/format";
import type { RatingProjectMoney } from "@shared/api/types";
import { topProjects, type MoneyKey } from "@entities/rating/model";
import { Card, CardEmpty, CardTabs } from "./Card";

const KEYS: { value: MoneyKey; key: Key }[] = [
  { value: "profit", key: "work.rating.profit" },
  { value: "income", key: "work.rating.income" },
  { value: "expense", key: "work.rating.expense" },
];

/** The projects at the top, by profit, income or expense. */
export function TopProjects({ per }: { per: RatingProjectMoney[] }) {
  const [key, setKey] = useState<MoneyKey>("profit");
  const rows = useMemo(() => topProjects(per, key), [per, key]);
  const field = key === "profit" ? "profitKopecks" : key === "income" ? "incomeKopecks" : "expenseKopecks";
  return (
    <Card
      title={t("work.rating.topTitle")}
      aside={<CardTabs value={key} options={KEYS.map((k) => ({ value: k.value, label: t(k.key) }))} onChange={setKey} label={t("work.rating.topTitle")} />}
    >
      {rows.length === 0 ? (
        <CardEmpty>{t("work.rating.topEmpty")}</CardEmpty>
      ) : (
        <ol className="rtop">
          {rows.map((p, i) => (
            <li key={p.projectId} className="rtop__row">
              <span className={"rtop__rank" + (i < 3 ? ` rtop__rank--${i + 1}` : "")}>{i + 1}</span>
              <span className="rlogo" style={{ background: colorFromString(p.projectId) }} aria-hidden="true">
                {p.title.trim().charAt(0).toUpperCase()}
              </span>
              <span className="rtop__title">{p.title}</span>
              <span className={"rtop__sum" + (key === "expense" ? " rtop__sum--out" : p[field] < 0 ? " rtop__sum--out" : " rtop__sum--in")}>{formatKopecks(p[field])}</span>
            </li>
          ))}
        </ol>
      )}
    </Card>
  );
}
