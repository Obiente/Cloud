import { describe, expect, it } from "vitest";
import { formatDate, formatDateOnly } from "../app/utils/common";

describe("server and client date presentation", () => {
  it("uses explicit UTC across midnight and accepts the epoch", () => {
    expect(formatDate("2026-01-01T00:30:00+02:00")).toBe("31 Dec 2025, 22:30:00 UTC");
    expect(formatDateOnly({seconds:0n,nanos:0})).toBe("1 Jan 1970");
    expect(formatDate(0)).toBe("1 Jan 1970, 00:00:00 UTC");
  });
  it("does not manufacture a date for missing or invalid data", () => {
    for (const value of [null, undefined, "", "invalid-date"]) {
      expect(formatDate(value)).toBe("Unavailable");
      expect(formatDateOnly(value)).toBe("Unavailable");
    }
  });
});
