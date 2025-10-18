// Dashboard data endpoint (mocked for now, can be wired to ConnectRPC later)
export default defineEventHandler(async () => {
	// In a real implementation, call backend services here (ConnectRPC or REST)
	// to fetch stats, recent deployments, and activity.

	const stats = {
		deployments: 12,
		vpsInstances: 4,
		databases: 3,
		monthlySpend: 145.5,
		environments: [
			{ name: "production", deployments: 6 },
			{ name: "staging", deployments: 4 },
			{ name: "development", deployments: 2 },
		],
		statuses: [
			{ status: "RUNNING", count: 8 },
			{ status: "BUILDING", count: 2 },
			{ status: "STOPPED", count: 1 },
			{ status: "ERROR", count: 1 },
		],
	};

	const recentDeployments = [
		{
			id: "1",
			name: "dashboard",
			domain: "dashboard.obiente.cloud",
			status: "RUNNING" as const,
			environment: "production" as const,
			updatedAt: new Date().toISOString(),
		},
		{
			id: "2",
			name: "marketing",
			domain: "marketing.obiente.cloud",
			status: "BUILDING" as const,
			environment: "staging" as const,
			updatedAt: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
		},
		{
			id: "3",
			name: "api",
			domain: "api.obiente.cloud",
			status: "STOPPED" as const,
			environment: "development" as const,
			updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 3).toISOString(),
		},
	];

	const activity = [
		{
			id: "a1",
			message: 'Deployment "dashboard" promoted to production',
			timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
		},
		{
			id: "a2",
			message: 'Backup completed for database "prod-db"',
			timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
		},
		{
			id: "a3",
			message: 'VPS "web-01" scaled to 2 vCPU',
			timestamp: new Date(Date.now() - 1000 * 60 * 60 * 6).toISOString(),
		},
	];

	return { stats, recentDeployments, activity };
});
