import http from "k6/http";
import { sleep } from "k6";
import { Trend } from "k6/metrics";
import { API, authHeaders, expectOK, seedUsers } from "./lib/api.js";

// REST hot path: sending a message, reading a conversation and searching —
// the three calls that dominate real traffic. Acceptance thresholds come from
// the project's Definition of Done (API p95 < 200 ms).
//
//   k6 run tests/load/rest.js
//   BASE_URL=http://localhost:8081 VUS=20 DURATION=1m k6 run tests/load/rest.js

const VUS = Number(__ENV.VUS || 10);
const DURATION = __ENV.DURATION || "30s";
const USERS = Number(__ENV.USERS || 5);

const sendLatency = new Trend("kisy_message_send_ms", true);

export const options = {
  scenarios: {
    rest: {
      executor: "constant-vus",
      vus: VUS,
      duration: DURATION,
      gracefulStop: "10s",
    },
  },
  thresholds: {
    // The project's stated target for typical operations.
    "http_req_duration{expected_response:true}": ["p(95)<200"],
    http_req_failed: ["rate<0.01"],
    kisy_message_send_ms: ["p(95)<200"],
  },
};

export function setup() {
  return seedUsers(__ENV.CEO_USER || "ceo", __ENV.CEO_PASSWORD, USERS);
}

export default function (data) {
  const user = data.users[__VU % data.users.length];
  const headers = authHeaders(user.cookie);

  // 1. Send a message (the write path: auth → access check → insert → fan-out).
  const send = http.post(
    `${API}/messages`,
    JSON.stringify({
      chatType: "private",
      chatId: user.chatId,
      text: `load ${__VU}-${__ITER} ${Date.now()}`,
    }),
    { headers, tags: { name: "messages/send" } },
  );
  expectOK(send, "send");
  sendLatency.add(send.timings.duration);

  // 2. Read the conversation back (the paginated read path).
  const list = http.get(
    `${API}/messages?chatType=private&chatId=${user.chatId}&limit=50`,
    { headers, tags: { name: "messages/list" } },
  );
  expectOK(list, "list");

  // 3. Chat list — polled by every client on focus.
  expectOK(http.get(`${API}/chats`, { headers, tags: { name: "chats/list" } }), "chats");

  // 4. Search runs a full-text query; it is the heaviest read.
  expectOK(
    http.get(`${API}/search?q=load&limit=20`, { headers, tags: { name: "search" } }),
    "search",
  );

  sleep(1);
}
