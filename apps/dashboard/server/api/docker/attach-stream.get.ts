import { ContainerApi } from "@obiente/docker-engine";
import { config } from "../../docker-config";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const { id, follow = "true", tail = "100", timestamps = "true" } = query;

  if (!id || typeof id !== "string") {
    throw createError({
      statusCode: 400,
      statusMessage: "Container ID is required",
    });
  }

  setHeader(event, "Content-Type", "text/event-stream");
  setHeader(event, "Cache-Control", "no-cache");
  setHeader(event, "Connection", "keep-alive");
  setHeader(event, "Access-Control-Allow-Origin", "*");
  setHeader(event, "Access-Control-Allow-Headers", "Cache-Control");

  try {
    const api = new ContainerApi(config);

    const containerResponse = await api.containerInspect(id);
    if (!containerResponse.data) {
      const errorEvent = {
        type: "error",
        data: "Container not found",
        timestamp: new Date().toISOString(),
      };
      event.node.res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
      event.node.res.end();
      return;
    }

    console.log(`Attaching to container ${id}...`);
    const res = await api.containerAttach(
      id,
      undefined,
      true,
      true,
      true,
      true,
      true,
      {
        responseType: "stream",
      }
    );
    console.log("connection established", res);

    const connectEvent = {
      type: "connected",
      data: `Connected to container ${id}`,
      timestamp: new Date().toISOString(),
    };
    event.node.res.write(`data: ${JSON.stringify(connectEvent)}\n\n`);

    if (res) {
      const stream = res.data as unknown as NodeJS.ReadableStream;
      let buffer = Buffer.alloc(0);

      stream.on("data", (chunk: Buffer) => {
        console.log("data received");
        buffer = Buffer.concat([buffer, chunk]);

        while (buffer.length >= 8) {
          // [STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4]
          const streamType = buffer[0];
          const size = buffer.readUInt32BE(4);

          if (buffer.length >= 8 + size) {
            const payload = buffer.slice(8, 8 + size);
            const message = payload.toString("utf8").trim();

            if (message) {
              const eventData = {
                type:
                  streamType === 1
                    ? "stdout"
                    : streamType === 2
                    ? "stderr"
                    : "stdin",
                data: message,
                timestamp: new Date().toISOString(),
              };

              event.node.res.write(`data: ${JSON.stringify(eventData)}\n\n`);
            }

            buffer = buffer.slice(8 + size);
          } else {
            break;
          }
        }
      });

      stream.on("error", (error: Error) => {
        console.error("Stream error:", error);
        const errorEvent = {
          type: "error",
          data: error.message,
          timestamp: new Date().toISOString(),
        };
        event.node.res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
      });

      stream.on("end", () => {
        const endEvent = {
          type: "end",
          data: "Stream ended",
          timestamp: new Date().toISOString(),
        };
        event.node.res.write(`data: ${JSON.stringify(endEvent)}\n\n`);
        event.node.res.end();
      });

      event.node.req.on("close", () => {
        console.log("Client disconnected from container stream");
        if (typeof (stream as any).destroy === "function") {
          (stream as any).destroy();
        }
      });

      return new Promise((resolve) => {
        event.node.req.on("close", resolve);
      });
    }
  } catch (error: any) {
    console.error("Error setting up container stream:", error);

    const errorEvent = {
      type: "error",
      data: error.message || "Failed to connect to container",
      timestamp: new Date().toISOString(),
    };

    event.node.res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
    event.node.res.end();
  }
});
