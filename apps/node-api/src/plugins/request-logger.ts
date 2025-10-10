import type { FastifyInstance } from "fastify";
import fp from "fastify-plugin";

async function requestLogger(fastify: FastifyInstance) {
  fastify.addHook("onRequest", async (request) => {
    request.log.info(
      {
        method: request.method,
        url: request.url,
        headers: {
          "user-agent": request.headers["user-agent"],
          "x-forwarded-for": request.headers["x-forwarded-for"],
        },
        ip: request.ip,
      },
      "Incoming request"
    );
  });

  fastify.addHook("onResponse", async (request, reply) => {
    request.log.info(
      {
        method: request.method,
        url: request.url,
        statusCode: reply.statusCode,
        responseTime: reply.getResponseTime(),
      },
      "Request completed"
    );
  });
}

export default fp(requestLogger, {
  name: "request-logger",
});
