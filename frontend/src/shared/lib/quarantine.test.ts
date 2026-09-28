import { describe, expect, it } from "vitest";
import { fileTooLargeForNewAccount, heldBackNotice, opensIn } from "./quarantine";

const held = { until: "2026-09-29T06:00:00Z", hoursLeft: 18, newChatsPerDay: 20, maxUploadBytes: 2 * 1024 * 1024 };

describe("the new-account hold, as the screens show it", () => {
  it("says when the feature opens, in Russian", () => {
    expect(heldBackNotice(held, "Публикация постов откроется")).toBe("Публикация постов откроется через 18 часов");
    expect(opensIn(1)).toBe("через час");
    expect(opensIn(2)).toBe("через 2 часа");
    expect(opensIn(21)).toBe("через 21 час");
    expect(opensIn(11)).toBe("через 11 часов");
  });

  it("says nothing for an account that is not held", () => {
    expect(heldBackNotice(null, "Публикация постов откроется")).toBeNull();
    expect(fileTooLargeForNewAccount(null, { size: 50 * 1024 * 1024 })).toBeNull();
  });

  it("refuses a file over the ceiling before it is uploaded", () => {
    expect(fileTooLargeForNewAccount(held, { size: 3 * 1024 * 1024 })).toContain("до 2 МБ");
    expect(fileTooLargeForNewAccount(held, { size: 2 * 1024 * 1024 })).toBeNull();
  });
});
