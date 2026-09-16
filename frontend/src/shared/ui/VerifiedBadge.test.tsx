import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { VerifiedBadge, VerifiedName } from "./VerifiedBadge";

describe("the verification mark", () => {
  it("draws the seal with an inner rim at 20px and without it at 14px", () => {
    const { container, rerender } = render(<VerifiedBadge size={20} />);
    expect(container.querySelectorAll("path")).toHaveLength(3);
    rerender(<VerifiedBadge size={14} />);
    expect(container.querySelectorAll("path")).toHaveLength(2);
    expect(container.querySelector("svg")?.getAttribute("width")).toBe("14");
  });

  it("is named for screen readers, as a person or as a community", () => {
    render(
      <>
        <VerifiedBadge subject="user" />
        <VerifiedBadge subject="group" />
      </>,
    );
    expect(screen.getByRole("img", { name: "Подтверждённый аккаунт" })).toBeTruthy();
    expect(screen.getByRole("img", { name: "Подтверждённое сообщество" })).toBeTruthy();
  });

  it("appears beside a name only when verified", () => {
    const { rerender } = render(<VerifiedName name="Анна Смирнова" verified={false} />);
    expect(screen.getByText("Анна Смирнова")).toBeTruthy();
    expect(screen.queryByRole("img")).toBeNull();
    rerender(<VerifiedName name="Анна Смирнова" verified />);
    expect(screen.getByRole("img", { name: "Подтверждённый аккаунт" })).toBeTruthy();
  });

  it("takes its colours from the theme, never from a literal", () => {
    // Seven themes, light and dark: a hard-coded colour would be wrong in most.
    const source = readFileSync(join(__dirname, "VerifiedBadge.tsx"), "utf8");
    expect(source).not.toMatch(/#[0-9a-fA-F]{3,8}\b|rgba?\(/);
    expect(source).toContain("var(--acc)");
    expect(source).toContain("var(--acc-on)");
  });
});
