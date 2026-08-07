import http from "k6/http";
import { check, fail } from "k6";

// Shared helpers for the k6 scenarios. Everything talks to the same public
// surface the SPA uses (cookie auth, /api/v1), so the numbers reflect what a
// real client experiences rather than an internal shortcut.

export const BASE_URL = __ENV.BASE_URL || "http://localhost:8081";
export const API = `${BASE_URL}/api/v1`;

// The seeded load users share one password; it must satisfy the production
// policy (12-128 chars, a letter and a digit).
export const LOAD_USER_PASSWORD = "LoadTest-2026-pw1";

const JSON_HEADERS = { "Content-Type": "application/json" };

/** Log in and return the Set-Cookie header value to reuse as a session. */
export function login(username, password) {
  const res = http.post(
    `${API}/auth/login`,
    JSON.stringify({ username, password }),
    { headers: JSON_HEADERS, tags: { name: "auth/login" } },
  );
  if (res.status !== 200) {
    fail(`login failed for ${username}: ${res.status} ${res.body}`);
  }
  return cookieHeader(res);
}

/** Collapse a response's cookies into a single Cookie request header. */
function cookieHeader(res) {
  const jar = res.cookies || {};
  return Object.keys(jar)
    .map((name) => `${name}=${jar[name][0].value}`)
    .join("; ");
}

export function authHeaders(cookie) {
  // Deliberately no Origin header: k6 is not a browser, and both the CSRF
  // middleware and the WebSocket origin checker pass requests that carry no
  // Origin (a non-browser client cannot be driven into a CSRF/CSWSH attack).
  // Sending one would in fact fail here — the edge proxy rewrites Host without
  // the port, so an Origin like http://host:8081 never matches.
  return { ...JSON_HEADERS, Cookie: cookie };
}

/**
 * Seed `count` load users (idempotent-ish: a username already taken is reused)
 * and open a private chat from each to the CEO. Runs once in k6 setup().
 */
export function seedUsers(ceoUser, ceoPassword, count) {
  const ceoCookie = login(ceoUser, ceoPassword);
  const users = [];

  for (let i = 0; i < count; i++) {
    const username = `load_user_${i}`;
    let session = tryLogin(username, LOAD_USER_PASSWORD);

    if (!session) {
      const invite = http.post(`${API}/invites`, null, {
        headers: authHeaders(ceoCookie),
        tags: { name: "invites/create" },
      });
      if (invite.status !== 200 && invite.status !== 201) {
        fail(`invite creation failed: ${invite.status} ${invite.body}`);
      }
      const token = invite.json("data.invitation.token") || invite.json("data.token");
      if (!token) fail(`no invite token in response: ${invite.body}`);

      const reg = http.post(
        `${API}/auth/register`,
        JSON.stringify({ inviteToken: token, username, password: LOAD_USER_PASSWORD }),
        { headers: JSON_HEADERS, tags: { name: "auth/register" } },
      );
      if (reg.status !== 200 && reg.status !== 201) {
        fail(`register failed for ${username}: ${reg.status} ${reg.body}`);
      }
      // Register already sets the auth cookies and returns the user, so reuse
      // them instead of spending another /auth/login — those endpoints are
      // rate-limited (register 5/min, login 10/min per IP), which is what
      // caps how many users one setup() can seed.
      session = { cookie: cookieHeader(reg), id: reg.json("data.user.id") };
      if (!session.cookie || !session.id) {
        fail(`register returned no session for ${username}: ${reg.body}`);
      }
    }

    users.push({
      username,
      id: session.id,
      cookie: session.cookie,
      chatId: openChatWithCEO(ceoCookie, session.id),
    });
  }

  return { users, ceoCookie };
}

function tryLogin(username, password) {
  const res = http.post(
    `${API}/auth/login`,
    JSON.stringify({ username, password }),
    { headers: JSON_HEADERS, tags: { name: "auth/login" } },
  );
  if (res.status !== 200) return null;
  return { cookie: cookieHeader(res), id: res.json("data.user.id") };
}

/**
 * Open (or reuse) the private chat between the CEO and a load user.
 *
 * The CEO has to be the initiator: access.CanInitiateChat only lets equal-or-
 * stronger clearance reach down, and a freshly invited user is the weakest
 * level. Once the chat exists both parties may post, which is what the
 * scenarios exercise.
 */
function openChatWithCEO(ceoCookie, userID) {
  const chat = http.post(`${API}/chats`, JSON.stringify({ userId: userID }), {
    headers: authHeaders(ceoCookie),
    tags: { name: "chats/open" },
  });
  const chatId = chat.json("data.chat.id");
  if (!chatId) fail(`cannot open chat: ${chat.status} ${chat.body}`);
  return chatId;
}

/** Assert a response succeeded, tagging failures for the summary. */
export function expectOK(res, label) {
  return check(res, { [`${label} 2xx`]: (r) => r.status >= 200 && r.status < 300 });
}
