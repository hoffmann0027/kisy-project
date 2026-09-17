import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@shared/api/envelope";

// Only the server can end a session.
//
// The store used to treat every failed /users/me as "not signed in". On a
// phone that is exactly wrong: woken by a call, radio still attaching, the
// first request goes nowhere — and the app answered a perfectly valid session
// with the password screen. Both people on the call saw it, and signing in by
// hand worked, because by then the network was up.

const api = vi.hoisted(() => ({ me: vi.fn() }));
const authApi = vi.hoisted(() => ({ login: vi.fn(), register: vi.fn(), logout: vi.fn() }));
vi.mock("@shared/api/endpoints", () => ({
  usersApi: api,
  authApi,
}));
vi.mock("@shared/lib/nativePush", () => ({ forgetNativePushDevice: vi.fn(async () => {}) }));

const { useAuthStore, setPageReloaderForTests } = await import("./auth");
const reload = vi.fn();

const user = { id: "u1", username: "hamza", displayName: "Hamza", roleLevel: 1 };

beforeEach(() => {
  vi.clearAllMocks();
  setPageReloaderForTests(reload);
  useAuthStore.setState({ user: null, status: "loading" });
});

describe("restoring the session on start", () => {
  it("signs in when the server answers", async () => {
    api.me.mockResolvedValue({ user });
    await useAuthStore.getState().bootstrap();
    expect(useAuthStore.getState().status).toBe("authenticated");
  });

  // The retries are spread over several seconds of real waiting, which is the
  // point of them; the clock is faked so the tests do not wait it out.
  async function bootstrapPastTheRetries() {
    vi.useFakeTimers();
    try {
      const done = useAuthStore.getState().bootstrap();
      await vi.advanceTimersByTimeAsync(30_000);
      await done;
    } finally {
      vi.useRealTimers();
    }
  }

  it("keeps trying while the network is still coming up", async () => {
    api.me
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue({ user });

    await bootstrapPastTheRetries();

    expect(api.me).toHaveBeenCalledTimes(3);
    expect(useAuthStore.getState().status).toBe("authenticated");
  });

  it("reports no connection rather than a sign-out when it never gets through", async () => {
    api.me.mockRejectedValue(new TypeError("Failed to fetch"));

    await bootstrapPastTheRetries();

    // Not "anonymous": that is what sent the user to the password screen.
    expect(useAuthStore.getState().status).toBe("offline");
  });

  it("signs out only when the server actually refuses the session", async () => {
    api.me.mockRejectedValue(new ApiError("AUTH_EXPIRED", "expired", "req-1", 401));

    await useAuthStore.getState().bootstrap();

    expect(useAuthStore.getState().status).toBe("anonymous");
    // One answer is enough when it is the server's: no pointless retries.
    expect(api.me).toHaveBeenCalledTimes(1);
  });
});

// Audit A-11: signing out cleared `user` and nothing else. The next account
// to sign in on the same page inherited the previous one's E2EE session, chat
// list, decrypted messages and drafts — all module state with no owner.
describe("the page belongs to one account", () => {
  const other = { ...user, id: "u2", username: "other" };

  it("reloads after signing out, so nothing of the account stays in memory", async () => {
    setPageReloaderForTests(reload, user.id);
    authApi.logout.mockResolvedValue({ loggedOut: true });
    await useAuthStore.getState().logout();
    expect(useAuthStore.getState().user).toBeNull();
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("reloads even when the server could not be told about the sign-out", async () => {
    setPageReloaderForTests(reload, user.id);
    authApi.logout.mockRejectedValue(new TypeError("Failed to fetch"));
    await useAuthStore.getState().logout().catch(() => {});
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("reloads before another account signing in on this page sees anything", async () => {
    // The session ended without logout() (revoked elsewhere, expired): the
    // page still holds the first account's state.
    setPageReloaderForTests(reload, user.id);
    authApi.login.mockResolvedValue({ user: other });
    await useAuthStore.getState().login("other", "password");
    expect(reload).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().user).toBeNull();
  });

  it("does not reload when the same account signs back in", async () => {
    setPageReloaderForTests(reload, user.id);
    authApi.login.mockResolvedValue({ user });
    await useAuthStore.getState().login("hamza", "password");
    expect(reload).not.toHaveBeenCalled();
    expect(useAuthStore.getState().user?.id).toBe(user.id);
  });

  it("does not reload for the first sign-in of a fresh page", async () => {
    authApi.login.mockResolvedValue({ user });
    await useAuthStore.getState().login("hamza", "password");
    expect(reload).not.toHaveBeenCalled();
  });
});
