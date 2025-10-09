import type { FastifyInstance, FastifyRequest, FastifyReply } from 'fastify';
import fp from 'fastify-plugin';
import { jwtVerify, importJWK } from 'jose';
import { config } from '../config/index.js';

// Interface for authenticated user
export interface AuthenticatedUser {
  id: string;
  email: string;
  name: string;
  organizationId?: string;
  role?: string;
}

// Extend Fastify request type to include user
declare module 'fastify' {
  interface FastifyRequest {
    user?: AuthenticatedUser;
  }
}

async function authPlugin(fastify: FastifyInstance) {
  // JWT verification function
  const verifyToken = async (token: string): Promise<AuthenticatedUser | null> => {
    try {
      // For now, we'll use a simple JWT verification
      // In production, this should verify against Zitadel's public keys
      const { payload } = await jwtVerify(token, new TextEncoder().encode(config.security.jwtSecret));
      
      return {
        id: payload.sub as string,
        email: payload.email as string,
        name: payload.name as string,
        organizationId: payload.organizationId as string,
        role: payload.role as string,
      };
    } catch (error) {
      fastify.log.warn('JWT verification failed:', error);
      return null;
    }
  };

  // Authentication hook
  fastify.addHook('preHandler', async (request: FastifyRequest, reply: FastifyReply) => {
    // Skip authentication for health check and public routes
    if (request.url === '/health' || request.url.startsWith('/docs')) {
      return;
    }

    const authHeader = request.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      reply.code(401).send({ error: 'Missing or invalid authorization header' });
      return;
    }

    const token = authHeader.substring(7); // Remove 'Bearer ' prefix
    const user = await verifyToken(token);

    if (!user) {
      reply.code(401).send({ error: 'Invalid or expired token' });
      return;
    }

    request.user = user;
  });

  // Helper function to require authentication
  fastify.decorate('requireAuth', function(this: FastifyInstance) {
    return async (request: FastifyRequest, reply: FastifyReply) => {
      if (!request.user) {
        reply.code(401).send({ error: 'Authentication required' });
        return;
      }
    };
  });

  // Helper function to require specific role
  fastify.decorate('requireRole', function(this: FastifyInstance, role: string) {
    return async (request: FastifyRequest, reply: FastifyReply) => {
      if (!request.user) {
        reply.code(401).send({ error: 'Authentication required' });
        return;
      }

      if (request.user.role !== role) {
        reply.code(403).send({ error: 'Insufficient permissions' });
        return;
      }
    };
  });
}

export default fp(authPlugin, {
  name: 'auth-plugin',
});