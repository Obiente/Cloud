const sections: Record<string,string> = {
 organizations: "Organizations", users: "Users", vps: "VPS", gameservers: "Game servers", databases: "Databases", deployments: "Deployments",
 nodes: "Nodes", plans: "Plans", invoices: "Invoices", income: "Income", abuse: "Abuse detection", dns: "DNS", "webhook-events": "Webhook events",
 quotas: "Quotas", roles: "Roles", bindings: "Role bindings", permissions: "Permissions", support: "Support", notifications: "Notifications",
};
export function adminPageContext(path: string) {
 const parts = path.split("/").filter(Boolean);
 const admin = parts[0] === "superadmin" || parts[0] === "admin";
 const section = admin ? (parts[0] === "superadmin" ? "Platform administration" : "Organization administration") : "";
 const base = parts[admin ? 1 : 0] || (admin ? "overview" : "dashboard");
 const title = sections[base] || (base === "overview" ? "Overview" : base === "dashboard" ? "Dashboard" : base.replace(/-/g," ").replace(/^./,c=>c.toUpperCase()));
 return { title: title + (admin && parts.length > 2 ? " details" : ""), section };
}
export function formatUsageHours(value: bigint | number | null | undefined, unit: "cpu" | "memory") {
 if (value == null) return "Unavailable";
 const hours = Number(value) / 3600 / (unit === "memory" ? 1024 ** 3 : 1);
 return new Intl.NumberFormat(undefined,{maximumFractionDigits:2}).format(hours) + (unit === "memory" ? " GiB-hours" : " core-hours");
}
export function formatResourceLimit(value: number | bigint | null | undefined) {
 return value == null ? "Unavailable" : Number(value) === 0 ? "Unlimited" : new Intl.NumberFormat().format(Number(value));
}
