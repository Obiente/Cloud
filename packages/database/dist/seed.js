import { db, client } from './connection.js';
import { organizations, users, organizationMemberships, billingAccounts } from './schema/index.js';
async function seedDatabase() {
    console.log('Seeding database with initial data...');
    try {
        // Create a sample organization
        const [org] = await db.insert(organizations).values({
            name: 'Acme Corporation',
            slug: 'acme-corp',
            description: 'Sample organization for testing',
            settings: { theme: 'light', notifications: true }
        }).returning();
        console.log('Created organization:', org.name);
        // Create a sample user
        const [user] = await db.insert(users).values({
            externalId: 'zitadel-user-123',
            email: 'admin@acme-corp.com',
            name: 'Admin User',
            preferences: { language: 'en', timezone: 'UTC' }
        }).returning();
        console.log('Created user:', user.name);
        // Create organization membership
        await db.insert(organizationMemberships).values({
            organizationId: org.id,
            userId: user.id,
            role: 'OWNER'
        });
        console.log('Created organization membership');
        // Create a sample billing account
        const [billingAccount] = await db.insert(billingAccounts).values({
            organizationId: org.id,
            status: 'ACTIVE',
            billingEmail: 'billing@acme-corp.com',
            companyName: 'Acme Corporation'
        }).returning();
        console.log('Created billing account');
        console.log('Database seeding completed successfully!');
    }
    catch (error) {
        console.error('Seeding failed:', error);
        process.exit(1);
    }
    finally {
        await client.end();
    }
}
// Run seeding if this file is executed directly
if (require.main === module) {
    seedDatabase();
}
