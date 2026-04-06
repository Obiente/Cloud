export type DocsSectionId = "start" | "features" | "management" | "help";

export interface DocsNavItem {
  path: string;
  label: string;
  section: DocsSectionId;
}

export const docsSectionLabels: Record<DocsSectionId, string> = {
  start: "Start",
  features: "Platform",
  management: "Management",
  help: "Help",
};

export const docsNavItems: DocsNavItem[] = [
  {
    path: "/docs",
    label: "Overview",
    section: "start",
  },
  {
    path: "/docs/getting-started",
    label: "Getting Started",
    section: "start",
  },
  {
    path: "/docs/dashboard",
    label: "Dashboard",
    section: "start",
  },
  {
    path: "/docs/deployments",
    label: "Deployments",
    section: "features",
  },
  {
    path: "/docs/gameservers",
    label: "Game Servers",
    section: "features",
  },
  {
    path: "/docs/vps",
    label: "VPS",
    section: "features",
  },
  {
    path: "/docs/databases",
    label: "Databases",
    section: "features",
  },
  {
    path: "/docs/billing",
    label: "Billing",
    section: "management",
  },
  {
    path: "/docs/organizations",
    label: "Organizations",
    section: "management",
  },
  {
    path: "/docs/permissions",
    label: "Permissions",
    section: "management",
  },
  {
    path: "/docs/self-hosting",
    label: "Self-Hosting",
    section: "management",
  },
  {
    path: "/docs/troubleshooting",
    label: "Troubleshooting",
    section: "help",
  },
];

export function getDocsCurrentItem(path: string): DocsNavItem | null {
  return docsNavItems.find((item) => item.path === path) ?? null;
}

export function getDocsNeighbors(path: string): {
  previous: DocsNavItem | null;
  next: DocsNavItem | null;
} {
  const index = docsNavItems.findIndex((item) => item.path === path);

  if (index === -1) {
    return {
      previous: null,
      next: null,
    };
  }

  return {
    previous: docsNavItems[index - 1] ?? null,
    next: docsNavItems[index + 1] ?? null,
  };
}
