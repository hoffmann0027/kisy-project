import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ConsentScreen } from "./ConsentScreen";

/**
 * Before anyone can sign in or sign up, two separate boxes: the privacy policy
 * and the community rules. Google Play's policy for user-generated content
 * wants the rules accepted before anyone can publish, and one combined box
 * would be one consent to two different things.
 */
describe("ConsentScreen", () => {
  const boxes = () => screen.getAllByRole("checkbox");
  const proceed = () => screen.getByRole("button", { name: "Продолжить" });

  it("offers two separate boxes, both unticked", () => {
    render(<ConsentScreen onAccept={vi.fn()} />);
    expect(boxes()).toHaveLength(2);
    for (const box of boxes()) expect(box).not.toBeChecked();
  });

  it("does not let anyone through with only one box ticked", () => {
    const onAccept = vi.fn();
    render(<ConsentScreen onAccept={onAccept} />);

    fireEvent.click(boxes()[0]);
    expect(proceed()).toBeDisabled();
    fireEvent.click(proceed());
    expect(onAccept).not.toHaveBeenCalled();

    fireEvent.click(boxes()[0]);
    fireEvent.click(boxes()[1]);
    expect(proceed()).toBeDisabled();
  });

  it("lets the person through once both are ticked", () => {
    const onAccept = vi.fn();
    render(<ConsentScreen onAccept={onAccept} />);

    fireEvent.click(boxes()[0]);
    fireEvent.click(boxes()[1]);
    expect(proceed()).toBeEnabled();
    fireEvent.click(proceed());
    expect(onAccept).toHaveBeenCalledTimes(1);
  });

  it("opens each document right on the screen, without ticking the box", () => {
    // Reading is not agreeing: opening the rules must not tick anything, and
    // on the phone a link out would leave the app and lose both ticks.
    render(<ConsentScreen onAccept={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Правила сообщества" }));
    expect(screen.getByText("Что запрещено")).toBeInTheDocument();
    for (const box of boxes()) expect(box).not.toBeChecked();
  });

  it("explains why an existing account is asked again", () => {
    render(<ConsentScreen reason="account" onAccept={vi.fn()} />);
    expect(screen.getByText(/обновились/)).toBeInTheDocument();
  });
});
