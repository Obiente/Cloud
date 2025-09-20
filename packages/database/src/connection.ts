import postgres from 'postgres';
import { drizzle } from 'drizzle-orm/postgres-js';
import * as schema from './schema/index.js';

// Database connection configuration
const connectionString = process.env.DATABASE_URL || 'postgresql://user:password@localhost:5432/obiente_cloud';

// Create postgres client
const client = postgres(connectionString);

// Create drizzle instance with schema
export const db = drizzle(client, { schema });

// Export the client for raw queries if needed
export { client };

// Export types
export type Database = typeof db;