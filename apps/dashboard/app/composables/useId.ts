let idCounter = 0;

export function useUniqueId(prefix = "oui"): string {
  idCounter += 1;
  return `${prefix}-${idCounter}`;
}

