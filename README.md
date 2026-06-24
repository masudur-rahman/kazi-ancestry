# Kazi Ancestry · কাজী বংশলতিকা

Interactive website for the Kazi family tree: a vanilla-JS pan/zoom canvas served
by a small Go backend over Postgres, with Google OAuth and server-side rendering
of the initial state.

## Features

- Three layouts:
  - **Tree** — pan/zoom canvas (drag to pan, wheel/pinch to zoom, `+ / − / fit`), node styles Cards · Medallions · Ledger
  - **Branch** — left→right tree on the same canvas; collapsing a node keeps it fixed on screen (no reflow jump)
  - **Columns** — Finder-style lineage explorer
- Detail panel: origin, alias, spouse, birth/death, parent, descendants, notes, tags
- Search any name → jump to person (expands the path)
- ✻ superscript marks a person tagged in records (extensible `tags` registry)
- Mobile (`≤760px`): floating on-canvas controls, detail/inbox as bottom sheets, touch pan + pinch-zoom

## Architecture

The Go server delivers data only to authenticated sessions and never exposes a
bulk data endpoint — the tree is **server-injected** into the page, not fetched.

```
main.go                 cmd.Execute()
cmd/                    cobra: serve, seed (--reseed)
api/web/                chi router, SSR page injection, JSON mutation API, auth handlers + middleware
configs/                env config + Postgres connection / schema sync
modules/auth/           Google OAuth + signed-cookie sessions
models/                 Person / User / Suggestion (+ TableName, typed errors)
repos/                  data-access interfaces (+ repos/<entity> SQL impls over styx)
services/               business-logic interfaces (+ services/<entity> impls, services/all DI)
pkg/slug/               Bengali→Latin slug ids (shortest-unique)
infra/logr/             zap logger
web/                    index.html (shell), app.js (the SPA), style.css, family.json (sample seed)
```

- **Data delivery:** `GET /` injects the tree as `<script type="application/json">` for authenticated requests only. Anonymous visitors get a **login wall** with no data in the page.
- **Mutations:** `POST/PUT/DELETE /api/v1/people` (admin), `POST /api/v1/suggestions` (any signed-in user), inbox approve/reject (admin).
- **Ids** are short readable slugs derived from names (`তাহের আলী কাজী` → `taher`), assigned once and stable; new people are slugged server-side.

## Roles

Roles come from the OAuth allowlist (`ALLOWLIST` / `ADMIN_EMAILS`):

| State | Role | Can |
|-------|------|-----|
| Logged out | — | nothing (login wall, no data served) |
| Allowlisted | **contributor** | browse + propose edits/adds → review queue |
| Admin email | **admin** | edit/add/delete directly, review & approve/reject suggestions |

When `GOOGLE_CLIENT_ID`/`SECRET` are unset the server runs in **open dev mode**
(no wall, every request is admin) so it is usable without credentials.

## Run locally

Needs Postgres. With a local server running:

```sh
# init the tree (idempotent); reads web/family.local.json, else the sample
go run . seed
# serve
go run . serve            # http://localhost:5294
```

Config precedence is **defaults < `.configs/.kazi-ancestry.yaml` < environment**.
Put settings in the YAML (gitignored — safe for secrets; copy from
`.configs/.kazi-ancestry.yaml.example`), or pass credentials via env, which
overrides the file. Override the path with `--config`.

### Enable Google OAuth (minimal)

1. Google Cloud Console → APIs & Services → **Credentials** → Create **OAuth client ID** → *Web application*.
2. Add the redirect URI: `http://localhost:5294/auth/callback` (and your prod `https://…/auth/callback`).
3. In `.configs/.kazi-ancestry.yaml`, set `auth.googleClientId` / `auth.googleClientSecret` (or env
   `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`), a random `auth.sessionSecret`, your `auth.admins`,
   and the `auth.allowlist` emails allowed to suggest.

Leaving the Google fields empty runs in **open dev mode** (no login wall, every request is admin).
Logged-out visitors can always view the tree read-only; logging in is required to suggest
(allowlisted) or resolve (admin).

## Data

- `web/family.json` — committed **fictional sample**, the seed fallback.
- `web/family.local.json` — your **real data** (gitignored, never committed or baked into the image). When present it overrides the sample.
- The server seeds the DB on first boot when empty. To regenerate ids after editing names:

```sh
go run . seed --reseed    # clears the person table and reimports with fresh ids
```

## Docker

`docker compose` provisions Postgres + the Go app (multi-stage static build).

```sh
cp .env.example .env      # set PGPASSWORD, Google OAuth, SESSION_SECRET, allowlist
docker compose up -d --build
# open http://localhost:8080
```

The image ships the SPA + the sample seed only. To run real data, mount your
`web/family.local.json` (see the commented volume in `docker-compose.yml`) and
set `SEED_PATH`. Terminate TLS in front of the container (host nginx, Cloudflare,
etc.); set `OAUTH_REDIRECT_URL` to the public `https://…/auth/callback`.

## Roadmap

- [x] Go backend + Postgres (styx) replacing the static/localStorage store
- [x] SSR state injection — no public data endpoint
- [x] Google OAuth login wall with allowlist roles
- [ ] Apply approved suggestions transactionally; audit trail
- [ ] Convert repo to the expense-tracker-bot conventions (Makefile/Docker/CI/lint, viper config)
