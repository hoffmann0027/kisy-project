import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@shared/api/client", () => ({ refreshSession: vi.fn(async () => {}) }));
vi.mock("@shared/lib/native", () => ({ isNative: () => false, apiOrigin: () => "", loadTokens: () => null }));

// A stand-in socket that the test opens and closes by hand.
class FakeSocket {
  static readonly OPEN = 1;
  static all: FakeSocket[] = [];
  readyState = 0;
  onopen: (() => void) | null = null;
  onclose: ((ev: { code: number }) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {
    FakeSocket.all.push(this);
  }
  send() {}
  close() {}
  serverOpens() {
    this.readyState = FakeSocket.OPEN;
    this.onopen?.();
  }
  serverCloses(code: number) {
    this.readyState = 3;
    this.onclose?.({ code });
  }
}

const { wsClient, CLOSE_REPLACED } = await import("./client");

describe("ws client after the server closed the socket", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    FakeSocket.all = [];
    vi.stubGlobal("WebSocket", FakeSocket);
  });
  afterEach(() => {
    wsClient.disconnect();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("reconnects on its own after an ordinary drop", () => {
    wsClient.connect();
    FakeSocket.all[0].serverOpens();
    FakeSocket.all[0].serverCloses(1006);
    vi.advanceTimersByTime(20_000);
    expect(FakeSocket.all).toHaveLength(2);
  });

  // Too many sockets for the account: the server closed the oldest. Dialling
  // back would close the newer one, and two tabs would evict each other forever.
  it("stays closed when replaced by a newer socket, until brought back", () => {
    wsClient.connect();
    FakeSocket.all[0].serverOpens();
    FakeSocket.all[0].serverCloses(CLOSE_REPLACED);
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.all).toHaveLength(1);

    wsClient.ensureConnected(); // the person returned to this tab
    expect(FakeSocket.all).toHaveLength(2);
  });
});
