import postgres from 'postgres';
import * as schema from './schema/index.js';
declare const client: postgres.Sql<{}>;
export declare const db: import("drizzle-orm/postgres-js/driver.js").PostgresJsDatabase<typeof schema> & {
    $client: postgres.Sql<{}>;
};
export { client };
export type Database = typeof db;
//# sourceMappingURL=connection.d.ts.map