# kazi-ancestry Helm chart

Deploys the Kazi Ancestry server (Go + Postgres, Gateway API) to Kubernetes.
**Postgres is not provisioned** — point the chart at an existing database.

## Resources

| Template | Resource | Notes |
|----------|----------|-------|
| `deployment.yaml` | Deployment | runs `kazi-ancestry serve`; `/healthz` probes; read-only rootfs |
| `service.yaml` | Service | ClusterIP → container port 5294 |
| `configmap.yaml` | ConfigMap | non-secret env (DB host, OAuth client id, redirect URL, admins, allowlist, privacy) |
| `secret.yaml` | Secret | `PGPASSWORD`, `GOOGLE_CLIENT_SECRET`, `SESSION_SECRET` (skipped if `existingSecret` set) |
| `serviceaccount.yaml` | ServiceAccount | |
| `httproute.yaml` | HTTPRoute | Gateway API; attaches to an existing or chart-created Gateway |
| `gateway.yaml` | Gateway | optional (`gateway.create=true`) |
| `seed-secret.yaml` | Secret | optional real-data seed (`seed.enabled=true`) |

Config is delivered as **env vars**; the app reads them over its built-in defaults
(`defaults < .configs yaml < env`), so no config file is mounted.

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
- `seed.enabled` / `seed.data` — seed real `family.local.json` on first boot (else the in-image sample).

The OAuth redirect URI in Google Cloud Console must match `oauth.redirectUrl`.
