import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

export const config = {
  // Server configuration
  port: parseInt(process.env.PORT || '3001', 10),
  host: process.env.HOST || '0.0.0.0',
  
  // Environment
  nodeEnv: process.env.NODE_ENV || 'development',
  isDev: process.env.NODE_ENV !== 'production',
  
  // Logging
  logLevel: process.env.LOG_LEVEL || 'info',
  
  // Database
  database: {
    url: process.env.DATABASE_URL || 'postgresql://user:password@localhost:5432/obiente_cloud',
  },
  
  // Authentication (Zitadel)
  auth: {
    zitadelUrl: process.env.ZITADEL_URL || 'https://your-zitadel.domain.com',
    clientId: process.env.ZITADEL_CLIENT_ID || '',
    clientSecret: process.env.ZITADEL_CLIENT_SECRET || '',
    redirectUri: process.env.ZITADEL_REDIRECT_URI || 'http://localhost:3000/auth/callback',
    jwksUri: process.env.ZITADEL_JWKS_URI || '',
  },
  
  // CORS configuration
  cors: {
    origin: process.env.CORS_ORIGIN 
      ? process.env.CORS_ORIGIN.split(',')
      : ['http://localhost:3000', 'http://localhost:5173'],
  },
  
  // Stripe (for billing)
  stripe: {
    secretKey: process.env.STRIPE_SECRET_KEY || '',
    webhookSecret: process.env.STRIPE_WEBHOOK_SECRET || '',
  },
  
  // API rate limiting
  rateLimit: {
    windowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '60000', 10), // 1 minute
    max: parseInt(process.env.RATE_LIMIT_MAX || '100', 10), // 100 requests per window
  },
  
  // Security
  security: {
    jwtSecret: process.env.JWT_SECRET || 'your-super-secret-jwt-key',
    sessionSecret: process.env.SESSION_SECRET || 'your-super-secret-session-key',
  },
} as const;