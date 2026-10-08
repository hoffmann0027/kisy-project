import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// A user's suggestion (feedback, September 2026): in the profile settings the
// close button scrolled away with the page, so the settings could not be
// closed from the middle of it. The whole dialog was the scroller. Now the
// dialog is a column whose header never scrolls and whose body does — in
// every modal, not just the profile. jsdom has no layout engine, so the rule
// is checked where it lives: the stylesheet.

const css = readFileSync(join(__dirname, "ui.css"), "utf8");

/** The declarations of the first top-level rule for selector. */
function rule(selector: string): string {
  const at = css.indexOf("\n" + selector + " {");
  expect(at, selector).toBeGreaterThanOrEqual(0);
  const open = css.indexOf("{", at);
  return css.slice(open + 1, css.indexOf("}", open));
}

describe("modal layout", () => {
  it("scrolls the body, never the whole dialog", () => {
    const modal = rule(".ui-modal");
    expect(modal).toMatch(/display:\s*flex/);
    expect(modal).toMatch(/flex-direction:\s*column/);
    expect(modal).toMatch(/overflow:\s*hidden/);
    expect(modal).not.toMatch(/overflow:\s*auto/);

    const body = rule(".ui-modal__body");
    expect(body).toMatch(/overflow-y:\s*auto/);
    expect(body).toMatch(/min-height:\s*0/);
  });

  it("keeps the header and its close button in place", () => {
    expect(rule(".ui-modal__header")).toMatch(/flex:\s*none/);
  });
});
