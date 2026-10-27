import { readFileSync } from "node:fs";
import { expect, it } from "vitest";
const css = readFileSync(new URL("../app/assets/css/main.css", import.meta.url), "utf8");
const luminance = (hex: string) => {
 const channels = hex.match(/../g)!.map(value => Number.parseInt(value,16)/255).map(value => value <= 0.04045 ? value/12.92 : ((value+0.055)/1.055)**2.4);
 return channels[0]!*0.2126+channels[1]!*0.7152+channels[2]!*0.0722;
};
it("keeps primary, secondary and tertiary text legible across theme surfaces", () => {
 const themes = [...css.matchAll(/:root\[data-theme="([^"]+)"\]\s*\{([^}]+)\}/g)];
 expect(themes).toHaveLength(3);
 for (const [,theme,block] of themes) {
  const tokens = Object.fromEntries([...block!.matchAll(/--oui-([\w-]+):\s*#([0-9a-f]{6})/g)].map(match=>[match[1],match[2]]));
  for (const text of ["text-primary","text-secondary","text-tertiary"]) {
   for (const background of ["background","surface-base","surface-raised","surface-overlay","surface-muted"]) {
    const a=luminance(tokens[text]!),b=luminance(tokens[background]!);
    expect((Math.max(a,b)+0.05)/(Math.min(a,b)+0.05),`${theme}: ${text} on ${background}`).toBeGreaterThanOrEqual(4.5);
   }
  }
 }
});
