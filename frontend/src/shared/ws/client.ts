import type { ClientFrame, ServerEvent } from "./events";

const WS_BASE = import.meta.env.VITE_WS_BASE_URL ?? "/ws";

type Handler = (e: ServerEvent) => void;

// WsClient owns the single WebSocket connection for the app. It
// auto-reconnects with backoff and multiplexes events to subscribers,
// keeping all socket concerns out of the React tree (docs/spec/02:
// "WebSocket client isolated in service layer").
class WsClient {
  private socket: WebSocket | null = null;
  private handlers = new Set<Handler>();
  private reconnectDelay = 1000;
  private shouldRun = false;
  private queue: string[] = [];

  connect() {
    this.shouldRun = true;
    this.open();
  }

  private open() {
    if (!this.shouldRun) return;

    const url = this.resolveUrl();
    const socket = new WebSocket(url);
    this.socket = socket;

    socket.onopen = () => {
      this.reconnectDelay = 1000;
      for (const frame of this.queue) socket.send(frame);
      this.queue = [];
    };

    socket.onmessage = (ev) => {
      try {
        const parsed = JSON.parse(ev.data) as ServerEvent;
        for (const h of this.handlers) h(parsed);
      } catch {
        // ignore malformed frames
      }
    };

    socket.onclose = () => {
      this.socket = null;
      if (this.shouldRun) {
        setTimeout(() => this.open(), this.reconnectDelay);
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 15000);
      }
    };

    socket.onerror = () => socket.close();
  }

  private resolveUrl(): string {
    // WS_BASE may be an absolute ws(s):// URL or a same-origin path.
    if (WS_BASE.startsWith("ws")) return WS_BASE;
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${proto}//${window.location.host}${WS_BASE}`;
  }

  send(frame: ClientFrame) {
    const payload = JSON.stringify(frame);
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(payload);
    } else {
      this.queue.push(payload);
    }
  }

  subscribe(handler: Handler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  disconnect() {
    this.shouldRun = false;
    this.queue = [];
    this.socket?.close();
    this.socket = null;
  }
}

export const wsClient = new WsClient();
