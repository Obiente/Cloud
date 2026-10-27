import type { AccessRestriction } from "~/lib/access-restriction";

export const useAccessRestriction = () =>
  useState<AccessRestriction | null>("access-restriction", () => null);
