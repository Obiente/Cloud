# Preview TLS certificates

Obiente Cloud uses one certificate for `my.obiente.cloud` and
`*.my.obiente.cloud`. It is loaded by every Traefik replica and covers both
running pull request previews and the fallback page shown while a preview is
stopped, waiting for approval, or building.

The included tooling uses [lego](https://go-acme.github.io/lego/) in a
short-lived container. lego supports more than 200 DNS providers and follows
DNS challenge CNAMEs by default. Obiente Cloud does not require credentials
for a particular DNS vendor.

## Prepare provider credentials

Find the environment variables for your provider in the
[lego DNS provider list](https://go-acme.github.io/lego/dns/). Put them in a
root-owned file outside the repository:

```bash
install -d -m 700 /etc/obiente
install -m 600 /dev/null /etc/obiente/preview-dns.env
editor /etc/obiente/preview-dns.env
```

For example, the file can contain:

```dotenv
PROVIDER_API_TOKEN=replace-with-the-provider-specific-token
```

Use the exact variable names documented for the selected provider. The setup
tool rejects group-readable or world-readable credential files and known lego
debug options that can print credentials.

When the bundled Obiente DNS service is authoritative for
`my.obiente.cloud`, set `PREVIEW_ACME_CHALLENGE_CNAME` in `.env` to a name in a
zone the selected provider can update. The target must be outside the entire
`my.obiente.cloud` zone:

```dotenv
PREVIEW_ACME_CHALLENGE_CNAME=_acme-challenge-preview.example.net
```

Deploy this DNS configuration before requesting the certificate. When `dig` is
available, the management script verifies that the public challenge CNAME is
present and points to the configured target before contacting the ACME server.
Set `ENABLE_DNS=false` only when another authoritative DNS service directly
handles the preview zone.

## One-time setup

### New cluster bootstrap

When the bundled DNS service is not deployed yet, create a short-lived
self-signed certificate so the first stack deployment can start its DNS and
Traefik services:

```bash
./scripts/manage-preview-tls.sh bootstrap
./scripts/deploy-swarm.sh
```

Bootstrap refuses to run after Traefik exists or when the selected secrets
already exist. Its certificate expires after seven days and is not suitable as
the final public certificate. Once the public challenge CNAME answers, run the
trusted setup below immediately; it rotates Traefik without redeploying the
rest of the stack.

### Trusted certificate

Run the setup on a Docker Swarm manager:

```bash
./scripts/manage-preview-tls.sh setup \
  --provider PROVIDER_CODE \
  --credentials-file /etc/obiente/preview-dns.env \
  --email admin@example.com \
  --accept-tos
```

Use `check` with the same provider, credentials, email, and TOS arguments to
validate permissions and configuration without pulling lego, issuing a
certificate, creating secrets, or touching the deployment.

The command:

1. Stores the ACME account and certificate under
   `/var/lib/obiente/preview-tls` with private permissions.
2. Runs the pinned lego container without capabilities, with a read-only root
   filesystem and the provider file mounted read-only.
3. Validates the apex and wildcard SANs, expiry, and private-key match.
4. Creates immutable, versioned Docker Swarm secrets.
5. Atomically updates `PREVIEW_TLS_CERT_SECRET` and
   `PREVIEW_TLS_KEY_SECRET` in `.env`.
6. Updates only the deployed Traefik service. If Traefik is not deployed yet,
   the next stack deployment loads the selected secrets.

Provider credentials are not copied into `.env`, passed as container
environment variables, or printed. Existing versioned secrets are retained so
an operator can roll back.

Use `--no-activate` to create and select the secrets without updating a running
Traefik service. Use `--force` only when an immediate replacement certificate
is necessary.

## Check status

```bash
./scripts/manage-preview-tls.sh status
```

This shows the locally stored certificate dates and fingerprint, selected
Swarm secret names, and the secret sources mounted by Traefik. It never prints
the private key or provider credentials.

## Automatic renewal

Install the systemd service and timer from the repository checkout:

```bash
sudo ./scripts/install-preview-tls-renewal.sh \
  --provider PROVIDER_CODE \
  --credentials-file /etc/obiente/preview-dns.env \
  --email admin@example.com \
  --accept-tos
```

The installer copies the credential file into `/etc/obiente` with mode `0600`,
installs a standalone copy of the management script, performs the initial
setup, and enables a daily timer. The timer has a six-hour randomized delay to
avoid synchronized ACME traffic. lego only renews when the configured renewal
window is reached; the default is 30 days.

Rerun the installer after changing provider credentials or after updating the
repository's renewal tooling. It validates the new configuration before
replacing the installed copy.

Useful commands:

```bash
systemctl status obiente-preview-tls-renew.timer
systemctl list-timers obiente-preview-tls-renew.timer
journalctl -u obiente-preview-tls-renew.service
systemctl start obiente-preview-tls-renew.service
```

The unattended renewal path is guarded by an operation lock. A successful
renewal creates new versioned secrets, updates `.env`, and rolls Traefik to the
new certificate. If the Traefik update fails, the previous `.env` is restored,
Docker is asked to roll the service back, and the new secrets remain available
for diagnosis.

## Staging and custom ACME servers

Test provider credentials against the Let's Encrypt staging directory before
production issuance:

```bash
./scripts/manage-preview-tls.sh setup \
  --provider PROVIDER_CODE \
  --credentials-file /etc/obiente/preview-dns.env \
  --email admin@example.com \
  --ca-server https://acme-staging-v02.api.letsencrypt.org/directory \
  --accept-tos \
  --issue-only
```

Issue-only mode obtains and validates the staging certificate but does not
create Swarm secrets, change `.env`, or update Traefik. The script refuses to
use a staging endpoint without this guard. Issue-only certificates and account
data are kept in a separate state directory so they cannot replace the local
production renewal state. Remove the staging override and run setup again for a
trusted production certificate. The lego image, state directory, renewal
threshold, stack name, and CA server can also be set with the `PREVIEW_TLS_*`
variables documented in the environment variable reference.
