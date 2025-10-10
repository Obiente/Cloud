import type { ConnectRouter } from "@connectrpc/connect";
import { AuthService, type GetCurrentUserResponse } from "@obiente/proto";
import { timestamp } from "@obiente/proto/utils";
export default (router: ConnectRouter) =>
  router.service(AuthService, {
    async getCurrentUser(_req, _context): Promise<GetCurrentUserResponse> {
      // Access the Fastify request from the context
      // The auth middleware should have set the user on the request

      // // Check if user is authenticated (should be set by auth middleware)
      // if (!request?.user) {
      //     throw new ConnectError(
      //         "Authentication required. Please log in.",
      //         Code.Unauthenticated,
      //     );
      // }

      return {
        $typeName: "obiente.cloud.auth.v1.GetCurrentUserResponse",
        user: {
          id: "mock-user-123",
          name: "Mock User",
          email: "mockuser@example.com",
          avatarUrl: "https://www.gravatar.com/avatar/mockuser",
          timezone: "UTC",
          createdAt: timestamp(new Date()),
          $typeName: "obiente.cloud.auth.v1.User",
        },
      };

      // Return the authenticated user from the JWT token
      // return {
      //     $typeName: "obiente.cloud.auth.v1.GetCurrentUserResponse",
      //     user: {
      //         id: request.user.id,
      //         name: request.user.name,
      //         email: request.user.email,
      //         avatarUrl: request.user.avatarUrl || "",
      //         timezone: request.user.timezone || "UTC",
      //         createdAt: timestamp(new Date()),
      //         $typeName: "obiente.cloud.auth.v1.User",
      //     },
      // };
    },
  });
