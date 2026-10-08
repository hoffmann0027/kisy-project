import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// On a phone the month went lopsided: one day wide, three days a sliver, and
// Sunday cut off at the edge. A bare 1fr column will not shrink below its
// content, so a long event title ("Новая иконка приложения") widened its day.
// jsdom has no layout engine; the rule is checked where it lives.

const css = readFileSync(join(__dirname, "calendar.css"), "utf8");

function rule(selector: string): string {
  const at = css.indexOf("\n" + selector + " {");
  expect(at, selector).toBeGreaterThanOrEqual(0);
  const open = css.indexOf("{", at);
  return css.slice(open + 1, css.indexOf("}", open));
}

describe("calendar layout", () => {
  it("gives the seven days equal columns whatever is written in them", () => {
    for (const selector of [".cal__weekdays", ".cal__grid"]) {
      expect(rule(selector), selector).toMatch(/grid-template-columns:\s*repeat\(7,\s*minmax\(0,\s*1fr\)\)/);
    }
  });

  it("lets a day and its events shrink, cutting long titles instead", () => {
    expect(rule(".cal__cell")).toMatch(/min-width:\s*0/);
    expect(rule(".cal__chips")).toMatch(/min-width:\s*0/);
    expect(rule(".cal__chip")).toMatch(/text-overflow:\s*ellipsis/);
  });
});
