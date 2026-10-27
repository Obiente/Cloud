import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";

export type AccessRestriction =
  | { scope: "account" }
  | { scope: "organization"; organizationId: string };

export function restrictionFromError(error: unknown): AccessRestriction | null {
  if (!(error instanceof ConnectError) || error.code !== Code.PermissionDenied)
    return null;
  const scope = error.metadata.get("X-Obiente-Restriction");
  if (scope === "account") return { scope };
  const organizationId = error.metadata.get("X-Obiente-Organization-ID");
  if (scope === "organization" && organizationId)
    return { scope, organizationId };
  return null;
}

// Capture the identity and organization when a request starts, including streams
// that can fail after navigation or a change of account.
export function createAccessRestrictionInterceptor(
  capture: () => (restriction: AccessRestriction) => void
): Interceptor {
  return (next) => async (request) => {
    const report = capture();
    const handle = (error: unknown) => {
      const restriction = restrictionFromError(error);
      if (restriction) report(restriction);
    };
    try {
      const response = await next(request);
      if (!response.stream) return response;
      const messages = response.message;
      return {
        ...response,
        message: (async function* () {
          try {
            yield* messages;
          } catch (error) {
            handle(error);
            throw error;
          }
        })(),
      };
    } catch (error) {
      handle(error);
      throw error;
    }
  };
}
