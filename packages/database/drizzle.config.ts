import type { Config } from 'drizzle-kit';

export default {
  schema: './src/schema/index.ts',
  out: './migrations',
  driver: 'pg',
  dbCredentials: {
    connectionString: process.env.DATABASE_URL || 'postgresql://user:password@localhost:5432/obiente_cloud',
  },
  verbose: true,
  strict: true,
} satisfies Config;