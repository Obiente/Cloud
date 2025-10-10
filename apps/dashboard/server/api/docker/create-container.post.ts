import { ContainerApi, NodeApi } from "@obiente/docker-engine";
import { config } from "../../docker-config";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);

  const randomName =
    Math.random().toString(36).substring(2, 15) +
    Math.random().toString(36).substring(2, 15);

  const api = new ContainerApi(config);

  const createResponse = await api.containerCreate({
    Hostname: randomName,
    Image: "alpine",
    Cmd: ["tail", "-f", "/dev/null"],
    Labels: {
      id: "dargy",
    },
    AttachStdin: false,
    AttachStdout: false,
    AttachStderr: false,
    Tty: false,
  });

  const containerId = createResponse.data.Id;
  if (containerId) {
    await api.containerStart(containerId);
  }

  return createResponse.data;
});
