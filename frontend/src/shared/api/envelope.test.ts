import { describe, expect, it } from "vitest";
import { ApiError, userFacingError } from "./envelope";

describe("userFacingError", () => {
  it("shows the server's wording when a quota refused the request", () => {
    const err = new ApiError("QUOTA_EXCEEDED", "Место для ваших файлов закончилось", "rid", 413);
    expect(userFacingError(err, "Не удалось загрузить")).toBe("Место для ваших файлов закончилось");
  });

  it("keeps the generic text for every other failure", () => {
    expect(userFacingError(new ApiError("INTERNAL_ERROR", "internal error", "rid", 500), "Не удалось")).toBe("Не удалось");
    expect(userFacingError(new Error("boom"), "Не удалось")).toBe("Не удалось");
  });
});
