import Fastify from 'fastify';
import cors from '@fastify/cors';
import helmet from '@fastify/helmet';
import multipart from '@fastify/multipart';
import rateLimit from '@fastify/rate-limit';
import swagger from '@fastify/swagger';
import swaggerUI from '@fastify/swagger-ui';
import { connectRoutes } from './routes/index.js';
import { config } from './config/index.js';
import { authPlugin } from './plugins/auth.js';
import { errorHandler } from './plugins/error-handler.js';
import { requestLogger } from './plugins/request-logger.js';

const fastify = Fastify({
  logger: {
    level: config.logLevel,
    transport: config.isDev ? {
      target: 'pino-pretty',
      options: {
        colorize: true,
      },
    } : undefined,
  },
});

// Register plugins
await fastify.register(cors, {
  origin: config.cors.origin,
  credentials: true,
});

await fastify.register(helmet, {
  contentSecurityPolicy: false,
});

await fastify.register(multipart);

await fastify.register(rateLimit, {
  max: 100,
  timeWindow: '1 minute',
});

if (config.isDev) {
  await fastify.register(swagger, {
    swagger: {
      info: {
        title: 'Obiente Cloud API',
        description: 'Multi-tenant cloud dashboard API',
        version: '0.1.0',
      },
      host: `localhost:${config.port}`,
      schemes: ['http'],
      consumes: ['application/json'],
      produces: ['application/json'],
    },
  });

  await fastify.register(swaggerUI, {
    routePrefix: '/docs',
    uiConfig: {
      docExpansion: 'full',
      deepLinking: false,
    },
  });
}

// Register custom plugins
await fastify.register(requestLogger);
await fastify.register(errorHandler);
await fastify.register(authPlugin);

// Register ConnectRPC routes
await fastify.register(connectRoutes);

// Health check endpoint
fastify.get('/health', async () => {
  return { status: 'ok', timestamp: new Date().toISOString() };
});

// Start server
const start = async () => {
  try {
    await fastify.listen({
      port: config.port,
      host: config.host,
    });
    
    fastify.log.info(`Server running on http://${config.host}:${config.port}`);
    
    if (config.isDev) {
      fastify.log.info(`API Documentation available at http://${config.host}:${config.port}/docs`);
    }
  } catch (err) {
    fastify.log.error(err);
    process.exit(1);
  }
};

// Graceful shutdown
const gracefulShutdown = async (signal: string) => {
  fastify.log.info(`Received ${signal}, shutting down gracefully`);
  
  try {
    await fastify.close();
    fastify.log.info('Server closed successfully');
    process.exit(0);
  } catch (err) {
    fastify.log.error('Error during shutdown:', err);
    process.exit(1);
  }
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));

// Start the server
start();