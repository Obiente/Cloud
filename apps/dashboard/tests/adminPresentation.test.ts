import { describe,expect,it } from "vitest";
import { adminPageContext,formatUsageHours,formatResourceLimit } from "../app/utils/adminPresentation";
describe("admin presentation",()=>{
 it("keeps page context through deep links",()=>{
  expect(adminPageContext("/superadmin/webhook-events")).toEqual({title:"Webhook events",section:"Platform administration"});
  expect(adminPageContext("/superadmin/vps/test-vps").title).toBe("VPS details");
  expect(adminPageContext("/admin/quotas").section).toBe("Organization administration");
 });
 it("distinguishes unlimited, zero usage and unknown values",()=>{
  expect(formatResourceLimit(0)).toBe("Unlimited");expect(formatResourceLimit(undefined)).toBe("Unavailable");
  expect(formatUsageHours(0n,"cpu")).toBe("0 core-hours");
  expect(formatUsageHours(3600n*1024n**3n,"memory")).toBe("1 GiB-hours");
  expect(formatUsageHours(undefined,"memory")).toBe("Unavailable");
 });
});
