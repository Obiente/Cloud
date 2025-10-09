import type { ConnectRouter } from "@connectrpc/connect";
import { OrganizationService } from "@obiente/proto";
import auth from "./auth.js";
export default (router: ConnectRouter) => {
  auth(router);
  router.service(OrganizationService, {
    async listMembers(req) {
      return {
        members: [
          {
            id: "member-1",
            name: "John Doe",
            email: "john.doe@example.com",
            role: "admin",
          },
          {
            id: "member-2",
            name: "Jane Smith",
            email: "jane.smith@example.com",
            role: "user",
          },
        ],
      };
    },
    async updateMember(req) {
      return {
        success: true,
        message: `Member updated: ${req}`,
      };
    },
  });
};
