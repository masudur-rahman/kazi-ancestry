# Kazi Ancestry · কাজী বংশলতিকা

Interactive website for the Kazi family tree. Vanilla-JS port of the Claude
Design prototype (`Kazi Ancestry.dc.html`) — runs as a plain static site, no
build step, no framework.

## Features

- Three layouts:
  - **Tree** — pan/zoom canvas (drag to pan, wheel/pinch to zoom, `+ / − / fit`), node styles Cards · Medallions · Ledger
  - **Branch** — left→right tree, same pan/zoom canvas; collapsing a node keeps it fixed on screen (no reflow jump)
  - **Columns** — Finder-style lineage explorer; drilled columns persist when the detail panel is closed
- 5 accent colors
- Detail panel: origin, alias, spouse, birth/death, parent, descendants, notes
- Search any name → jump to person (expands the path)
- ✻ = noted in records · *italic* = place of origin
- **Mobile** (`≤760px`): compact header with a labeled hamburger → full-width
  dropdown menu (search, layout, node style, expand/collapse, accent, account).
  Detail panel & review inbox become bottom sheets; in Branch and Columns the
  layout sits *above* the sheet (master–detail, no overlap) and the sheet is
  **drag-resizable** (grab the handle to split the screen, both panes follow).
  Tree supports touch pan + pinch-zoom.

## Roles (auth-driven)

Roles come from who is signed in — there is no manual role switch:

| State | Role | Can |
|-------|------|-----|
| Logged out | **viewer** | browse / search / switch layouts only |
| Signed in | **contributor** | propose edits/adds → go to the review queue as *pending* |
| Signed in + admin code | **admin** (you) | edit/add/delete directly, review & approve/reject suggestions |

Auth is a **localStorage stub**: the admin code is `kazi-admin` (see `auth.ADMIN_CODE`
in `app.js`). Replace the `auth` object with real OAuth + a server-side admin
allowlist when the backend lands.

## Privacy

`lockDown()` in `app.js` applies best-effort deterrents: no right-click, no
copy/selection (form fields excluded), blocked devtools/save/print/view-source
shortcuts, and the page blurs when hidden (app-switcher / screen-share previews).

**These are deterrents, not real protection.** The OS screenshot key and a phone
camera can't be blocked, and devtools can't be reliably disabled. The real leak is
that the whole dataset ships to every client in `data.js` — anyone can read it from
the Network tab regardless of the UI. **True privacy requires the backend**: gate
the data behind auth server-side so unauthorized clients never receive it.

## Layout

```
web/index.html    shell (loads data.js then app.js)
web/app.js        the whole app (vanilla JS)
web/style.css     base styles + fonts (Spectral / Noto Serif Bengali)
web/data.js       window.KAZI_SEED — first-load seed (file:// safe)
web/family.json   same data as a clean JSON artifact (for the future backend)
tools/gen.mjs     curated source tree -> data.js + family.json
```

Data persists in the browser via `localStorage`. `store` in `app.js` has the
`GET/PUT /api/...` seams marked for swapping in a backend later.

## Regenerate data

Edit the curated tree in `tools/gen.mjs`, then:

```sh
node tools/gen.mjs
```

## Run locally

```sh
python3 tools/serve.py        # http://localhost:8000, no-cache (edits show on reload)
```

Or plain `cd web && python3 -m http.server 8000` (caches assets — hard-reload after
edits). Opening `web/index.html` directly (`file://`) also works — the seed is
embedded in `data.js`, no fetch required.

## Docker

Static site served by **nginx** (listens on port 80). `nginx:alpine` is multi-arch,
so the same Dockerfile builds for amd64 or arm64.

```sh
cp .env.example .env
```

### Run locally (on the Mac)

```sh
PLATFORM=linux/arm64 docker compose up -d --build   # native on Apple silicon
# open http://localhost:8080
```

### Build on Mac (arm64) → run on a Debian VM

Cross-build for the VM's architecture and export a loadable tar:

```sh
tools/build.sh                      # default linux/amd64 -> kazi-ancestry-amd64.tar
# or: PLATFORM=linux/arm64 tools/build.sh
```

Copy it over and run on the VM:

```sh
scp kazi-ancestry-amd64.tar user@vm:~
ssh user@vm 'docker load -i kazi-ancestry-amd64.tar \
  && docker run -d --restart unless-stopped -p 80:80 --name kazi-ancestry kazi-ancestry:latest'
```

Point the domain's A record at the VM and open it at `http://your-domain` (port 80).
Cross-build uses buildx + QEMU, which ship with Docker Desktop.

> TLS/HTTPS is intentionally not in the container — terminate it on the VM
> (host nginx, Cloudflare, etc.) in front of port 80.

Rebuild after editing `web/`: re-run `tools/build.sh` (VM) or `docker compose up -d --build` (local).

## Roadmap

- [x] Static interactive tree (Tree / Branch / Columns)
- [x] Suggestion + admin approval flow (currently localStorage)
- [x] Auth-driven roles (viewer / contributor / admin) — stubbed
- [ ] Go backend + persistent store (replace the `store` adapter in `app.js`)
- [ ] Real auth replacing the `auth` stub (OAuth + admin allowlist)
```
