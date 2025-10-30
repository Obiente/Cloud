import { defineEventHandler } from 'h3';
// Dashboard data endpoint (mocked for now, can be wired to ConnectRPC later)
export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const { createClient } = await import('@connectrpc/connect');
  const { createConnectTransport } = await import('@connectrpc/connect-node');
  const { createAuthInterceptor } = await import('~/app/lib/transport');
  const { getServerToken } = await import('~/server/utils/serverAuth');
  const { DeploymentService, DeploymentStatus, Environment } = await import('@obiente/proto');

  const tokenGetter = async () => getServerToken(event);
  const transport = createConnectTransport({
    baseUrl: config.public.apiHost,
    httpVersion: '1.1',
    useBinaryFormat: false,
    interceptors: [createAuthInterceptor(tokenGetter)],
  });
  const client = createClient(DeploymentService, transport);

  // Fetch deployments for the user's default/selected org (server will resolve if empty)
  let deployments: any[] = [];
  try {
    const res = await client.listDeployments({ organizationId: '' });
    deployments = res.deployments ?? [];
  } catch (e) {
    // If the call fails, keep empty arrays and zeros; the UI will show loading/empty states
    deployments = [];
  }

  // Compute stats
  const deploymentsCount = deployments.length;
  const statusesMap: Record<string, number> = {};
  for (const d of deployments) {
    // d.status is a number (enum). Map to string variants used by UI
    let s = 'PENDING';
    switch (d.status) {
      case DeploymentStatus.RUNNING:
        s = 'RUNNING';
        break;
      case DeploymentStatus.BUILDING:
        s = 'BUILDING';
        break;
      case DeploymentStatus.STOPPED:
        s = 'STOPPED';
        break;
      case DeploymentStatus.FAILED:
        s = 'ERROR';
        break;
      default:
        s = 'PENDING';
    }
    statusesMap[s] = (statusesMap[s] || 0) + 1;
  }
  const statuses = Object.entries(statusesMap).map(([status, count]) => ({ status, count }));

  // Recent deployments: sort by LastDeployedAt or CreatedAt desc
  const recentDeployments = [...deployments]
    .sort((a, b) => {
      const at = (a.lastDeployedAt?.seconds ?? a.createdAt?.seconds ?? 0) * 1000;
      const bt = (b.lastDeployedAt?.seconds ?? b.createdAt?.seconds ?? 0) * 1000;
      return bt - at;
    })
    .slice(0, 5)
    .map((d) => {
      let env = 'production';
      switch (d.environment) {
        case Environment.STAGING:
          env = 'staging';
          break;
        case Environment.DEVELOPMENT:
          env = 'development';
          break;
        default:
          env = 'production';
      }
      const status = (() => {
        switch (d.status) {
          case DeploymentStatus.RUNNING:
            return 'RUNNING';
          case DeploymentStatus.BUILDING:
            return 'BUILDING';
          case DeploymentStatus.STOPPED:
            return 'STOPPED';
          case DeploymentStatus.FAILED:
            return 'ERROR';
          default:
            return 'PENDING';
        }
      })();
      return {
        id: d.id,
        name: d.name,
        domain: d.domain,
        status,
        environment: env,
        updatedAt: new Date(
          ((d.lastDeployedAt?.seconds ?? d.createdAt?.seconds ?? 0) * 1000)
        ).toISOString(),
      };
    });

  // Activity: placeholder until server-side events available
  const activity: Array<{ id: string; message: string; timestamp: string }> = [];

  const stats = {
    deployments: deploymentsCount,
    vpsInstances: 0,
    databases: 0,
    monthlySpend: 0,
    statuses,
  };

  return { stats, recentDeployments, activity };
});
