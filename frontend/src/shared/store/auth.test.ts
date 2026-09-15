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
vi.mock("@shared/api/endpoints", () => ({
  usersApi: api,
  authApi: { login: vi.fn(), register: vi.fn(), logout: vi.fn() },
}));
vi.mock("@shared/lib/nativePush", () => ({ forgetNativePushDevice: vi.fn(async () => {}) }));

const { useAuthStore } = await import("./auth");

const user = { id: "u1", username: "hamza", displayName: "Hamza", roleLevel: 1 };

beforeEach(() => {
  vi.clearAllMocks();
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
