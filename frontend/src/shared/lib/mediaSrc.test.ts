import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// isNative()/apiOrigin()/nativeAuthHeaders() are the shell-detection layer;
// here they are scriptable so one test can be a phone and the next a browser.
const shell = vi.hoisted(() => ({ native: false }));

vi.mock("@shared/lib/native", () => ({
  isNative: () => shell.native,
  apiOrigin: () => (shell.native ? "https://api.kisy.example" : ""),
  nativeAuthHeaders: () => (shell.native ? { Authorization: "Bearer token-1", "X-Kisy-Client": "native" } : {}),
}));

vi.mock("@shared/api/client", () => ({ refreshSession: async () => "rejected" }));

import { downloadAsset, handleDownloadClick } from "./mediaSrc";

describe("handleDownloadClick (audit D-16)", () => {
  let clicked: { href: string; download: string }[];

  // Captured once: taking it inside beforeEach would, on the second test, pick
  // up the previous test's spy and recurse.
  const realCreateElement = document.createElement.bind(document);

  beforeEach(() => {
    shell.native = false;
    clicked = [];
    // Anchors created by downloadAsset record their click instead of asking
    // jsdom to navigate.
    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      const el = realCreateElement(tag) as HTMLElement;
      if (tag === "a") {
        el.click = () => {
          const a = el as HTMLAnchorElement;
          clicked.push({ href: a.getAttribute("href") ?? "", download: a.download });
        };
      }
      return el;
    });
    vi.stubGlobal("URL", { ...URL, createObjectURL: () => "blob:kisy/1", revokeObjectURL: () => {} });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("leaves the browser alone: the anchor keeps its own href", () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
    let prevented = false;

    handleDownloadClick({ preventDefault: () => (prevented = true) }, "/api/v1/attachments/a1", "отчёт.pdf");

    expect(prevented).toBe(false);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("on native fetches the bytes with the session token and saves them as a blob", async () => {
    shell.native = true;
    const fetchSpy = vi.fn(async () => new Response(new Blob(["bytes"]), { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);

    await downloadAsset("/api/v1/attachments/a1", "отчёт.pdf");

    // A bare href would have gone to the WebView's own origin with no
    // credentials at all — the tap did nothing.
    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe("https://api.kisy.example/api/v1/attachments/a1");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer token-1");
    expect(clicked).toEqual([{ href: "blob:kisy/1", download: "отчёт.pdf" }]);
  });

  it("on native takes the click over from the anchor", () => {
    shell.native = true;
    vi.stubGlobal("fetch", vi.fn(async () => new Response(new Blob(["bytes"]), { status: 200 })));
    let prevented = false;

    handleDownloadClick({ preventDefault: () => (prevented = true) }, "/api/v1/attachments/a1", "отчёт.pdf");

    expect(prevented).toBe(true);
  });

  it("does not intercept a URL the API does not serve", () => {
    shell.native = true;
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
    let prevented = false;

    handleDownloadClick({ preventDefault: () => (prevented = true) }, "https://example.com/file.pdf");

    expect(prevented).toBe(false);
    expect(fetchSpy).not.toHaveBeenCalled();
  });
});
