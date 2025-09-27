/**
 * Zitadel OAuth token response
 * @see https://zitadel.com/docs/apis/openidoauth/endpoints#token-response
 */
export interface ZitadelTokenResponse {
  access_token: string
  token_type: 'Bearer'
  expires_in: number
  refresh_token?: string
  scope: string
  id_token?: string
}

/**
 * Decoded Zitadel JWT token claims
 * @see https://zitadel.com/docs/apis/openidoauth/claims
 */
export interface ZitadelTokenClaims {
  aud: string | string[]
  exp: number
  iat: number
  iss: string
  sub: string
  auth_time: number
  azp?: string
  email?: string
  email_verified?: boolean
  family_name?: string
  given_name?: string
  locale?: string
  name?: string
  nickname?: string
  preferred_username?: string
  picture?: string
  updated_at?: number
  roles?: string[]
  'urn:zitadel:iam:org:project:roles'?: string[]
  'urn:zitadel:iam:org:projects'?: { [projectId: string]: string[] }
}

/**
 * Zitadel error response
 * @see https://zitadel.com/docs/apis/openidoauth/endpoints#error-response
 */
export interface ZitadelErrorResponse {
  error: string
  error_description?: string
  state?: string
}