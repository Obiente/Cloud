import type { FastifyInstance } from 'fastify';
import fp from 'fastify-plugin';

// Import ConnectRPC service implementations (will be created later)
// import { authRoutes } from './auth.js';
// import { organizationRoutes } from './organization.js';
// import { deploymentRoutes } from './deployment.js';
// import { vpsRoutes } from './vps.js';
// import { databaseRoutes } from './database.js';
// import { billingRoutes } from './billing.js';

async function connectRoutes(fastify: FastifyInstance) {
  // Register ConnectRPC routes
  // These will be implemented as we progress through the tasks
  
  fastify.log.info('Setting up ConnectRPC routes...');
  
  // Placeholder route for testing
  fastify.get('/api/ping', async () => {
    return { message: 'ConnectRPC API is running', timestamp: new Date().toISOString() };
  });
  
  // TODO: Register actual ConnectRPC service handlers
  // await fastify.register(authRoutes, { prefix: '/api' });
  // await fastify.register(organizationRoutes, { prefix: '/api' });
  // await fastify.register(deploymentRoutes, { prefix: '/api' });
  // await fastify.register(vpsRoutes, { prefix: '/api' });
  // await fastify.register(databaseRoutes, { prefix: '/api' });
  // await fastify.register(billingRoutes, { prefix: '/api' });
  
  fastify.log.info('ConnectRPC routes registered');
}

export default fp(connectRoutes, {
  name: 'connect-routes',
});