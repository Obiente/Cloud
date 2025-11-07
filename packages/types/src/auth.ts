import type { User } from "./user";

export interface UserSession {
  /** Session ID */
  id: string;
  user?: User;
  secure?: SecureSessionData;
}
export interface UserSessionRequired extends UserSession {
  user: User;
  secure: SecureSessionData;
}
export interface SecureSessionData {
  scope: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
  access_token: string;
  id_token?: string; // ID token for logout (OIDC)
}
export interface SessionResponse {
  session: UserSession | null;
  status: "authenticated" | "unauthenticated";
}
