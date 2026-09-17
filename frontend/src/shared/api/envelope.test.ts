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

  it("tells how long to wait when a rate limit refused the request", () => {
    expect(userFacingError(new ApiError("RATE_LIMITED", "too many requests, slow down", "rid", 429, 42), "Не удалось")).toBe(
      "Слишком часто. Попробуйте через 42 с",
    );
    expect(userFacingError(new ApiError("RATE_LIMITED", "too many requests, slow down", "rid", 429, 1800), "Не удалось")).toBe(
      "Слишком часто. Попробуйте через 30 мин",
    );
  });
});
