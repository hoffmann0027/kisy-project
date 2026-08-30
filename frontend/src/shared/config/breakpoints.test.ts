import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// A phone in landscape is WIDE (≈915px) and SHORT (≈412px). A width-only
// breakpoint therefore hands it the desktop layout, which assumes a tall
// viewport: the columns run past the bottom edge, the rail's lower buttons
// leave the screen, and because the page body itself does not scroll there is
// no way to reach any of it — the app looks frozen.
//
// So every rule that makes the app phone-shaped has to fire on a short
// viewport too. jsdom has no layout engine and the project has no browser
// test runner, so this guards the invariant at the only place it can be
// checked automatically: the stylesheets themselves. It exists to catch the
// next `@media (max-width: 760px)` block someone adds without the height arm.
const PHONE_WIDTH = "max-width: 760px";
const SHORT_VIEWPORT = "max-height: 500px";

function cssFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return cssFiles(path);
    return entry.name.endsWith(".css") ? [path] : [];
  });
}

/** Every `@media …` prelude in a stylesheet, normalised to one line. */
function mediaQueries(css: string): string[] {
  return [...css.matchAll(/@media([^{]+)\{/g)].map((m) => m[1].replace(/\s+/g, " ").trim());
}

describe("phone breakpoints", () => {
  const files = cssFiles(join(__dirname, "..", ".."));

  it("finds the stylesheets", () => {
    expect(files.length).toBeGreaterThan(5);
  });

  it("cover landscape phones (short viewports), not just narrow ones", () => {
    const offenders: string[] = [];

    for (const file of files) {
      for (const query of mediaQueries(readFileSync(file, "utf8"))) {
        if (!query.includes(PHONE_WIDTH)) continue;
        if (!query.includes(SHORT_VIEWPORT)) {
          offenders.push(`${file.split(/[\\/]src[\\/]/)[1]}: @media ${query}`);
        }
      }
    }

    expect(
      offenders,
      `these blocks only fire on narrow screens, so a landscape phone falls back to the desktop layout:\n${offenders.join("\n")}`,
    ).toEqual([]);
  });
});
