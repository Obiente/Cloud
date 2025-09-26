import { ContainerApi, NodeApi } from '@obiente/docker-engine';
import { config } from '../../docker-config';

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const id = query && query.id ? String(query.id) : undefined;
  if (!id) throw new Error("Container ID is required");
  const status = query && query.status ? String(query.status) : undefined;
  if (!status) throw new Error("Status parameter is required");

  const api = new ContainerApi(config);
  switch (status) {
    case 'start':
      const startData = (await api.containerStart(id)).data;
      return startData;
      break;
    case 'stop':
      const stopData = (await api.containerStop(id)).data;
      return stopData;
      break;
    case 'restart':
      const restartData = (await api.containerRestart(id)).data;
      return restartData;
      break;
    case 'kill':
      const killData = (await api.containerKill(id)).data;
      return killData;
      break;
    default:
      throw new Error("Invalid status parameter");
      break;
  }
});