import { ContainerApi, NodeApi } from '@obiente/docker-engine';
import { config } from '../docker-config';

export default defineEventHandler(async event => {
  const api = new ContainerApi(config);
  const data = (await api.containerList()).data;
  return data;
});
