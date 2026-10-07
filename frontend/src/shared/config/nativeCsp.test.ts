// @vitest-environment node
//
// Audit A-45: inside the Capacitor shell there was no CSP at all — the bundle
// is served by the WebView, with no server to send the header.
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { nativeContentSecurityPolicy } from "./nativeCsp";

function directives(policy: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const part of policy.split(";")) {
    const [name, ...sources] = part.trim().split(/\s+/);
    if (name) out[name] = sources.join(" ");
  }
  return out;
}

describe("the native shell's CSP", () => {
  const d = directives(nativeContentSecurityPolicy("https://kisy.onrender.com"));

  it("reaches the API and its socket, and no other host", () => {
    expect(d["connect-src"]).toBe("'self' https://kisy.onrender.com wss://kisy.onrender.com");
    for (const sources of Object.values(d)) {
      expect(sources.split(" ")).not.toContain("ws:");
      expect(sources.split(" ")).not.toContain("wss:");
      expect(sources.split(" ")).not.toContain("https:");
      expect(sources).not.toContain("*");
    }
  });

  it("runs only the app's own scripts and the captcha", () => {
    expect(d["script-src"]).toBe("'self' 'wasm-unsafe-eval' https://challenges.cloudflare.com");
    expect(d["object-src"]).toBe("'none'");
    expect(d["base-uri"]).toBe("'self'");
  });

  it("lets authorized media through as blob: URLs", () => {
    expect(d["img-src"]).toContain("blob:");
    expect(d["media-src"]).toContain("blob:");
  });

  it("refuses an API origin that is not a bare https origin", () => {
    for (const bad of ["http://kisy.onrender.com", "https://kisy.onrender.com/api", "https://a.example?x=1", "javascript:alert(1)"]) {
      expect(() => nativeContentSecurityPolicy(bad)).toThrow();
    }
  });
});

describe("index.html", () => {
  // The policy, on the web and in the app, allows no inline script: one
  // there would simply never run (the theme script did not, for months).
  it("has no inline script", () => {
    const html = readFileSync(path.resolve(__dirname, "../../../index.html"), "utf8");
    for (const tag of html.match(/<script\b[^>]*>/g) ?? []) {
      expect(tag).toMatch(/\ssrc=/);
    }
  });
});
