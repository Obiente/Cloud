// Selecting the listed permissions must not silently grant future permissions
// through an inferred wildcard. Wildcards require an explicit selection.
export function updatePermissionSelection(current: string[], requested: string[], checked: boolean): string[] {
  const selection = new Set(current);
  if (checked) {
    requested.forEach(permission => selection.add(permission));
  } else {
    for (const permission of selection) {
      if (requested.some(target => permission === target || permission === "*" ||
        (permission.endsWith(".*") && target.startsWith(permission.slice(0, -1))))) {
        selection.delete(permission);
      }
    }
  }
  return [...selection];
}
