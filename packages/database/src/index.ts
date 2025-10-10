export * from "./schema/index.js";

// Re-export commonly used Drizzle ORM functions
export {
  sql,
  eq,
  and,
  or,
  not,
  isNull,
  isNotNull,
  inArray,
  notInArray,
} from "drizzle-orm";
export { drizzle } from "drizzle-orm/postgres-js";
export type { PostgresJsDatabase } from "drizzle-orm/postgres-js";
