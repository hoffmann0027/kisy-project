import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { apiClient } from "./client";

// The app used to sign the user out roughly every quarter of an hour: the
// access token expires after 15 minutes, and although /auth/refresh existed,
// nothing ever called it — the first 401 propagated straight to the auth
// store, which fell back to "anonymous" and showed the password screen.

const ok = (data: unknown) =>
  new Response(JSON.stringify({ success: true, data }), { status: 200 });
const unauthorized = () =>
  new Response(JSON.stringify({ success: false, error: { code: "AUTH_EXPIRED", message: "expired" } }), {
    status: 401,
  });

describe("api client session renewal", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("refreshes once and replays the request after a 401", async () => {
    fetchMock
      .mockResolvedValueOnce(unauthorized()) // GET /users/me
      .mockResolvedValueOnce(ok({})) //         POST /auth/refresh
      .mockResolvedValueOnce(ok({ user: { id: "u1" } })); // replay

    await expect(apiClient.get("/users/me")).resolves.toEqual({ user: { id: "u1" } });

    const paths = fetchMock.mock.calls.map((c) => String(c[0]));
    expect(paths[1]).toContain("/auth/refresh");
    expect(paths[2]).toContain("/users/me");
  });

  it("gives up after one failed refresh instead of looping", async () => {
    fetchMock
      .mockResolvedValueOnce(unauthorized())
      .mockResolvedValueOnce(new Response("{}", { status: 401 })); // refresh rejected

    await expect(apiClient.get("/users/me")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("does not try to refresh a failed sign-in", async () => {
    fetchMock.mockResolvedValueOnce(unauthorized());

    await expect(apiClient.post("/auth/login", { username: "x", password: "y" })).rejects.toMatchObject({
      status: 401,
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("shares one refresh between concurrent 401s", async () => {
    // Refresh rotates the token, so a screen firing several queries at once
    // must not spend several refresh tokens — the reuse detector would treat
    // that as a stolen token and revoke the whole session.
    fetchMock
      .mockResolvedValueOnce(unauthorized())
      .mockResolvedValueOnce(unauthorized())
      .mockResolvedValueOnce(unauthorized())
      .mockImplementation((url: string) =>
        Promise.resolve(String(url).includes("/auth/refresh") ? ok({}) : ok({ replayed: true })),
      );

    await Promise.all([apiClient.get("/a"), apiClient.get("/b"), apiClient.get("/c")]);

    const refreshCalls = fetchMock.mock.calls.filter((c) => String(c[0]).includes("/auth/refresh"));
    expect(refreshCalls).toHaveLength(1);
  });
});
