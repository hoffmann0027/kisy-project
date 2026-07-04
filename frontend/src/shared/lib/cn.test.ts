import { describe, expect, it } from "vitest";
import { cn } from "./cn";

describe("cn", () => {
  it("joins truthy class names", () => {
    expect(cn("a", "b", "c")).toBe("a b c");
  });
  it("drops falsy values", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });
  it("supports conditional classes", () => {
    const active = true;
    const disabled = false;
    expect(cn("btn", active && "btn--active", disabled && "btn--disabled")).toBe("btn btn--active");
  });
  it("returns an empty string with no classes", () => {
    expect(cn()).toBe("");
  });
});
