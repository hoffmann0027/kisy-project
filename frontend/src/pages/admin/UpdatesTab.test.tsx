import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// "New Update": announce a version to everyone, and see who still runs an old build.

const api = vi.hoisted(() => ({
  releases: vi.fn(),
  sendRelease: vi.fn(async (input: { version: string }) => ({ release: { id: "x", version: input.version, recipientCount: 42 } })),
}));
vi.mock("@shared/api/endpoints", () => ({ adminApi: api }));

const { UpdatesTab, buildShare } = await import("./UpdatesTab");

function renderTab() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <UpdatesTab />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  api.releases.mockResolvedValue({
    releases: [],
    versions: [
      { version: "1.4.0", build: 200, users: 30 },
      { version: "1.3.0", build: 100, users: 12 },
    ],
  });
  vi.spyOn(window, "confirm").mockReturnValue(true);
});

describe("announcing a version", () => {
  it("sends the version, the notes and the link", async () => {
    renderTab();
    fireEvent.change(screen.getByLabelText("Версия"), { target: { value: " 1.5.0 " } });
    fireEvent.change(screen.getByLabelText("Что нового"), { target: { value: "Новая панель" } });
    fireEvent.change(screen.getByLabelText("Ссылка на скачивание (необязательно)"), { target: { value: "https://kisy.app/a.apk" } });
    fireEvent.click(screen.getByText("Отправить всем"));
    await waitFor(() => expect(api.sendRelease).toHaveBeenCalledWith({ version: "1.5.0", notes: "Новая панель", downloadUrl: "https://kisy.app/a.apk" }));
  });

  it("will not send a link that is not https", () => {
    renderTab();
    fireEvent.change(screen.getByLabelText("Версия"), { target: { value: "1.5.0" } });
    fireEvent.change(screen.getByLabelText("Что нового"), { target: { value: "Новая панель" } });
    fireEvent.change(screen.getByLabelText("Ссылка на скачивание (необязательно)"), { target: { value: "http://kisy.app/a.apk" } });
    expect((screen.getByText("Отправить всем").closest("button") as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText("Ссылка должна начинаться с https://")).toBeTruthy();
  });

  it("counts who runs the newest build and who an older one", async () => {
    renderTab();
    expect(await screen.findByText(/На новейшей сборке \(1\.4\.0\)/)).toBeTruthy();
    expect(buildShare([{ version: "a", build: 1, users: 5 }, { version: "b", build: 9, users: 2 }])).toEqual({
      latest: { version: "b", build: 9, users: 2 },
      onLatest: 2,
      older: 5,
    });
    expect(buildShare([]).latest).toBeNull();
  });
});
