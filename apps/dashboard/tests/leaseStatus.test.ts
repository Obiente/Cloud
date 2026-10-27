import { describe, it, expect } from "vitest";
import { leaseStatus } from "../app/utils/leaseStatus";
describe("DHCP lease expiry", () => {
 it("distinguishes expired, missing and future leases", () => {
  const now = Date.UTC(2026,0,1);
  expect(leaseStatus(undefined, now)).toBe("unknown");
  expect(leaseStatus(new Date(NaN), now)).toBe("unknown");
  expect(leaseStatus(new Date(now), now)).toBe("expired");
  expect(leaseStatus(new Date(now + 3600000), now)).toBe("expiring");
  expect(leaseStatus(new Date(now + 86400000), now)).toBe("active");
 });
});
