import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

// The Android resource XML is compiled by aapt2, not by anything that runs
// in CI before the APK step — a malformed file therefore surfaces as a
// gradle failure minutes into the build, with no local signal. This caught a
// real one: a comment containing "--" (illegal per XML 1.0 §2.5, and written
// naturally when quoting a CSS custom property like --bg) failed the whole
// Android build while every other check stayed green.
// __dirname is frontend/src/shared/config, so three levels up is frontend/.
const RES = join(__dirname, "..", "..", "..", "android", "app", "src", "main", "res");

function xmlFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return xmlFiles(path);
    return entry.name.endsWith(".xml") ? [path] : [];
  });
}

describe.runIf(existsSync(RES))("android resource xml", () => {
  it("finds the resources", () => {
    expect(xmlFiles(RES).length).toBeGreaterThan(0);
  });

  it("has no double hyphen inside a comment", () => {
    const offenders = xmlFiles(RES).filter((file) =>
      [...readFileSync(file, "utf8").matchAll(/<!--([\s\S]*?)-->/g)].some((m) => m[1].includes("--")),
    );

    expect(
      offenders.map((f) => relative(RES, f)),
      'XML forbids "--" inside comments; aapt2 rejects the file and the APK build fails',
    ).toEqual([]);
  });
});
