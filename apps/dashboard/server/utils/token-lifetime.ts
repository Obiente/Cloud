import type { SecureSessionData } from "@obiente/types";

export function remainingTokenLifetime(secure?: SecureSessionData): number {
  if (!secure?.access_token) return 0;
  let expiry = secure.expires_at;
  // Legacy JWT sessions lack expires_at. This reads expiry only; the API still
  // verifies the token. Legacy opaque tokens must be renewed once.
  if (!Number.isFinite(expiry)) {
    try {
      const payload = JSON.parse(
        Buffer.from(
          secure.access_token.split(".")[1] || "",
          "base64url"
        ).toString("utf8")
      );
      if (typeof payload.exp === "number") expiry = payload.exp * 1000;
    } catch {
      /* Opaque or malformed token: require renewal. */
    }
  }
  return Number.isFinite(expiry)
    ? Math.max(0, Math.floor((expiry! - Date.now()) / 1000))
    : 0;
}
