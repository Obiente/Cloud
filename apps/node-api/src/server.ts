import cors from "@fastify/cors";
import helmet from "@fastify/helmet";
import multipart from "@fastify/multipart";
import rateLimit from "@fastify/rate-limit";
import swagger from "@fastify/swagger";
import swaggerUI from "@fastify/swagger-ui";
import routes from "./routes/index";
import { config } from "./config/index";
// import authPlugin from "./plugins/auth";
// import errorHandler from "./plugins/error-handler";
// import requestLogger from "./plugins/request-logger";
import { fastify } from "fastify";
import { fastifyConnectPlugin } from "@connectrpc/connect-fastify";

const server = fastify({
  logger: {
    level: config.logLevel,
    // transport: config.isDev
    //   ? {
    //     target: "pino-pretty",
    //     options: {
    //       colorize: true,
    //     },
    //   }
    //   : undefined,
  },
});

// // Register plugins

await server.register(helmet, {
  contentSecurityPolicy: false,
});

await server.register(multipart);

await server.register(rateLimit, {
  max: 100,
  timeWindow: "1 minute",
});

if (config.isDev) {
  await server.register(swagger, {
    swagger: {
      info: {
        title: "Obiente Cloud API",
        description: "Multi-tenant cloud dashboard API",
        version: "0.1.0",
      },
      host: `${config.hostname}`,
      schemes: ["http", "https"],
      consumes: ["application/json"],
      produces: ["application/json"],
      methods: ["POST"],
    },
  });

  await server.register(swaggerUI, {
    routePrefix: "/docs",
    uiConfig: {
      docExpansion: "full",
      deepLinking: true,
    },
  });
} else {
  await server.register(cors, {
    origin: config.cors.origin,
    credentials: true,
  });
}
await server.register(fastifyConnectPlugin, {
  routes,
});
// // Register custom plugins
// await server.register(requestLogger);
// await server.register(errorHandler);
// await server.register(authPlugin);

// Register ConnectRPC routes

try {
  await server.listen({
    port: config.port,
    host: config.host,
  });

  server.log.info(`Server running on http://${config.host}:${config.port}`);

  if (config.isDev) {
    server.log.info(
      `API Documentation available at http://${config.hostname}/docs`
    );
  }
} catch (err) {
  server.log.error(err);
  process.exit(1);
}

// Graceful shutdown
const gracefulShutdown = async (signal: string) => {
  server.log.info(`Received ${signal}, shutting down gracefully`);

  try {
    await server.close();
    server.log.info("Server closed successfully");
    process.exit(0);
  } catch (err) {
    server.log.error("Error during shutdown:", err);
    process.exit(1);
  }
};

process.on("SIGTERM", () => gracefulShutdown("SIGTERM"));
process.on("SIGINT", () => gracefulShutdown("SIGINT"));
