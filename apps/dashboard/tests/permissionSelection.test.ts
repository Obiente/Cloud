import { expect, it } from "vitest";
import { updatePermissionSelection } from "../app/utils/permissionSelection";

it("does not turn the last available read permission into unrestricted access",()=>{
 expect(updatePermissionSelection([], ["database.read"], true)).toEqual(["database.read"]);
});
it("selects concrete group permissions without granting future actions",()=>{
 expect(updatePermissionSelection(["vps.read"], ["database.read","database.manage"], true))
  .toEqual(["vps.read","database.read","database.manage"]);
});
it("preserves an explicitly requested wildcard",()=>{
 expect(updatePermissionSelection([], ["database.*"], true)).toEqual(["database.*"]);
});
it("revokes ancestor wildcards that would defeat deselection",()=>{
 expect(updatePermissionSelection(["*","organization.*","organization.members.*","organization.members.read","database.read"], ["organization.members.read"], false))
  .toEqual(["database.read"]);
});
