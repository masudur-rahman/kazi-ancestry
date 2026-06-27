# kazi-ancestry Helm chart

Deploys the Kazi Ancestry server (Go + Postgres, Gateway API) to Kubernetes.
**Postgres is not provisioned** — point the chart at an existing database.

## Resources

| Template | Resource | Notes |
|----------|----------|-------|
| `deployment.yaml` | Deployment | runs `kazi-ancestry serve`; `/healthz` probes; read-only rootfs |
| `service.yaml` | Service | ClusterIP → container port 5294 |
| `configmap.yaml` | ConfigMap | the app's native `kazi-ancestry.yaml` (DB conn, server, OAuth client id, redirect URL, admins, allowlist, privacy) |
| `secret.yaml` | Secret | `PGPASSWORD`, `GOOGLE_CLIENT_SECRET`, `SESSION_SECRET` (skipped if `existingSecret` set) |
| `serviceaccount.yaml` | ServiceAccount | |
| `httproute.yaml` | HTTPRoute | Gateway API; off by default (`gateway.enabled=true` to attach to a Gateway) |
| `gateway.yaml` | Gateway | optional (`gateway.create=true`) |
| `ingress.yaml` | Ingress | alternative to Gateway API; off by default (`ingress.enabled=true`) |
| `seed-secret.yaml` | Secret | optional real-data seed (`seed.enabled=true`) |

Non-secret config is mounted as the app's native **YAML file** (the ConfigMap, at
`/etc/kazi-ancestry/kazi-ancestry.yaml`, via `CONFIG_FILE`). The three credentials
come from the Secret as **env vars**, which override the file
(`defaults < config file < env`), so secrets never sit in the ConfigMap.

## Install

Keep secrets out of git — put them in a private values file:

```yaml
# secrets.yaml  (do NOT commit)
database:
  host: postgres.db.svc.cluster.local
  password: <pg-password>
oauth:
  clientId: <google-client-id>
  clientSecret: <google-client-secret>
  sessionSecret: <openssl rand -base64 32>
gateway:
  hostname: family.mrahman.xyz
  parentRefs:
    - name: <your-shared-gateway>
      namespace: <gateway-namespace>
```

```sh
helm upgrade --install kazi charts/kazi-ancestry -n kazi --create-namespace -f secrets.yaml
```

Alternatively pre-create the Secret (keys `PGPASSWORD`, `GOOGLE_CLIENT_SECRET`,
`SESSION_SECRET`) and set `existingSecret: <name>`.

## Key values

- `image.repository` / `image.tag` — defaults to `masudjuly02/kazi-ancestry:<appVersion>`.
- `database.*` — external Postgres connection (host required; password → Secret).
- `oauth.*` — Google OAuth; empty `clientSecret` ⇒ open dev mode (no login wall).
- `oauth.redirectUrl` — defaults to `https://{gateway.hostname}/auth/callback`.
- `config.admins` / `config.allowlist` — admin + contributor emails.
- `config.guestNamesOnly` — privacy: guests see only names when `true`.
- `gateway.parentRefs` — existing Gateway to attach to (or `gateway.create=true` for a dedicated one).
- `ingress.*` — alternative router for non-Gateway-API clusters (off by default).
- `seed.enabled` / `seed.data` — seed real `family.local.json` on first boot (else the in-image sample).

## Seeding

The app **auto-seeds on first `serve` boot** and is idempotent: `Person.Seed` is a
no-op once the table has rows, so restarts/rollouts never re-import. The source is
`SEED_PATH` (`seedPath` in the config file):

- `seed.enabled=false` (default) → seeds the small **in-image sample** (`web/family.json`).
- `seed.enabled=true` → paste the real `family.local.json` into `seed.data`; the chart
  renders it into a Secret, mounts it at `/app/seed/family.local.json`, and points
  `SEED_PATH` there. Real data stays out of the image (in GitOps it lives in the
  SOPS-encrypted values secret).

To **re-seed** after editing names (regenerate ids), run the one-off command in a pod:
`kubectl -n <ns> exec deploy/<release> -- kazi-ancestry seed --reseed`.

The OAuth redirect URI in Google Cloud Console must match `oauth.redirectUrl`.
