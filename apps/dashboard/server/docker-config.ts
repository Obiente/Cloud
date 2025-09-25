import { Configuration } from '@obiente/docker-engine';

export const config = new Configuration({
  baseOptions: {
    socketPath: '/var/run/docker.sock',
  },
});
