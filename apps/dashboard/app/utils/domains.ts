import { GameType } from "@obiente/proto";

const OBIENTE_DOMAIN_SUFFIX = "my.obiente.cloud";

export function normalizeDomain(domain?: string | null): string {
  return String(domain || "").trim().toLowerCase().replace(/\.$/, "");
}

export function isDefaultObienteDomain(
  domain?: string | null,
  prefixes: string[] = []
): boolean {
  const normalized = normalizeDomain(domain);
  if (!normalized.endsWith(`.${OBIENTE_DOMAIN_SUFFIX}`)) {
    return false;
  }

  if (prefixes.length === 0) {
    return true;
  }

  const label = normalized.slice(0, -(OBIENTE_DOMAIN_SUFFIX.length + 1));
  return prefixes.some((prefix) => label.startsWith(`${prefix}-`));
}

export function getDefaultObienteDomain(resourceId?: string | null): string {
  const label = String(resourceId || "").trim().toLowerCase();
  return label ? `${label}.${OBIENTE_DOMAIN_SUFFIX}` : "";
}

export function getGameServerConnectionDomain(
  gameServerId?: string | null
): string {
  return getDefaultObienteDomain(gameServerId);
}

export function getGameServerSrvDomains(
  gameServerId?: string | null,
  gameType?: number | null
): Array<{ label: string; domain: string; description: string }> {
  const targetDomain = getGameServerConnectionDomain(gameServerId);
  if (!targetDomain || gameType === undefined || gameType === null) {
    return [];
  }

  const domains: Array<{ label: string; domain: string; description: string }> =
    [];

  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_JAVA) {
    domains.push({
      label: "Minecraft Java (SRV)",
      domain: `_minecraft._tcp.${targetDomain}`,
      description:
        "Use this domain in Minecraft Java Edition for automatic port resolution",
    });
  }

  if (
    gameType === GameType.MINECRAFT ||
    gameType === GameType.MINECRAFT_BEDROCK
  ) {
    domains.push({
      label: "Minecraft Bedrock (SRV)",
      domain: `_minecraft._udp.${targetDomain}`,
      description:
        "Use this domain in Minecraft Bedrock Edition for automatic port resolution",
    });
  }

  if (gameType === GameType.RUST) {
    domains.push({
      label: "Rust (SRV)",
      domain: `_rust._udp.${targetDomain}`,
      description: "Use this domain in Rust for automatic port resolution",
    });
  }

  return domains;
}
