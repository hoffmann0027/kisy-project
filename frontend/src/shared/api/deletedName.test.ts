import { afterEach, describe, expect, it } from "vitest";
import { applyDictionary, loadDictionary } from "@shared/i18n";
import { ru } from "@shared/i18n/locales/ru";
import { DELETED_ACCOUNT_NAME, localizeDeletedNames } from "./client";

// The server writes a deleted account's name once, in Russian. A German
// screen must not show Cyrillic where the person used to be.

afterEach(() => applyDictionary("ru", ru));

describe("a deleted account's name", () => {
  it("reads in the language on screen, wherever it sits in a response", async () => {
    applyDictionary("de", await loadDictionary("de"));
    const body = JSON.stringify({ members: [{ displayName: DELETED_ACCOUNT_NAME }, { displayName: "Anna" }], forwardedFrom: { senderName: DELETED_ACCOUNT_NAME } });
    const parsed = JSON.parse(body, localizeDeletedNames);
    expect(parsed.members[0].displayName).toBe("Gelöschtes Konto");
    expect(parsed.members[1].displayName).toBe("Anna");
    expect(parsed.forwardedFrom.senderName).toBe("Gelöschtes Konto");
  });

  it("is the very name the server writes", () => {
    // backend/internal/users/deletion.go: DeletedDisplayName
    expect(DELETED_ACCOUNT_NAME).toBe(ru["common.deletedAccount"]);
  });
});
