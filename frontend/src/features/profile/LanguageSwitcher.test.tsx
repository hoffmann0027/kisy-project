import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LanguageSwitcher } from "./LanguageSwitcher";

const reload = vi.fn();
Object.defineProperty(window, "location", { value: { ...window.location, reload }, writable: true });

afterEach(() => localStorage.clear());

describe("the language switcher", () => {
  it("offers every language by its own name, and says Language in English too", () => {
    render(<LanguageSwitcher />);
    expect(screen.getByText("Язык · Language")).toBeTruthy();
    expect(screen.getByRole("option", { name: "English" })).toBeTruthy();
    expect(screen.getByRole("option", { name: "Русский" })).toBeTruthy();
  });

  it("remembers the choice and restarts in it", () => {
    render(<LanguageSwitcher />);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "en" } });
    expect(localStorage.getItem("kisy-lang")).toBe("en");
    expect(reload).toHaveBeenCalled();
  });
});
