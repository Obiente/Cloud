import { ContainerApi } from '@obiente/docker-engine';
import { config } from '../../docker-config';

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const { all = 'false', size = 'false' } = query;
  
  try {
    const api = new ContainerApi(config);
    
    // List containers (all=true includes stopped containers)
    const response = await api.containerList(
      all === 'true',     // all containers or just running
      undefined,          // limit
      size === 'true',    // include size information
      undefined           // filters
    );

    // Transform the response to include useful information
    const containers = response.data.map(container => ({
      id: container.Id,
      names: container.Names,
      image: container.Image,
      imageId: container.ImageID,
      command: container.Command,
      created: container.Created,
      ports: container.Ports,
      labels: container.Labels,
      state: container.State,
      status: container.Status,
      hostConfig: container.HostConfig,
      networkSettings: container.NetworkSettings,
      mounts: container.Mounts,
      size: container.SizeRw || 0,
      sizeRootFs: container.SizeRootFs || 0
    }));

    return {
      success: true,
      containers,
      count: containers.length
    };
    
  } catch (error) {
    console.error('Error listing containers:', error);
    throw createError({
      statusCode: 500,
      statusMessage: 'Failed to list containers: ' + (error as Error).message
    });
  }
});
