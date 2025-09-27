import { subtle, getRandomValues } from 'uncrypto';
function getRandomBytes(size: number = 32) {
  return getRandomValues(new Uint8Array(size));
}

function encodeBase64Url(input: Uint8Array): string {
  return btoa(String.fromCharCode.apply(null, input as unknown as number[]))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '');
}

export default async function (callback: boolean = false): Promise<{
  code_verifier: string;
  code_challenge?: string;
  code_challenge_method?: string;
}> {
  const verifier = useCookie<string>('pkce-verifier');
  if (callback) {
    return { code_verifier: verifier.value };
  }
  verifier.value = encodeBase64Url(getRandomBytes());

  // Get pkce
  const encodedPkce = new TextEncoder().encode(verifier.value);
  const pkceHash = await subtle.digest('SHA-256', encodedPkce);
  const pkce = encodeBase64Url(new Uint8Array(pkceHash));

  return {
    code_verifier: verifier.value,
    code_challenge: pkce,
    code_challenge_method: 'S256',
  };
}

