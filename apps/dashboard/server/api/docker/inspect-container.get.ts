import { ContainerApi, NodeApi } from "@obiente/docker-engine";
import { config } from "../../docker-config";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const id = query && query.id ? String(query.id) : undefined;
  if (!id) throw new Error("Container ID is required");

  const api = new ContainerApi(config);
  const data = (await api.containerInspect(id)).data;
  return data;
});
