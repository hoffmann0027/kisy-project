import { useMemo } from "react";
import { t } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import type { RatingProjectMoney } from "@shared/api/types";
import { profitShare } from "@entities/rating/model";
import { Card, CardEmpty } from "./Card";
import { Donut } from "./Donut";
import { SHARE } from "./palette";

/** Each project's share of the net profit. */
export function ProfitShare({ per }: { per: RatingProjectMoney[] }) {
  const share = useMemo(() => profitShare(per), [per]);
  const total = share.reduce((s, p) => s + p.value, 0);
  const slices = share.map((p, i) => ({ key: p.title + i, name: p.title, value: p.value, color: SHARE[i % SHARE.length] }));
  return (
    <Card title={t("work.rating.profitShareTitle")}>
      {share.length === 0 ? (
        <CardEmpty>{t("work.rating.noProfitData")}</CardEmpty>
      ) : (
        <Donut
          slices={slices}
          center={
            <>
              <strong>{formatKopecks(total)}</strong>
              <span>{t("work.rating.profit")}</span>
            </>
          }
          rows={slices.map((s) => ({ key: s.key, name: s.name, color: s.color, figure: `${Math.round((s.value / total) * 100)}%` }))}
          formatValue={formatKopecks}
        />
      )}
    </Card>
  );
}
