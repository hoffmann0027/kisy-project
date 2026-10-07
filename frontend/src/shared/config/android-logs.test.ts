import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

// Audit A-49: the call receiver wrote the notification's extras — the
// caller's name among them — to logcat, and the incoming-call log line named
// the caller too. Logcat is readable over a cable and travels in every bug
// report; it gets ids and states, never who is calling or what was said.

// __dirname is frontend/src/shared/config, so three levels up is frontend/.
const JAVA = join(__dirname, "..", "..", "..", "android", "app", "src", "main", "java");

function javaFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    return statSync(path).isDirectory() ? javaFiles(path) : path.endsWith(".java") ? [path] : [];
  });
}

describe.runIf(existsSync(JAVA))("android logcat", () => {
  it("never carries a name, a message or an intent's extras", () => {
    const calls: string[] = [];
    for (const file of javaFiles(JAVA)) {
      // A Log call may wrap onto the next lines; read it up to its ");".
      for (const m of readFileSync(file, "utf8").matchAll(/\bLog\.[vdiwe]\(([\s\S]*?)\);/g)) {
        calls.push(m[1]);
      }
    }
    expect(calls.length).toBeGreaterThan(0);
    for (const args of calls) {
      expect(args, args).not.toMatch(/getExtras\(\)|\bname\b|callerName|displayName|\btitle\b|\bbody\b|\btext\b/i);
    }
  });
});
