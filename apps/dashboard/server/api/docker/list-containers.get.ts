import { ContainerApi, NodeApi } from '@obiente/docker-engine';
import { config } from '../../docker-config';

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const id = query && query.id ? query.id : undefined;
  let filters;

  if (id) {
    filters = {
      label: [`id=${id}`]
    };
  }

  const api = new ContainerApi(config);
  const data = (await api.containerList(undefined, undefined, undefined, filters ? JSON.stringify(filters) : undefined)).data;
  return data;
});
