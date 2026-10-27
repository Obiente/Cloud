export function leaseStatus(expiresAt: Date | undefined, now = Date.now()): "unknown" | "expired" | "expiring" | "active" {
  const expiry = expiresAt?.getTime();
  if (expiry === undefined || !Number.isFinite(expiry)) return "unknown";
  const remaining = expiry - now;
  if (remaining <= 0) return "expired";
  return remaining < 24 * 60 * 60 * 1000 ? "expiring" : "active";
}
