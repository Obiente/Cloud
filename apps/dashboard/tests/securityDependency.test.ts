import { createRequire } from 'node:module';
import { generateKeyPairSync } from 'node:crypto';
import { describe, expect, it } from 'vitest';

// Resolve the actual transitive dependency used by the Nuxt server toolchain.
let dependencyRequire = createRequire(import.meta.url);
for (const dependency of ['nuxt', '@nuxt/nitro-server', 'nitropack', 'listhen']) {
  dependencyRequire = createRequire(dependencyRequire.resolve(dependency === 'listhen' ? dependency : `${dependency}/package.json`));
}
const forge = dependencyRequire('node-forge');

describe('patched RSA DigestAlgorithm verification', () => {
  const generated = generateKeyPairSync('rsa', { modulusLength: 2048 });
  const privateKey = forge.pki.privateKeyFromPem(generated.privateKey.export({ type: 'pkcs1', format: 'pem' }));
  const publicKey = forge.pki.publicKeyFromPem(generated.publicKey.export({ type: 'pkcs1', format: 'pem' }));
  const digest = forge.md.sha256.create().update('synthetic regression fixture').digest().getBytes();
  const asn = forge.asn1;
  const nullElement = (value = '') => asn.create(asn.Class.UNIVERSAL, asn.Type.NULL, false, value);
  const signDigestInfo = (parameters: unknown[]) => {
    const algorithm = asn.create(asn.Class.UNIVERSAL, asn.Type.SEQUENCE, true, [
      asn.create(asn.Class.UNIVERSAL, asn.Type.OID, false, asn.oidToDer(forge.oids.sha256).getBytes()), ...parameters,
    ]);
    const info = asn.create(asn.Class.UNIVERSAL, asn.Type.SEQUENCE, true, [algorithm,
      asn.create(asn.Class.UNIVERSAL, asn.Type.OCTETSTRING, false, digest),
    ]);
    return privateKey.sign(asn.toDer(info).getBytes(), 'NONE');
  };

  it('accepts valid SHA-256 signatures with absent or NULL parameters', () => {
    expect(publicKey.verify(digest, signDigestInfo([]))).toBe(true);
    expect(publicKey.verify(digest, signDigestInfo([nullElement()]))).toBe(true);
  });
  it('rejects extra nested elements and nonempty NULL parameters', () => {
    for (const fields of [[nullElement(), nullElement()], [nullElement('garbage')],
      [nullElement(), asn.create(asn.Class.UNIVERSAL, asn.Type.OCTETSTRING, false, 'garbage')]]) {
      expect(() => publicKey.verify(digest, signDigestInfo(fields))).toThrow(/DigestAlgorithm/);
    }
  });
});
