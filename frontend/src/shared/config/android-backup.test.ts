import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// Audit A-22 and E-09. The WebView's storage holds the session tokens and the
// end-to-end encryption keys. allowBackup="true" sent them to Google's cloud
// backup together with the data they protect, and a restore onto another phone
// cloned the device id. CI builds only a debug APK and never reads the merged
// manifest, so a template default creeping back would go unnoticed.

// __dirname is frontend/src/shared/config, so three levels up is frontend/.
const MAIN = join(__dirname, "..", "..", "..", "android", "app", "src", "main");
const read = (path: string) => readFileSync(join(MAIN, path), "utf8");

describe.runIf(existsSync(MAIN))("android backup and file sharing", () => {
  it("keeps the app's data out of Android backups", () => {
    const manifest = read("AndroidManifest.xml");
    expect(manifest).toMatch(/android:allowBackup="false"/);
    // allowBackup alone does not stop device-to-device transfer on Android 12+.
    expect(manifest).toMatch(/android:dataExtractionRules="@xml\/data_extraction_rules"/);

    const rules = read("res/xml/data_extraction_rules.xml");
    for (const section of ["cloud-backup", "device-transfer"]) {
      const body = rules.split(`<${section}>`)[1]?.split(`</${section}>`)[0] ?? "";
      for (const domain of ["root", "file", "database", "sharedpref", "external"]) {
        expect(body, `${section} must exclude ${domain}`).toMatch(new RegExp(`<exclude domain="${domain}"`));
      }
      expect(body, `${section} must not include anything back`).not.toMatch(/<include/);
    }
  });

  it("shares only the camera folder, not the whole external storage", () => {
    const paths = read("res/xml/file_paths.xml");
    // The template's <external-path path="."> exposed every file on the
    // shared storage to whatever the provider was asked for.
    expect(paths).not.toMatch(/<external-path\b/);
    expect(paths).toMatch(/<external-files-path [^>]*path="Pictures\/"/);
  });
});
