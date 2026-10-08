import { describe, expect, it } from "vitest";
import { axisTicks, periodLabel } from "./format";

describe("the chart's axis", () => {
  it("starts at zero when nothing was lost", () => {
    // recharts' own ticks for 0…18 420 € were −6 500, 0, 6 500, 13 000,
    // 19 500: a loss that never happened, drawn on the CEO's chart.
    expect(axisTicks([1842000, 712000, 1130000])).toEqual([0, 500000, 1000000, 1500000, 2000000]);
    expect(axisTicks([])).toEqual([0, 1]);
  });

  it("reaches below zero only for a real loss", () => {
    expect(axisTicks([120000, 184000, -64000])[0]).toBeLessThan(0);
  });
});

describe("period labels", () => {
  it("name quarters and years as they are", () => {
    expect(periodLabel("2026-Q3")).toBe("Q3 2026");
    expect(periodLabel("2026")).toBe("2026");
  });

  it("add the year to a month of another year", () => {
    const now = new Date(2026, 9, 8);
    expect(periodLabel("2026-10", now)).not.toMatch(/26/);
    expect(periodLabel("2025-10", now)).toMatch(/25$/);
  });
});
