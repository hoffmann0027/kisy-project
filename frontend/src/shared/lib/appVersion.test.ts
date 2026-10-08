import { beforeEach, describe, expect, it, vi } from "vitest";

const info = vi.hoisted(() => ({ version: "1.4.0", build: "29312345" }));
vi.mock("@capacitor/app", () => ({ App: { getInfo: vi.fn(async () => info) } }));

const { formatAppVersion, initAppVersion } = await import("./appVersion");
const native = await import("./native");

function setNative(on: boolean) {
  Object.defineProperty(window, "Capacitor", {
    value: { isNativePlatform: () => on },
    configurable: true,
  });
}

beforeEach(() => native.setAppVersionHeader(null));

describe("the app's version header", () => {
  it("is what the server parses, and nothing it would refuse", () => {
    expect(formatAppVersion("1.4.0", "29312345")).toBe("1.4.0 (29312345)");
    expect(formatAppVersion("dev-80c3ae3", "1")).toBe("dev-80c3ae3 (1)");
    expect(formatAppVersion("1.4 beta", "1")).toBeNull();
    expect(formatAppVersion("1.4.0", "x")).toBeNull();
  });

  it("rides on every request of the packaged app once read", async () => {
    setNative(true);
    expect(native.nativeAuthHeaders()["X-Kisy-App-Version"]).toBeUndefined();
    await initAppVersion();
    expect(native.nativeAuthHeaders()["X-Kisy-App-Version"]).toBe("1.4.0 (29312345)");
  });

  it("is not sent by the browser", async () => {
    setNative(false);
    await initAppVersion();
    expect(native.nativeAuthHeaders()).toEqual({});
  });
});
