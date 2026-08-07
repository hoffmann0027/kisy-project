import ws from "k6/ws";
import { check } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import { BASE_URL, seedUsers } from "./lib/api.js";

// WebSocket capacity: how many concurrent sockets one instance holds while
// still delivering events promptly. Each VU keeps a socket open, subscribes to
// presence, emits typing and sends messages, measuring the round trip from
// "message.send" to the "message.created" the server fans back out.
//
//   VUS=100 DURATION=1m k6 run tests/load/ws.js

const VUS = Number(__ENV.VUS || 25);
const DURATION = __ENV.DURATION || "30s";
const USERS = Number(__ENV.USERS || 5);
// How long each VU holds its socket before reconnecting (a real client keeps
// it open for the whole session; cycling here exercises connect/teardown too).
const HOLD_SECONDS = Number(__ENV.HOLD_SECONDS || 20);

const connected = new Counter("kisy_ws_connections");
const deliveryLatency = new Trend("kisy_ws_delivery_ms", true);
const deliverySuccess = new Rate("kisy_ws_delivery_success");

export const options = {
  scenarios: {
    sockets: {
      executor: "constant-vus",
      vus: VUS,
      duration: DURATION,
      gracefulStop: "15s",
    },
  },
  thresholds: {
    // A fan-out that takes longer than this is felt as lag in the chat UI.
    kisy_ws_delivery_ms: ["p(95)<500"],
    kisy_ws_delivery_success: ["rate>0.95"],
    ws_connecting: ["p(95)<1000"],
  },
};

export function setup() {
  return seedUsers(__ENV.CEO_USER || "ceo", __ENV.CEO_PASSWORD, USERS);
}

export default function (data) {
  const user = data.users[__VU % data.users.length];
  const url = `${BASE_URL.replace(/^http/, "ws")}/ws`;
  const params = { headers: { Cookie: user.cookie } };

  const sent = {}; // marker text -> send timestamp

  const res = ws.connect(url, params, function (socket) {
    socket.on("open", function () {
      connected.add(1);

      // Watch the peers a real client would care about.
      socket.send(
        JSON.stringify({
          type: "presence.subscribe",
          data: { userIds: data.users.map((u) => u.id).filter(Boolean) },
        }),
      );

      // Typing indicators are the chattiest frames in normal use.
      socket.setInterval(function () {
        socket.send(
          JSON.stringify({
            type: "typing.start",
            data: { chatType: "private", chatId: user.chatId },
          }),
        );
      }, 3000);

      // Send a message every 5s and time the fan-out back to us.
      socket.setInterval(function () {
        const marker = `ws ${__VU}-${Date.now()}`;
        sent[marker] = Date.now();
        socket.send(
          JSON.stringify({
            type: "message.send",
            data: { chatType: "private", chatId: user.chatId, text: marker },
          }),
        );
      }, 5000);

      socket.setTimeout(function () {
        socket.close();
      }, HOLD_SECONDS * 1000);
    });

    socket.on("message", function (raw) {
      let frame;
      try {
        frame = JSON.parse(raw);
      } catch (e) {
        return;
      }
      // Outbound frames are {event, data}; message.created carries the
      // message DTO directly, so the body is data.text.
      if (frame.event !== "message.created") return;

      const text = frame.data && frame.data.text;
      if (text && sent[text] !== undefined) {
        deliveryLatency.add(Date.now() - sent[text]);
        deliverySuccess.add(true);
        delete sent[text];
      }
    });

    socket.on("error", function (e) {
      if (e && e.error() !== "websocket: close sent") {
        deliverySuccess.add(false);
      }
    });
  });

  check(res, { "ws handshake 101": (r) => r && r.status === 101 });

  // Anything still unacknowledged when the socket closed never came back.
  for (const pending in sent) {
    deliverySuccess.add(false);
  }
}
