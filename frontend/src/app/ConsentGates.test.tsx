import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ acceptConsent: vi.fn() }));
vi.mock("@shared/api/endpoints", () => ({ usersApi: api }));

const store = vi.hoisted(() => ({ setUser: vi.fn() }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (selector: (s: typeof store) => unknown) => selector(store),
}));

const { RequireLocalConsent, AccountConsentGate } = await import("./ConsentGates");
const { saveLocalConsent, hasLocalConsent } = await import("@shared/lib/consent");

function tickBothAndContinue() {
  for (const box of screen.getAllByRole("checkbox")) fireEvent.click(box);
  fireEvent.click(screen.getByRole("button", { name: "Продолжить" }));
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("before sign-in", () => {
  it("keeps the sign-in form behind the two boxes", () => {
    render(
      <RequireLocalConsent>
        <div>форма входа</div>
      </RequireLocalConsent>,
    );
    expect(screen.queryByText("форма входа")).not.toBeInTheDocument();

    tickBothAndContinue();
    expect(screen.getByText("форма входа")).toBeInTheDocument();
    expect(hasLocalConsent()).toBe(true);
  });

  it("does not ask again on this device once accepted", () => {
    saveLocalConsent();
    render(
      <RequireLocalConsent>
        <div>форма входа</div>
      </RequireLocalConsent>,
    );
    expect(screen.getByText("форма входа")).toBeInTheDocument();
  });
});

describe("after sign-in, an account without consent on record", () => {
  it("records the boxes ticked just before signing in, without asking twice", async () => {
    saveLocalConsent();
    const updated = { id: "u1", consentRequired: false };
    api.acceptConsent.mockResolvedValue({ user: updated });

    render(<AccountConsentGate />);

    await waitFor(() => expect(store.setUser).toHaveBeenCalledWith(updated));
    expect(api.acceptConsent).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });

  it("asks in person when nothing was ticked on this device", async () => {
    render(<AccountConsentGate />);
    expect(screen.getAllByRole("checkbox")).toHaveLength(2);
    expect(api.acceptConsent).not.toHaveBeenCalled();

    api.acceptConsent.mockResolvedValue({ user: { id: "u1" } });
    tickBothAndContinue();
    await waitFor(() => expect(api.acceptConsent).toHaveBeenCalledTimes(1));
  });

  it("falls back to asking when the silent record is refused (texts changed)", async () => {
    saveLocalConsent();
    api.acceptConsent.mockRejectedValue(new Error("CONSENT_REQUIRED"));

    render(<AccountConsentGate />);

    await waitFor(() => expect(screen.getAllByRole("checkbox")).toHaveLength(2));
    expect(store.setUser).not.toHaveBeenCalled();
    expect(hasLocalConsent()).toBe(false);
  });
});
