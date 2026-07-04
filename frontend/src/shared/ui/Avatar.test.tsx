import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Avatar } from "./Avatar";

describe("Avatar", () => {
  it("renders initials when no image url is given", () => {
    render(<Avatar name="Alice Smith" />);
    expect(screen.getByText("AS")).toBeInTheDocument();
  });

  it("renders an image when a url is given", () => {
    render(<Avatar name="Alice" url="https://example.com/a.png" />);
    const img = screen.getByRole("img", { name: "Alice" });
    expect(img).toHaveAttribute("src", "https://example.com/a.png");
  });

  it("shows a presence dot when online", () => {
    const { container } = render(<Avatar name="Bob" presence="online" />);
    expect(container.querySelector(".ui-avatar__presence")).not.toBeNull();
  });
});
