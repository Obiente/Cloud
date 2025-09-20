import { migrate } from 'drizzle-orm/postgres-js/migrator';
import { client, db } from './connection.js';
async function runMigrations() {
    console.log('Running database migrations...');
    try {
        await migrate(db, {
            migrationsFolder: './migrations',
        });
        console.log('Migrations completed successfully!');
    }
    catch (error) {
        console.error('Migration failed:', error);
        process.exit(1);
    }
    finally {
        await client.end();
    }
}
// Run migrations if this file is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
    runMigrations();
}
