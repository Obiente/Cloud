import type { FastifyInstance } from "fastify";
import fp from "fastify-plugin";

async function errorHandler(fastify: FastifyInstance) {
  fastify.setErrorHandler(async (error, request, reply) => {
    // Log the error
    fastify.log.error(
      {
        error: error.message,
        stack: error.stack,
        url: request.url,
        method: request.method,
        headers: request.headers,
      },
      "Request error"
    );

    // ConnectRPC errors
    if (error.name === "ConnectError") {
      return reply.code(error.code || 500).send({
        error: error.message,
        code: error.code,
      });
    }

    // Validation errors
    if (error.validation) {
      return reply.code(400).send({
        error: "Validation error",
        details: error.validation,
      });
    }

    // Authentication errors
    if (
      error.message.includes("Authorization") ||
      error.message.includes("token")
    ) {
      return reply.code(401).send({
        error: "Authentication failed",
        message: error.message,
      });
    }

    // Database errors
    if (
      error.message.includes("database") ||
      error.message.includes("connection")
    ) {
      return reply.code(500).send({
        error: "Database error",
        message: "Unable to process request. Please try again later.",
      });
    }

    // Default error response
    const statusCode = error.statusCode || 500;
    const message =
      statusCode === 500 ? "Internal server error" : error.message;

    return reply.code(statusCode).send({
      error: message,
      ...(fastify.config?.isDev && { stack: error.stack }),
    });
  });

  // Handle 404 errors
  fastify.setNotFoundHandler(async (request, reply) => {
    return reply.code(404).send({
      error: "Route not found",
      message: `Cannot ${request.method} ${request.url}`,
    });
  });
}

export default fp(errorHandler, {
  name: "error-handler",
});
