// Helper to create AlterTableOperation messages for the dashboard composable
import { AlterTableOperationSchema } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";

export function makeAlterTableOperation(op: any) {
  // op: { addColumn: { ... } } or { dropColumn: { ... } } etc.
  // Find the operation key
  const keys = [
    "addColumn",
    "dropColumn",
    "modifyColumn",
    "renameColumn",
    "addForeignKey",
    "dropForeignKey",
    "addUnique",
    "dropConstraint"
  ];
  for (const key of keys) {
    if (op[key]) {
      return create(AlterTableOperationSchema, {
        operation: { case: key, value: op[key] }
      });
    }
  }
  throw new Error("Invalid alter table operation");
}
