import { beforeEach, describe, expect, it, vi } from "vitest";
import { computed, ref } from "vue";
import { Code, ConnectError } from "@connectrpc/connect";
const mocks = vi.hoisted(() => ({ permissions: vi.fn(), overview: vi.fn() }));
vi.mock("~/lib/connect-client", () => ({useConnectClient: () => ({getMySuperadminPermissions:mocks.permissions,getOverview:mocks.overview})}));
import { useSuperAdmin } from "../app/composables/useSuperAdmin";

describe("administrator access initialization", () => {
 beforeEach(() => {
  vi.resetAllMocks(); const values = new Map(); const app = {};
  vi.stubGlobal("useNuxtApp", () => app);vi.stubGlobal("computed",computed);
  vi.stubGlobal("useState", (key:string, init:()=>unknown) => {if(!values.has(key))values.set(key,ref(init()));return values.get(key)});
  mocks.overview.mockResolvedValue({counts:{totalOrganizations:2n}});
 });
 it("shares an in-flight request across cold layout and route callers",async()=>{
  let resolve!: (value:unknown)=>void;mocks.permissions.mockImplementation(()=>new Promise(r=>{resolve=r}));
  const first=useSuperAdmin();const second=useSuperAdmin();const a=first.fetchOverview();const b=second.fetchOverview();
  expect(a).toBe(b);expect(first.loading.value).toBe(true);
  resolve({permissions:["superadmin.overview.read"]});await Promise.all([a,b]);
  expect(mocks.permissions).toHaveBeenCalledTimes(1);expect(mocks.overview).toHaveBeenCalledTimes(1);expect(second.allowed.value).toBe(true);
 });
 it("retries transient failures without caching denial",async()=>{
  mocks.permissions.mockRejectedValueOnce(new ConnectError("unavailable",Code.Unavailable)).mockResolvedValueOnce({permissions:["superadmin.overview.read"]});
  const admin=useSuperAdmin();await admin.fetchOverview();expect(admin.allowed.value).toBe(null);expect(admin.initialized.value).toBe(false);
  await admin.fetchOverview();expect(admin.allowed.value).toBe(true);expect(admin.error.value).toBe(null);
 });
 it("does not let an old login request overwrite reset state",async()=>{
  let resolve!:(value:unknown)=>void;mocks.permissions.mockImplementationOnce(()=>new Promise(r=>{resolve=r}));
  const admin=useSuperAdmin();const pending=admin.fetchOverview();admin.reset();resolve({isFullSuperadmin:true,permissions:[]});await pending;
  expect(admin.allowed.value).toBe(null);expect(admin.initialized.value).toBe(false);
 });
});
