import { subtle, getRandomValues } from "uncrypto";
import type { H3Event } from "h3";
import { _useSession } from "./session";

function getRandomBytes(size: number = 32) {
  return getRandomValues(new Uint8Array(size));
}

function encodeBase64Url(input: Uint8Array): string {
  return btoa(String.fromCharCode.apply(null, input as unknown as number[]))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

export default async function (event: H3Event): Promise<{
  code_verifier: string;
  code_challenge?: string;
  code_challenge_method?: string;
}> {
  const verifier = await _useSession<{ code_verifier: string }>(event, {
    name: "pkce-verifier",
  });
  const data = verifier.data;
  if (data.code_verifier && event.path.startsWith("/auth/callback")) {
    await verifier.clear();
    return { code_verifier: data.code_verifier };
  }
  data.code_verifier = encodeBase64Url(getRandomBytes());
  await verifier.update(data);
  const encodedPkce = new TextEncoder().encode(data.code_verifier);
  const pkceHash = await subtle.digest("SHA-256", encodedPkce);
  const pkce = encodeBase64Url(new Uint8Array(pkceHash));

  return {
    code_verifier: data.code_verifier,
    code_challenge: pkce,
    code_challenge_method: "S256",
  };
}
