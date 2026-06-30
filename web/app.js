"use strict";
/*
 * Kazi Ancestry — vanilla port of the Claude Design prototype (Kazi Ancestry.dc.html).
 * Layouts: Tree (pan/zoom canvas) · Branch (left→right) · Columns (Finder-style).
 * Detail panel, search, suggestion inbox (approve/reject), add/edit/delete, export/import.
 *
 * Roles are auth-driven (NOT a manual toggle):
 *   - logged out          -> viewer      (read-only)
 *   - logged in           -> contributor (edits become pending suggestions)
 *   - logged in + admin   -> reviewer    (direct edit/add/delete + review inbox + export/import)
 * Auth here is a localStorage stub; swap `auth` for real OAuth + an admin allowlist later.
 */

// ---- tiny DOM helper -------------------------------------------------------
const SKIP = { onClick: 1, onInput: 1, onChange: 1, onMouseDown: 1, onCompositionStart: 1, onCompositionEnd: 1, ref: 1, style: 1 };
function camel(k) { return k.replace(/[A-Z]/g, (m) => "-" + m.toLowerCase()); }
const SVG_NS = "http://www.w3.org/2000/svg";
const SVG_TAGS = { svg: 1, g: 1, path: 1, line: 1, polyline: 1, polygon: 1, circle: 1, rect: 1 };
function h(tag, props, ...kids) {
  const el = SVG_TAGS[tag] ? document.createElementNS(SVG_NS, tag) : document.createElement(tag);
  if (props) {
    if (props.style) el.style.cssText = Object.keys(props.style).map((k) => camel(k) + ":" + props.style[k]).join(";");
    if (props.onClick) el.addEventListener("click", props.onClick);
    if (props.onInput) el.addEventListener("input", props.onInput);
    if (props.onChange) el.addEventListener("change", props.onChange);
    if (props.onMouseDown) el.addEventListener("mousedown", props.onMouseDown);
    if (props.onCompositionStart) el.addEventListener("compositionstart", props.onCompositionStart);
    if (props.onCompositionEnd) el.addEventListener("compositionend", props.onCompositionEnd);
    if (props.ref) props.ref(el);
    for (const k in props) {
      if (SKIP[k]) continue;
      if (k === "value") el.value = props[k];
      else if (k === "checked") el.checked = !!props[k];
      else if (props[k] != null && props[k] !== false) el.setAttribute(k, props[k]);
    }
  }
  for (const kid of kids.flat()) {
    if (kid == null || kid === false) continue;
    el.appendChild(kid.nodeType ? kid : document.createTextNode(String(kid)));
  }
  return el;
}
function hover(el, css) {
  const base = el.style.cssText;
  el.addEventListener("mouseenter", () => (el.style.cssText = base + ";" + css));
  el.addEventListener("mouseleave", () => (el.style.cssText = base));
}

// ---- Bengali -> Latin romanizer (mirrors pkg/slug/slug.go) ------------------
// Best-effort phonetic map so an English query ("masud") matches Bengali names
// ("মাসুদ", "মাসুদুর রহমান"). Output is lower-cased, tokens space-joined.
const BN_VOWEL = { "অ": "o", "আ": "a", "ই": "i", "ঈ": "i", "উ": "u", "ঊ": "u", "ঋ": "ri", "এ": "e", "ঐ": "oi", "ও": "o", "ঔ": "ou" };
const BN_KAR = { "া": "a", "ি": "i", "ী": "i", "ু": "u", "ূ": "u", "ৃ": "ri", "ে": "e", "ৈ": "oi", "ো": "o", "ৌ": "ou" };
const BN_CONS = {
  "ক": "k", "খ": "kh", "গ": "g", "ঘ": "gh", "ঙ": "ng",
  "চ": "ch", "ছ": "chh", "জ": "j", "ঝ": "jh", "ঞ": "n",
  "ট": "t", "ঠ": "th", "ড": "d", "ঢ": "dh", "ণ": "n",
  "ত": "t", "থ": "th", "দ": "d", "ধ": "dh", "ন": "n",
  "প": "p", "ফ": "f", "ব": "b", "ভ": "bh", "ম": "m",
  "য": "j", "র": "r", "ল": "l", "শ": "sh", "ষ": "sh", "স": "s", "হ": "h",
  "ৎ": "t", "য়": "y", "ড়": "r", "ঢ়": "rh",
  "Ɏ": "y", "Ɍ": "r", "Ʀ": "rh",
};
const BN_VIRAMA = "্", BN_NUKTA = "়", BN_INHERENT = "o";
const BN_NUKFOLD = { "য": "Ɏ", "জ": "Ɏ", "ড": "Ɍ", "ঢ": "Ʀ" };
const BN_SKIP = { "ঁ": 1, "ং": 1, "ঃ": 1, "‌": 1, "‍": 1 };
const BN_OVERRIDE = {
  "কাজী": "kazi", "আলী": "ali", "আলি": "ali",
  "ময়না": "moyna", "নজরুল": "nojrul", "জাহাঙ্গীর": "jahangir",
  "স্বপন": "swapon", "উজ্জ্বল": "ujjal", "ইউসুফ": "yusuf", "ইয়াসিন": "yasin",
};
function bnToken(tok) {
  if (BN_OVERRIDE[tok]) return BN_OVERRIDE[tok];
  const ch = [];
  for (const r of tok) {
    if (r === BN_NUKTA && ch.length && BN_NUKFOLD[ch[ch.length - 1]]) { ch[ch.length - 1] = BN_NUKFOLD[ch[ch.length - 1]]; continue; }
    ch.push(r);
  }
  let out = "";
  for (let i = 0; i < ch.length; i++) {
    const c = ch[i];
    if (BN_SKIP[c]) continue;
    if (BN_VOWEL[c]) { out += BN_VOWEL[c]; continue; }
    if (BN_KAR[c]) { out += BN_KAR[c]; continue; }
    if (BN_CONS[c]) {
      out += BN_CONS[c];
      if (i + 1 < ch.length) {
        const nxt = ch[i + 1];
        if (nxt === BN_VIRAMA) { i++; continue; }   // conjunct: drop inherent, consume virama
        if (BN_KAR[nxt]) continue;                  // explicit vowel follows
      }
      let last = true;
      for (let j = i + 1; j < ch.length; j++) { if (!BN_SKIP[ch[j]]) { last = false; break; } }
      // schwa deletion: drop the inherent o when this consonant sits between an
      // explicit vowel and an "open" next consonant (mirrors pkg/slug/slug.go).
      if (!last && !(bnPrevVowel(ch, i) && bnNextConsOpen(ch, i))) out += BN_INHERENT;
    }
  }
  return out;
}
// nearest sounded rune before i is an explicit vowel (independent vowel or kar)
function bnPrevVowel(ch, i) {
  for (let j = i - 1; j >= 0; j--) {
    if (BN_SKIP[ch[j]]) continue;
    return !!(BN_KAR[ch[j]] || BN_VOWEL[ch[j]]);
  }
  return false;
}
// next sounded consonant after i is "open" — immediately followed (past skips) by a kar
function bnNextConsOpen(ch, i) {
  let nj = -1;
  for (let j = i + 1; j < ch.length; j++) { if (!BN_SKIP[ch[j]]) { nj = j; break; } }
  if (nj < 0) return false;
  for (let j = nj + 1; j < ch.length; j++) { if (BN_SKIP[ch[j]]) continue; return !!BN_KAR[ch[j]]; }
  return false;
}
function bnRomanize(name) {
  return (name || "").split(/\s+/).map((t) => bnToken(t).toLowerCase()).filter(Boolean).join(" ");
}
const HAS_LATIN = /[a-z]/i;

// ---- store (localStorage; now only backs the client-side auth stub) --------
// People and suggestions are server-owned: the tree arrives via the injected
// page bootstrap, and mutations go through the /api/v1 endpoints.
const store = {
  read(k, f) { try { const v = localStorage.getItem(k); return v ? JSON.parse(v) : f; } catch (e) { return f; } },
  write(k, v) { try { localStorage.setItem(k, JSON.stringify(v)); } catch (e) {} },
  del(k) { try { localStorage.removeItem(k); } catch (e) {} },
};

// ---- auth stub -------------------------------------------------------------
// Replace with real auth: viewer = not signed in, contributor = signed in,
// admin = signed-in user on a server-side allowlist. ADMIN_CODE is a placeholder.
const auth = {
  K: "kazi.auth.v1",
  ADMIN_CODE: "kazi-admin",
  get() { return store.read(this.K, null); },     // -> { name, role } | null
  set(u) { store.write(this.K, u); },
  clear() { store.del(this.K); },
};

const FIELDS = ["name", "origin", "alias", "spouse", "birth", "death", "note"];

const App = {
  ACCENTS: ["#9c4326", "#7a5230", "#5c6b4a", "#6d4636", "#8a6d2f"],
  // person.tags drives the small marks after a name. Add more entries here to show
  // a different glyph for a different tag — no schema/field changes needed.
  TAGS: { died_young: { mark: "✻", color: "#9c4326", label: "অল্প বয়সে মৃত্যু" } },

  // Reads the server-injected initial state from the page (no data fetch). The
  // tree is embedded only for authorized requests, so there is no data endpoint.
  _loadBootstrap() {
    const el = document.getElementById("kazi-bootstrap");
    if (!el) return { people: [], user: null, suggestions: [] };
    try { return JSON.parse(el.textContent) || {}; } catch (e) { return { people: [], user: null, suggestions: [] }; }
  },

  // _api wraps the JSON mutation API. Returns parsed body (or null), throws on !ok.
  _api(method, path, body) {
    return fetch("/api/v1" + path, {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
      credentials: "same-origin",
    }).then((r) => {
      if (!r.ok) throw new Error(method + " " + path + " -> " + r.status);
      return r.status === 204 ? null : r.json().catch(() => null);
    });
  },
  _apiErr(e) { console.error(e); this.toast("সার্ভার ত্রুটি"); },

  // Maps a server suggestion row back to the rich client-side suggestion object.
  _sugFromRow(row) {
    let s = {};
    try { s = JSON.parse(row.payload) || {}; } catch (e) {}
    s.id = row.id; s.status = row.status;
    if (row.submittedBy) s.by = row.submittedBy;
    return s;
  },

  init() {
    const boot = this._loadBootstrap();
    this.people = (boot.people || []);
    this.people.forEach((p) => { if (!Array.isArray(p.tags)) p.tags = []; });
    this.rebuild();

    const expanded = {};
    this.people.forEach((p) => { if (this.depthOf[p.id] <= 3 && this.childrenOf[p.id].length) expanded[p.id] = true; });

    this._topbarH = 64;
    this._w = window.innerWidth;
    this.state = {
      expanded, selectedId: null, query: "",
      tx: 80, ty: 60, scale: 0.78,
      layout: "tree", variant: "card", accent: this.ACCENTS[0],
      user: boot.user || null,
      openSuggestions: !!boot.openSuggestions,
      suggestions: (boot.suggestions || []).map((row) => this._sugFromRow(row)),
      modal: null, signin: null, form: this.blankForm(), reorder: null,
      showInbox: false, inboxTab: "pending", showMine: false, mine: null,
      toast: "", menu: false, search: false, colSel: null,
      panelH: Math.max(200, Math.round(window.innerHeight * 0.42)), // mobile sheet height (drag-resizable)
    };

    this.root = document.getElementById("app");
    this._pmove = (e) => this.onPanelResizeMove(e);
    this._pend = () => this.onPanelResizeEnd();
    window.addEventListener("mousemove", (e) => this._onMove(e));
    // record whether the gesture moved (a pan) BEFORE clearing _pan, so the
    // canvas click handler can tell a tap (close overlays) from a drag.
    window.addEventListener("mouseup", () => { this._dragMoved = !!(this._pan && this._pan.moved); this._pan = null; });
    window.addEventListener("resize", () => { const m = this.isMobile(); this._w = window.innerWidth; if (m !== this.isMobile()) this.render(); else this.measureChrome(); });
    this.lockDown();
    this.render();
    setTimeout(() => this.focusPerson(this.rootId, true), 80);
  },

  isMobile() { return (this._w || window.innerWidth) <= 760; },

  // Best-effort privacy deterrents (NOT real security — see README). Blocks
  // right-click, copy/cut/selection, drag, and the common devtools/save/print/
  // view-source shortcuts.
  lockDown() {
    const inField = (e) => { const t = e.target; return t && (t.tagName === "INPUT" || t.tagName === "TEXTAREA"); };
    const stop = (e) => { e.preventDefault(); e.stopPropagation(); return false; };
    document.addEventListener("contextmenu", stop);
    document.addEventListener("dragstart", stop);
    ["copy", "cut", "selectstart"].forEach((ev) => document.addEventListener(ev, (e) => { if (!inField(e)) stop(e); }));
    document.addEventListener("keydown", (e) => {
      const k = (e.key || "").toLowerCase(), mod = e.ctrlKey || e.metaKey;
      if (mod && inField(e) && ["c", "x", "v", "a", "z"].includes(k)) return; // allow normal editing in fields
      if (e.key === "F12") return stop(e);
      if (mod && e.shiftKey && (k === "i" || k === "j" || k === "c")) return stop(e); // devtools
      if (mod && (k === "u" || k === "s" || k === "p" || k === "c" || k === "a")) return stop(e); // source/save/print/copy/select-all
      if (e.key === "PrintScreen") { try { navigator.clipboard && navigator.clipboard.writeText(""); } catch (x) {} return stop(e); }
    });
  },

  setState(patch) { Object.assign(this.state, patch); this.render(); },

  // IME-safe input handlers: avoid rebuilding the DOM mid-composition (Avro/phonetic),
  // which would otherwise destroy the input node and reset the keyboard each keystroke.
  // commit(value) stores the model value; we only re-render once composition is done.
  _ime(commit, noRender) {
    return {
      onInput: (e) => { commit(e.target.value); if (!noRender && !(e.isComposing || this._composing)) this.render(); },
      onCompositionStart: () => { this._composing = true; },
      onCompositionEnd: (e) => { this._composing = false; commit(e.target.value); if (!noRender) this.render(); },
    };
  },

  rebuild() {
    const byId = {}, children = {};
    this.people.forEach((p) => { byId[p.id] = p; children[p.id] = []; });
    this.people.forEach((p) => { if (p.parentId != null && children[p.parentId]) children[p.parentId].push(p.id); });
    // order siblings by their stored position; Array.sort is stable, so ties
    // (e.g. un-reseeded rows all at 0) keep their original insertion order
    Object.keys(children).forEach((pid) => { children[pid].sort((a, b) => (byId[a].position || 0) - (byId[b].position || 0)); });
    const root = this.people.find((p) => p.parentId == null) || this.people[0];
    const depth = {}, desc = {};
    const dfs = (id, d) => { depth[id] = d; let t = 0; children[id].forEach((c) => { t += 1 + dfs(c, d + 1); }); desc[id] = t; return t; };
    dfs(root.id, 0);
    this.byId = byId; this.childrenOf = children; this.depthOf = depth; this.descOf = desc;
    this.rootId = root.id;
    this.maxDepth = Math.max.apply(null, Object.keys(depth).map((k) => depth[k]));
  },

  // ---- roles ----
  role() { return this.state.user ? this.state.user.role : "viewer"; },
  isAdmin() { return this.role() === "admin"; },
  isContrib() { return this.role() === "contributor"; },
  // Any logged-in user may suggest when openSuggestions is on (server-gated too).
  canSuggest() { return !!this.state.user && (this.isContrib() || this.state.openSuggestions); },
  canAct() { return this.isAdmin() || this.canSuggest(); },

  // ---- mutations ----
  // commit re-derives indices and re-renders. The server persists each change via
  // its own endpoint (optimistic UI: apply locally now, sync in the background).
  commit() { this.rebuild(); this.render(); },
  applyEdit(id, fields) {
    Object.assign(this.byId[id], fields); this.commit();
    this._api("PUT", "/people/" + encodeURIComponent(id), this.byId[id]).catch((e) => this._apiErr(e));
  },
  // addPerson inserts with a temporary id, then adopts the server-assigned slug id
  // on response (see _reId). Returns the temp id so the caller can select/focus it.
  addPerson(parentId, fields) {
    const tmp = "tmp-" + Date.now();
    const np = Object.assign({ id: tmp, parentId: parentId }, this.sanitize(fields));
    this.people.push(np); this.commit();
    this._api("POST", "/people", { parentId: parentId, name: np.name, origin: np.origin, alias: np.alias, spouse: np.spouse, birth: np.birth, death: np.death, note: np.note, tags: np.tags })
      .then((srv) => { if (srv && srv.id) this._reId(tmp, srv); })
      .catch((e) => this._apiErr(e));
    return tmp;
  },
  // ---- sibling reorder (staged draft) ----
  // Reorder is a draft: ←/→ rearrange the live sibling order (tree previews it)
  // without persisting. The admin then commits with "সম্পন্ন", or a contributor
  // sends a single proposal — nothing is saved per keystroke.
  startReorder(id) {
    const p = this.byId[id]; if (!p || p.parentId == null) return;
    const order = (this.childrenOf[p.parentId] || []).slice();
    this.setState({ reorder: { parentId: p.parentId, focusId: id, orig: order } });
  },
  reorderFocus(id) { if (this.state.reorder) { this.state.reorder.focusId = id; this.render(); } },
  // moveSibling shifts the focused sibling one slot in the draft (dir -1 = earlier,
  // +1 = later) and re-renders so the tree shows the change immediately.
  moveSibling(dir) {
    const r = this.state.reorder; if (!r) return;
    const arr = this.childrenOf[r.parentId];
    const i = arr.indexOf(r.focusId), j = i + dir;
    if (i < 0 || j < 0 || j >= arr.length) return;
    arr.splice(i, 1); arr.splice(j, 0, r.focusId);
    this.render();
  },
  // reorderCancel discards the draft, restoring the original sibling order.
  reorderCancel() { const r = this.state.reorder; if (r) this.childrenOf[r.parentId] = r.orig.slice(); this.setState({ reorder: null }); },
  // reorderDone commits the draft: admin persists immediately, contributor sends
  // one proposal carrying the full before/after order.
  reorderDone() {
    const r = this.state.reorder; if (!r) return;
    const order = (this.childrenOf[r.parentId] || []).slice();
    if (!order.some((id, i) => id !== r.orig[i])) { this.setState({ reorder: null }); return; } // unchanged
    if (this.isAdmin()) { this.reorderSiblings(r.parentId, order); this.toast("ক্রম পরিবর্তন করা হয়েছে"); }
    else { this.submitSuggestion({ type: "reorder", parentId: r.parentId, parentName: this.byId[r.parentId].name, order: order, before: r.orig.slice() }); this.toast("প্রস্তাব পাঠানো হয়েছে"); }
    this.setState({ reorder: null });
  },
  // reorderSiblings applies a new sibling order optimistically, then persists it.
  reorderSiblings(parentId, order) {
    order.forEach((cid, idx) => { if (this.byId[cid]) this.byId[cid].position = idx; });
    this.commit();
    this._api("POST", "/people/reorder", { parentId: parentId, order: order }).catch((e) => this._apiErr(e));
  },
  deletePerson(id) {
    if (this.childrenOf[id].length) return false;
    this.people = this.people.filter((p) => p.id !== id); this.commit();
    this._api("DELETE", "/people/" + encodeURIComponent(id)).catch((e) => this._apiErr(e));
    return true;
  },
  // _reId swaps a temp add id for the server's canonical slug id across people,
  // child parent refs, expansion, and selection.
  _reId(tmp, srv) {
    const p = this.byId[tmp]; if (!p) return;
    Object.assign(p, srv);
    this.people.forEach((c) => { if (c.parentId === tmp) c.parentId = srv.id; });
    if (this.state.expanded[tmp]) { this.state.expanded[srv.id] = this.state.expanded[tmp]; delete this.state.expanded[tmp]; }
    if (this.state.selectedId === tmp) this.state.selectedId = srv.id;
    this.commit();
  },
  sanitize(f) {
    const t = (v) => (typeof v === "string" ? v.trim() : v);
    return { name: t(f.name) || "অজানা", origin: t(f.origin) || "", alias: t(f.alias) || "", spouse: t(f.spouse) || "", birth: t(f.birth) || "", death: t(f.death) || "", note: t(f.note) || "", tags: Array.isArray(f.tags) ? f.tags.filter((x) => this.TAGS[x]) : [] };
  },
  blankForm() { return { name: "", origin: "", alias: "", spouse: "", birth: "", death: "", note: "", tags: [] }; },
  formFrom(p) { return { name: p.name, origin: p.origin || "", alias: p.alias || "", spouse: p.spouse || "", birth: p.birth || "", death: p.death || "", note: p.note || "", tags: Array.isArray(p.tags) ? p.tags.slice() : [] }; },
  setForm(k, v) { this.state.form = Object.assign({}, this.state.form, { [k]: v }); /* no re-render: keep focus */ },

  toast(msg) { this.setState({ toast: msg }); if (this._toastT) clearTimeout(this._toastT); this._toastT = setTimeout(() => this.setState({ toast: "" }), 2600); },

  // ---- auth flows ----
  // ---- tags ----
  hasTag(p, t) { return !!(p && Array.isArray(p.tags) && p.tags.indexOf(t) !== -1); },
  _migratePerson(p) { if (!Array.isArray(p.tags)) p.tags = []; if (p.star && p.tags.indexOf("died_young") === -1) p.tags.push("died_young"); if ("star" in p) delete p.star; return p; },
  _setFormTag(t, on) { const cur = Array.isArray(this.state.form.tags) ? this.state.form.tags.slice() : []; const i = cur.indexOf(t); if (on && i === -1) cur.push(t); else if (!on && i !== -1) cur.splice(i, 1); this.setForm("tags", cur); },
  // small superscript mark(s) after a name in the tree, one per known tag
  _marks(p) { return Array.isArray(p.tags) ? p.tags.map((t) => { const d = this.TAGS[t]; return d ? h("sup", { style: { "font-size": ".7em", color: d.color, "margin-left": "1px" } }, d.mark) : null; }) : []; },
  _nm(p) { return [p.name].concat(this._marks(p)); },
  // small, muted tag markers at the bottom of the detail panel — the ✻ ties each
  // back to the mark shown on the name in the tree; shaped to read as a label, not a button
  _tagChips(p) {
    if (!Array.isArray(p.tags) || !p.tags.length) return null;
    const chips = p.tags.map((t) => { const d = this.TAGS[t]; return d ? h("span", { style: { display: "inline-flex", "align-items": "center", gap: "4px", "font-size": "11px", color: "#9c6a52", background: "#f3e8df", "border-radius": "5px", padding: "3px 8px" } }, h("span", { style: { color: "#9c4326" } }, d.mark), d.label) : null; });
    return h("div", { style: { display: "flex", "flex-wrap": "wrap", gap: "6px", "margin-top": "16px" } }, chips);
  },
  // clickable name pill used for both the parent row and the children list
  _personPill(pid) { const b = h("button", { onClick: () => this.goTo(pid), style: { padding: "7px 13px", "font-size": "14px", border: "1px solid #d4c096", "border-radius": "20px", background: "#fdf9ee", color: "#3b2f21", cursor: "pointer" } }, this.byId[pid].name); hover(b, "background:#f1e6cb;border-color:#9c4326"); return b; },
  // Auth is server-side (Google OAuth). Login redirects to the OAuth flow;
  // logout clears the server session cookie, then reloads so the server
  // re-renders the login wall (no tree data for anonymous).
  login() { window.location.href = "/auth/login"; },
  signOut() {
    // /auth/logout is a top-level route, not under /api/v1 — call it directly
    // (not via _api, which prefixes /api/v1) so the server actually clears the session.
    fetch("/auth/logout", { method: "POST", credentials: "same-origin" })
      .catch(() => {}).then(() => { auth.clear(); window.location.reload(); });
  },

  // ---- modal ----
  onEdit() { if (!this.canAct()) return; const p = this.byId[this.state.selectedId]; if (!p) return; this.setState({ modal: { kind: "edit", target: p.id, asSuggestion: !this.isAdmin() }, form: this.formFrom(p) }); },
  onAddChild() { if (!this.canAct()) return; const id = this.state.selectedId; this.setState({ modal: { kind: "add", parentId: id, asSuggestion: !this.isAdmin() }, form: this.blankForm() }); },
  onAddSibling() { if (!this.canAct()) return; const p = this.byId[this.state.selectedId]; if (!p || p.parentId == null) return; this.setState({ modal: { kind: "add", parentId: p.parentId, asSuggestion: !this.isAdmin() }, form: this.blankForm() }); },
  onModalCancel() { this.setState({ modal: null }); },

  onModalSave() {
    const m = this.state.modal, f = this.state.form;
    if (!m) return;
    if (m.kind === "edit") {
      const p = this.byId[m.target];
      if (m.asSuggestion) {
        const changes = {}, before = {};
        FIELDS.forEach((k) => { const nv = typeof f[k] === "string" ? f[k].trim() : f[k]; if (nv !== (p[k] || (typeof f[k] === "boolean" ? false : ""))) { changes[k] = nv; before[k] = p[k]; } });
        if (Object.keys(changes).length === 0) { this.toast("কোনো পরিবর্তন নেই"); this.setState({ modal: null }); return; }
        this.submitSuggestion({ type: "edit", targetId: p.id, targetName: p.name, changes, before });
        this.toast("প্রস্তাব পাঠানো হয়েছে");
      } else { this.applyEdit(m.target, this.sanitize(f)); this.toast("সংরক্ষণ করা হয়েছে"); }
    } else {
      if (!f.name || !f.name.trim()) { this.toast("নাম দিতে হবে"); return; }
      const parentName = this.byId[m.parentId] ? this.byId[m.parentId].name : "";
      if (m.asSuggestion) { this.submitSuggestion({ type: "add", parentId: m.parentId, parentName, fields: this.sanitize(f) }); this.toast("প্রস্তাব পাঠানো হয়েছে"); }
      else { const nid = this.addPerson(m.parentId, f); this.state.expanded[m.parentId] = true; this.setState({ selectedId: nid }); requestAnimationFrame(() => this.focusPerson(nid, false)); this.toast("যোগ করা হয়েছে"); }
    }
    this.setState({ modal: null });
  },

  onDelete() {
    if (!this.isAdmin()) return;
    const id = this.state.selectedId, p = this.byId[id]; if (!p) return;
    if (this.childrenOf[id].length) { this.toast("আগে সন্তানদের সরান"); return; }
    if (window.confirm(p.name + " কে গাছ থেকে মুছে ফেলবেন?")) { const parent = p.parentId; this.deletePerson(id); this.setState({ selectedId: parent }); this.toast("মুছে ফেলা হয়েছে"); }
  },

  // ---- suggestions ----
  submitSuggestion(s) {
    s.at = Date.now();
    s.by = (this.state.user && this.state.user.name) || "Anonymous"; s.status = "pending";
    const personId = String(s.targetId || s.parentId || "");
    this._api("POST", "/suggestions", { personId: personId, payload: JSON.stringify(s) })
      .then((row) => { if (row && row.id) { s.id = row.id; this.setState({ suggestions: this.state.suggestions.concat([s]) }); } })
      .catch((e) => this._apiErr(e));
  },
  setStatus(sid, status) {
    const list = this.state.suggestions.map((s) => (s.id === sid ? Object.assign({}, s, { status }) : s));
    this.setState({ suggestions: list });
    const verb = status === "approved" ? "approve" : "reject";
    this._api("POST", "/suggestions/" + encodeURIComponent(sid) + "/" + verb).catch((e) => this._apiErr(e));
  },
  approve(s) { if (s.type === "edit" && this.byId[s.targetId]) this.applyEdit(s.targetId, s.changes); else if (s.type === "add" && this.byId[s.parentId]) this.addPerson(s.parentId, s.fields); else if (s.type === "reorder" && this.byId[s.parentId]) this.reorderSiblings(s.parentId, s.order); this.setStatus(s.id, "approved"); this.toast("অনুমোদন করা হয়েছে"); },
  reject(s) { this.setStatus(s.id, "rejected"); this.toast("প্রত্যাখ্যান করা হয়েছে"); },
  toggleInbox() {
    const opening = !this.state.showInbox;
    // detail panel and the drawers are mutually exclusive (same screen real estate)
    if (opening && this.state.reorder) this.reorderCancel();
    this.setState({ showInbox: opening, showMine: false, selectedId: opening ? null : this.state.selectedId });
    // Refresh the inbox from the server when an admin opens it.
    if (opening && this.isAdmin()) {
      this._api("GET", "/suggestions")
        .then((rows) => { if (Array.isArray(rows)) this.setState({ suggestions: rows.map((r) => this._sugFromRow(r)) }); })
        .catch((e) => this._apiErr(e));
    }
  },
  // toggleMine opens a contributor's own-suggestions panel, lazy-loading from the server.
  toggleMine() {
    const opening = !this.state.showMine;
    if (opening && this.state.reorder) this.reorderCancel();
    this.setState({ showMine: opening, showInbox: false, selectedId: opening ? null : this.state.selectedId });
    if (opening && this.canSuggest()) {
      this.setState({ mine: null });
      this._api("GET", "/suggestions/mine")
        .then((rows) => { this.setState({ mine: Array.isArray(rows) ? rows.map((r) => this._sugFromRow(r)) : [] }); })
        .catch((e) => { this._apiErr(e); this.setState({ mine: [] }); });
    }
  },

  // ---- colors ----
  hx(hex) { hex = hex.replace("#", ""); if (hex.length === 3) hex = hex.split("").map((c) => c + c).join(""); return [parseInt(hex.slice(0, 2), 16), parseInt(hex.slice(2, 4), 16), parseInt(hex.slice(4, 6), 16)]; },
  hexA(hex, a) { const c = this.hx(hex); return "rgba(" + c[0] + "," + c[1] + "," + c[2] + "," + a + ")"; },
  lighten(hex, amt) { const c = this.hx(hex), f = (v) => Math.round(v + (255 - v) * amt); return "rgb(" + f(c[0]) + "," + f(c[1]) + "," + f(c[2]) + ")"; },
  darken(hex, amt) { const c = this.hx(hex), f = (v) => Math.round(v * (1 - amt)); return "rgb(" + f(c[0]) + "," + f(c[1]) + "," + f(c[2]) + ")"; },

  // ---- pan / zoom (tree only) ----
  applyTransform() { if (this.stage) this.stage.style.transform = "translate(" + this.state.tx + "px," + this.state.ty + "px) scale(" + this.state.scale + ")"; },
  onPanStart(e) { if (e.button !== 0) return; this._pan = { x: e.clientX, y: e.clientY, tx: this.state.tx, ty: this.state.ty, moved: false }; },
  _onMove(e) { if (!this._pan) return; const dx = e.clientX - this._pan.x, dy = e.clientY - this._pan.y; if (Math.abs(dx) + Math.abs(dy) > 3) this._pan.moved = true; this.state.tx = this._pan.tx + dx; this.state.ty = this._pan.ty + dy; this.applyTransform(); },
  _onWheel(e) { e.preventDefault(); const s = this.state.scale, ns = Math.max(0.2, Math.min(2.4, s * (1 + -e.deltaY * 0.0014))); const r = this.vp.getBoundingClientRect(); const cx = e.clientX - r.left, cy = e.clientY - r.top, k = ns / s; this.state.scale = ns; this.state.tx = cx - (cx - this.state.tx) * k; this.state.ty = cy - (cy - this.state.ty) * k; this.applyTransform(); },
  zoomBy(f) { const s = this.state.scale, ns = Math.max(0.2, Math.min(2.4, s * f)); const r = this.vp ? this.vp.getBoundingClientRect() : { width: innerWidth, height: innerHeight }; const cx = r.width / 2, cy = r.height / 2, k = ns / s; this.state.scale = ns; this.state.tx = cx - (cx - this.state.tx) * k; this.state.ty = cy - (cy - this.state.ty) * k; this.applyTransform(); },
  zoomIn() { this.zoomBy(1.2); }, zoomOut() { this.zoomBy(1 / 1.2); },
  resetView() { const o = this.state.layout === "outline"; this.setState({ scale: o ? 0.95 : 0.78, tx: o ? 40 : 80, ty: 60 }); setTimeout(() => this.focusRoot(), 30); },
  // fit/center reading zoom — fixed so apparent font stays constant regardless of expand count (tune live)
  FIT_ZOOM: { tree: 0.91, outline: 1.07 },
  // transpose of the tree anchor: branch is the tree rotated 90° (parent centred against its
  // whole subtree), so tree's top-centre becomes branch's left-centre. Preserves current scale.
  focusRoot() {
    if (!this.vp) return;
    const el = this.vp.querySelector('[data-pid="' + this.rootId + '"]'); if (!el) return;
    const vr = this.vp.getBoundingClientRect(), er = el.getBoundingClientRect();
    const topBar = this._topbarH || 64, bottom = this.isMobile() ? 140 : 24, margin = 44;
    if (this.state.layout === "outline") {
      // branch: root at the left margin, vertically centred in the visible band
      const bandCY = vr.top + topBar + (vr.height - topBar - bottom) / 2;
      this.state.tx += (vr.left + margin) - er.left;
      this.state.ty += bandCY - (er.top + er.height / 2);
    } else {
      // tree: root horizontally centred, near the top
      this.state.tx += (vr.left + vr.width / 2) - (er.left + er.width / 2);
      this.state.ty += (vr.top + topBar + margin) - er.top;
    }
    this.applyTransform();
  },
  // Snap to a fixed reading zoom and centre the root. Scale is constant (not fit-to-all)
  // so the apparent font stays the same no matter how many nodes are expanded; big trees
  // overflow intentionally (pan to explore). focusRoot centres root preserving the scale.
  fitView() {
    if (!this.vp || !this._fitEl) return this.resetView();
    const s = this.FIT_ZOOM[this.state.layout === "outline" ? "outline" : "tree"];
    this.setState({ scale: s });
    setTimeout(() => this.focusRoot(), 30);
  },

  // ---- touch pan / pinch (tree) ----
  _touchMid(e) { const a = e.touches[0], b = e.touches[1], r = this.vp.getBoundingClientRect(); return { dist: Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY), cx: (a.clientX + b.clientX) / 2 - r.left, cy: (a.clientY + b.clientY) / 2 - r.top }; },
  _onTouchStart(e) {
    if (e.touches.length === 1) { const t = e.touches[0]; this._pan = { x: t.clientX, y: t.clientY, tx: this.state.tx, ty: this.state.ty, moved: false }; this._pinch = null; }
    else if (e.touches.length === 2) { this._pan = null; const m = this._touchMid(e); this._pinch = Object.assign(m, { scale: this.state.scale, tx: this.state.tx, ty: this.state.ty }); }
  },
  _onTouchMove(e) {
    if (this._pinch && e.touches.length === 2) {
      e.preventDefault(); const m = this._touchMid(e), p = this._pinch, k = (m.dist / p.dist);
      const ns = Math.max(0.2, Math.min(2.4, p.scale * k)), f = ns / p.scale;
      this.state.scale = ns; this.state.tx = p.cx - (p.cx - p.tx) * f; this.state.ty = p.cy - (p.cy - p.ty) * f; this.applyTransform();
    } else if (this._pan && e.touches.length === 1) {
      const t = e.touches[0], dx = t.clientX - this._pan.x, dy = t.clientY - this._pan.y;
      if (Math.abs(dx) + Math.abs(dy) > 3) { this._pan.moved = true; e.preventDefault(); }
      this.state.tx = this._pan.tx + dx; this.state.ty = this._pan.ty + dy; this.applyTransform();
    }
  },
  _onTouchEnd(e) { if (e.touches.length === 0) { this._dragMoved = !!((this._pan && this._pan.moved) || this._pinch); this._pan = null; this._pinch = null; } else if (e.touches.length === 1) { const t = e.touches[0]; this._pinch = null; this._pan = { x: t.clientX, y: t.clientY, tx: this.state.tx, ty: this.state.ty, moved: true }; } },

  // ---- mobile sheet resize (drag the grabber) ----
  onPanelResizeStart(e) {
    const pt = e.touches ? e.touches[0] : e;
    this._presize = { y: pt.clientY, h: this.state.panelH };
    window.addEventListener("mousemove", this._pmove);
    window.addEventListener("mouseup", this._pend);
    window.addEventListener("touchmove", this._pmove, { passive: false });
    window.addEventListener("touchend", this._pend);
    if (e.cancelable) e.preventDefault();
  },
  onPanelResizeMove(e) {
    if (!this._presize) return;
    const pt = e.touches ? e.touches[0] : e;
    const h = Math.max(150, Math.min(window.innerHeight * 0.92, this._presize.h + (this._presize.y - pt.clientY)));
    if (e.cancelable) e.preventDefault();
    this._liveH = h;
    // live-resize both panes without a full re-render (smooth drag)
    if (this.panelEl) this.panelEl.style.height = h + "px";
    if (this.columnsEl) this.columnsEl.style.bottom = h + "px";
  },
  onPanelResizeEnd() {
    this._presize = null;
    window.removeEventListener("mousemove", this._pmove);
    window.removeEventListener("mouseup", this._pend);
    window.removeEventListener("touchmove", this._pmove);
    window.removeEventListener("touchend", this._pend);
    if (this._liveH != null) { this.setState({ panelH: Math.round(this._liveH) }); this._liveH = null; }
  },

  // anchor on root so the view stays put (no jump to top) — same trick as toggle()
  expandAll() { const keep = this._nodePos(this.rootId); const e = {}; this.people.forEach((p) => { if (this.childrenOf[p.id].length) e[p.id] = true; }); this.state.expanded = e; this.render(); if (keep) this._restorePos(this.rootId, keep); },
  collapseAll() { const keep = this._nodePos(this.rootId); const e = {}; e[this.rootId] = true; this.state.expanded = e; this.render(); if (keep) this._restorePos(this.rootId, keep); },
  // toggle, keeping the toggled node fixed on screen (tree & branch) so the
  // view doesn't jump/rearrange when a node collapses or expands
  toggle(id) {
    const keep = this._nodePos(id);
    this.state.expanded[id] = !this.state.expanded[id];
    this.render();
    if (keep) this._restorePos(id, keep);
  },
  _nodePos(id) { if (!this.vp) return null; const el = this.vp.querySelector('[data-pid="' + id + '"]'); if (!el) return null; const r = el.getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; },
  _restorePos(id, keep) { if (!this.vp) return; const el = this.vp.querySelector('[data-pid="' + id + '"]'); if (!el) return; const r = el.getBoundingClientRect(); this.state.tx += keep.x - (r.left + r.width / 2); this.state.ty += keep.y - (r.top + r.height / 2); this.applyTransform(); },
  expandAncestors(id) { let cur = this.byId[id]; while (cur && cur.parentId != null) { this.state.expanded[cur.parentId] = true; cur = this.byId[cur.parentId]; } },
  select(id) { if (this._pan && this._pan.moved) return; if (this.state.reorder && id !== this.state.reorder.focusId) this.reorderCancel(); this.setState({ selectedId: id, showInbox: false, showMine: false }); },
  // columns: clicking drills + opens detail; colSel is the column anchor so that
  // closing the detail panel does NOT collapse the drilled-in columns
  colSelect(id) { this.state.colSel = id; this.setState({ selectedId: id, showInbox: false, showMine: false }); },
  goTo(id) { this.expandAncestors(id); this.state.colSel = id; this.setState({ selectedId: id, query: "", showInbox: false, search: false }); requestAnimationFrame(() => this.focusPerson(id, false)); },
  closePanel() { if (this.state.reorder) this.reorderCancel(); this.setState({ selectedId: null }); },
  // Empty-canvas tap closes the topmost open overlay (node clicks stopPropagation,
  // so they never reach here). A drag-pan sets _dragMoved and is ignored. One
  // overlay per tap, outermost-first, so the canvas behaves like a modal scrim.
  _canvasTap() {
    if (this._dragMoved) return;
    const s = this.state;
    if (s.modal) return;                                   // modal has its own scrim
    if (s.search) return this.setState({ search: false });
    if (s.showInbox) return this.toggleInbox();
    if (s.showMine) return this.toggleMine();
    if (s.selectedId != null) return this.closePanel();
  },
  pathTo(id) { const out = []; let cur = this.byId[id]; while (cur) { out.unshift(cur.id); cur = cur.parentId != null ? this.byId[cur.parentId] : null; } return out; },
  focusPerson(id, top) {
    if (!this.vp) return;
    const el = this.vp.querySelector('[data-pid="' + id + '"]'); if (!el) return;
    const vr = this.vp.getBoundingClientRect(), er = el.getBoundingClientRect();
    const tx = vr.left + vr.width / 2, ty = top ? vr.top + 150 : vr.top + vr.height * 0.42;
    this.state.tx += tx - (er.left + er.width / 2); this.state.ty += ty - (er.top + er.height / 2); this.applyTransform();
  },

  // ---- TREE node ----
  node(id) {
    const p = this.byId[id]; if (!p) return null;
    const variant = this.state.variant, accent = this.state.accent, conn = "#b7a47e";
    const isSel = this.state.selectedId === id, isOpen = !!this.state.expanded[id];
    const kidIds = this.childrenOf[id] || [], hasKids = kidIds.length > 0;
    const onCard = (e) => { e.stopPropagation(); this.select(id); };
    const sub = p.origin || p.alias;

    let card;
    if (variant === "medallion") {
      const initial = Array.from(p.name)[0] || "";
      card = h("div", { onClick: onCard, style: { cursor: "pointer", display: "flex", "flex-direction": "column", "align-items": "center", gap: "7px", width: "104px" } },
        h("div", { style: { width: "56px", height: "56px", "border-radius": "50%", background: "linear-gradient(145deg," + this.lighten(accent, 0.22) + "," + this.darken(accent, 0.14) + ")", color: "#fbf5e7", display: "grid", "place-items": "center", "font-size": "22px", "font-weight": "600", border: "3px solid #fbf5e7", "box-shadow": isSel ? "0 0 0 3px " + this.hexA(accent, 0.35) + ",0 3px 9px rgba(80,55,20,.25)" : "0 2px 7px rgba(80,55,20,.22)" } }, initial),
        h("div", { style: { "font-size": "13.5px", color: "#3b2f21", "text-align": "center", "font-weight": isSel ? "700" : "500", "white-space": "nowrap", "line-height": "1.2" } }, this._nm(p)),
        sub ? h("div", { style: { "font-size": "11px", color: accent, "font-style": "italic", "margin-top": "-3px" } }, sub) : null);
    } else if (variant === "ledger") {
      card = h("div", { onClick: onCard, style: { cursor: "pointer", padding: "5px 16px 7px", "border-bottom": "2px solid " + (isSel ? accent : "#c2b189"), background: isSel ? this.hexA(accent, 0.08) : "transparent", "text-align": "center", "min-width": "84px" } },
        h("div", { style: { "font-size": "15.5px", color: "#3b2f21", "white-space": "nowrap", "letter-spacing": ".2px", "font-weight": isSel ? "600" : "400" } }, this._nm(p)),
        sub ? h("div", { style: { "font-size": "10.5px", color: accent, "font-style": "italic", "margin-top": "1px" } }, sub) : null);
    } else {
      card = h("div", { onClick: onCard, style: { cursor: "pointer", background: isSel ? "#f7ecd0" : "#fdf8ec", border: "1px solid " + (isSel ? accent : "#d9c9a2"), "border-radius": "11px", padding: "9px 17px", "min-width": "96px", "text-align": "center", "box-shadow": isSel ? "0 0 0 3px " + this.hexA(accent, 0.22) + ",0 4px 10px rgba(80,55,20,.16)" : "0 2px 6px rgba(80,55,20,.12)", transition: "box-shadow .15s,border-color .15s" } },
        h("div", { style: { "font-size": "16px", "font-weight": "600", color: "#3b2f21", "white-space": "nowrap", "line-height": "1.2" } }, this._nm(p)),
        sub ? h("div", { style: { "font-size": "11px", color: accent, "font-style": "italic", "margin-top": "3px", "white-space": "nowrap" } }, sub) : null);
    }

    let toggle = null;
    if (hasKids) toggle = h("button", { title: this.descOf[id] + " descendants", onClick: (e) => { e.stopPropagation(); this.toggle(id); }, style: { border: "1px solid " + accent, background: isOpen ? "#fbf5e7" : accent, color: isOpen ? accent : "#fbf5e7", height: "21px", "min-width": "21px", padding: isOpen ? "0" : "0 8px", "border-radius": "11px", "font-size": "12.5px", "font-weight": "700", cursor: "pointer", "line-height": "1", display: "flex", "align-items": "center", "justify-content": "center", "box-shadow": "0 1px 3px rgba(80,55,20,.25)" } }, isOpen ? "–" : "+" + kidIds.length);

    const cardWrap = h("div", { "data-pid": id, style: { display: "flex", "flex-direction": "column", "align-items": "center", gap: "6px" } }, card, toggle);
    const stack = [cardWrap];
    if (hasKids && isOpen) {
      const L = kidIds.length;
      stack.push(h("div", { style: { width: "2px", height: "20px", background: conn } }));
      stack.push(h("div", { style: { display: "flex", "align-items": "flex-start", gap: "30px" } },
        kidIds.map((cid, i) => {
          const cell = [];
          // extend 15px into the 30px sibling gap so the horizontal line is continuous
          if (i > 0) cell.push(h("div", { style: { position: "absolute", top: "0", left: "-15px", width: "calc(50% + 15px)", height: "2px", background: conn } }));
          if (i < L - 1) cell.push(h("div", { style: { position: "absolute", top: "0", right: "-15px", width: "calc(50% + 15px)", height: "2px", background: conn } }));
          cell.push(h("div", { style: { position: "absolute", top: "0", left: "calc(50% - 1px)", width: "2px", height: "26px", background: conn } }));
          return h("div", { style: { display: "flex", "flex-direction": "column", "align-items": "center" } }, h("div", { style: { position: "relative", width: "100%", height: "26px" } }, cell), this.node(cid));
        })));
    }
    return h("div", { style: { display: "flex", "flex-direction": "column", "align-items": "center" } }, stack);
  },

  // ---- BRANCH (left→right) node ----
  outlineNode(id) {
    const p = this.byId[id]; if (!p) return null;
    const accent = this.state.accent, conn = "#b7a47e";
    const kids = this.childrenOf[id] || [], hasKids = kids.length > 0;
    const isOpen = !!this.state.expanded[id], isSel = this.state.selectedId === id;
    const sub = p.origin || p.alias;

    const card = h("div", { onClick: (e) => { e.stopPropagation(); this.select(id); }, style: { cursor: "pointer", background: isSel ? "#f7ecd0" : "#fdf8ec", border: "1px solid " + (isSel ? accent : "#d9c9a2"), "border-radius": "10px", padding: "8px 15px", "white-space": "nowrap", "box-shadow": isSel ? "0 0 0 3px " + this.hexA(accent, 0.2) + ",0 3px 8px rgba(80,55,20,.14)" : "0 1px 4px rgba(80,55,20,.1)", transition: "box-shadow .15s,border-color .15s" } },
      h("div", { "data-pid": id, style: { "font-size": "15.5px", "font-weight": "600", color: "#3b2f21", "line-height": "1.2" } }, this._nm(p)),
      sub ? h("div", { style: { "font-size": "11px", "font-style": "italic", color: accent, "margin-top": "2px" } }, sub) : null);

    const toggle = hasKids ? h("button", { title: this.descOf[id] + " descendants", onClick: (e) => { e.stopPropagation(); this.toggle(id); }, style: { flex: "none", border: "1px solid " + accent, background: isOpen ? "#fbf5e7" : accent, color: isOpen ? accent : "#fbf5e7", height: "22px", "min-width": "22px", padding: isOpen ? "0" : "0 8px", "border-radius": "11px", "font-size": "12px", "font-weight": "700", cursor: "pointer", "line-height": "1", display: "flex", "align-items": "center", "justify-content": "center", "box-shadow": "0 1px 3px rgba(80,55,20,.22)" } }, isOpen ? "–" : "+" + kids.length) : null;

    const cardWrap = h("div", { style: { display: "flex", "align-items": "center", gap: "7px", flex: "none" } }, card, toggle);
    if (!hasKids || !isOpen) return h("div", { style: { display: "flex", "align-items": "center" } }, cardWrap);

    const L = kids.length;
    const childCol = h("div", { style: { display: "flex", "flex-direction": "column" } },
      kids.map((cid, i) => {
        // Vertical spine: each segment overshoots its row's 5px padding so it meets
        // the neighbour's segment across the 10px inter-row gap — one continuous line,
        // no break at junctions. First reaches down, last reaches up, middles span both.
        let vTop = "0", vH = "0", showV = true;
        if (L === 1) showV = false;
        else if (i === 0) { vTop = "50%"; vH = "calc(50% + 5px)"; }
        else if (i === L - 1) { vTop = "-5px"; vH = "calc(50% + 5px)"; }
        else { vTop = "-5px"; vH = "calc(100% + 10px)"; }
        const connector = h("div", { style: { position: "relative", width: "30px", flex: "none", "align-self": "stretch" } },
          h("div", { style: { position: "absolute", top: "calc(50% - 1px)", left: "0", width: "100%", height: "2px", background: conn } }),
          showV ? h("div", { style: { position: "absolute", left: "0", top: vTop, height: vH, width: "2px", background: conn } }) : null);
        return h("div", { style: { display: "flex", "align-items": "center", padding: "5px 0" } }, connector, this.outlineNode(cid));
      }));
    const parentStub = h("div", { style: { flex: "none", width: "16px", height: "2px", background: conn, "align-self": "center" } });
    return h("div", { style: { display: "flex", "align-items": "center" } }, cardWrap, parentStub, childCol);
  },

  // ---- COLUMNS (Finder-style) ----
  columnsView() {
    const accent = this.state.accent, sel = this.state.selectedId;
    // columns are anchored on colSel (persists when the detail panel is closed),
    // falling back to the selection / root; sel only drives row highlight
    const anchor = this.state.colSel != null ? this.state.colSel : sel != null ? sel : this.rootId;
    const path = this.pathTo(anchor);
    const cols = [];
    for (let i = 0; i < path.length; i++) {
      const parentId = path[i], kids = this.childrenOf[parentId] || [];
      if (!kids.length) break;
      const parent = this.byId[parentId], highlight = path[i + 1];
      cols.push(h("div", { style: { flex: "none", width: "244px", height: "100%", display: "flex", "flex-direction": "column", "border-right": "1px solid #d9c9a2", background: i % 2 ? "rgba(253,248,236,.45)" : "rgba(251,246,234,.55)" } },
        h("div", { style: { padding: "13px 16px 10px", "border-bottom": "1px solid #e6d8b8", flex: "none" } },
          h("div", { style: { "font-size": "10.5px", "letter-spacing": ".4px", color: "#a8854a" } }, "যাঁর সন্তান"),
          h("div", { style: { "font-size": "15.5px", "font-weight": "600", color: "#3b2f21", "white-space": "nowrap", overflow: "hidden", "text-overflow": "ellipsis" } }, this._nm(parent)),
          h("div", { style: { "font-size": "11.5px", color: "#9c8456", "margin-top": "2px" } }, kids.length + " জন")),
        h("div", { class: "kz-scroll", style: { flex: "1", overflow: "auto", padding: "6px" } },
          kids.map((cid) => {
            const c = this.byId[cid], hl = cid === highlight, csel = cid === sel, cKids = this.childrenOf[cid] || [], csub = c.origin || c.alias;
            return h("button", { onClick: () => this.colSelect(cid), style: { display: "flex", "align-items": "center", gap: "8px", width: "100%", "text-align": "left", border: "none", cursor: "pointer", "border-radius": "8px", padding: "9px 11px", "margin-bottom": "2px", background: csel ? accent : hl ? this.hexA(accent, 0.14) : "transparent", color: csel ? "#fbf5e7" : "#3b2f21" } },
              h("div", { style: { flex: "1", "min-width": "0" } },
                h("div", { style: { "font-size": "14.5px", "font-weight": hl || csel ? "600" : "500", "white-space": "nowrap", overflow: "hidden", "text-overflow": "ellipsis" } }, this._nm(c)),
                csub ? h("div", { style: { "font-size": "11px", "font-style": "italic", opacity: ".82", color: csel ? "#fbf5e7" : accent, "white-space": "nowrap", overflow: "hidden", "text-overflow": "ellipsis" } }, csub) : null),
              cKids.length ? h("span", { style: { flex: "none", "font-size": "12px", opacity: ".85", color: csel ? "#fbf5e7" : "#9c8456" } }, cKids.length + " ›") : null);
          }))));
    }
    return h("div", { style: { display: "flex", height: "100%", width: "max-content" } }, cols);
  },

  // ---- detail helpers ----
  fieldLabel(k) { return { name: "নাম", origin: "এলাকা", alias: "ডাকনাম", spouse: "স্বামী/স্ত্রী", birth: "জন্ম", death: "মৃত্যু", note: "মন্তব্য" }[k] || k; },
  fmtVal(k, v) { return v == null || v === "" ? "—" : String(v); },
  relTime(t) { if (!t) return ""; const d = Math.floor((Date.now() - t) / 1000); if (d < 60) return "এইমাত্র"; if (d < 3600) return Math.floor(d / 60) + " মিনিট আগে"; if (d < 86400) return Math.floor(d / 3600) + " ঘণ্টা আগে"; return new Date(t).toLocaleDateString(); },

  // ---- topbar + controls ----
  _seg(label, active, onClick, activeBg, full) { const base = { padding: "6px 12px", "font-size": "13px", border: "none", "border-radius": "7px", cursor: "pointer", background: active ? activeBg : "transparent", color: active ? "#fbf5e7" : "#7d6740", "font-weight": active ? "600" : "400", transition: "all .15s" }; if (full) Object.assign(base, { flex: "1", padding: "10px 4px", "font-size": "14px", "text-align": "center" }); return h("button", { onClick, style: base }, label); },
  _grp(kids) { return h("div", { style: { display: "flex", gap: "3px", background: "#ece0c2", border: "1px solid #d4c096", "border-radius": "9px", padding: "3px", "flex-wrap": "wrap" } }, kids); },

  // The input node is kept alive while typing (no full re-render) so Avro/phonetic
  // composition and the mobile keyboard's suggestion strip survive. _fillSearch
  // rebuilds only the dropdown's contents in place.
  _searchItem(p) {
    const sub = "প্রজন্ম " + (this.depthOf[p.id] + 1) + (p.parentId != null && this.byId[p.parentId] ? " · " + this.byId[p.parentId].name + "-এর সন্তান" : " · মূল পুরুষ");
    // mousedown + preventDefault: fire the selection BEFORE the focused input blurs,
    // so a single tap on mobile acts immediately (no "first tap just hides keyboard").
    const b = h("button", { onMouseDown: (e) => { e.preventDefault(); this.goTo(p.id); }, style: { display: "flex", "flex-direction": "column", "align-items": "flex-start", gap: "1px", width: "100%", "text-align": "left", padding: "7px 10px", border: "none", background: "transparent", "border-radius": "7px", cursor: "pointer", color: "#3b2f21" } }, h("span", { style: { "font-size": "14.5px", "font-weight": "500" } }, this._nm(p)), h("span", { style: { "font-size": "11px", color: "#9c8456" } }, sub));
    hover(b, "background:#efe2c2"); return b;
  },
  _fillSearch() {
    const el = this._searchDrop; if (!el) return;
    const q = this.state.query.trim();
    // Latin query -> match against the romanized name ("masud" => মাসুদ / মাসুদুর রহমান);
    // Bengali query -> substring match on the name as typed. _rkey cached per person.
    const ql = q.toLowerCase(), latin = HAS_LATIN.test(q);
    const hit = (p) => latin ? (p._rkey || (p._rkey = bnRomanize(p.name))).indexOf(ql) !== -1 : p.name.indexOf(q) !== -1;
    const results = q ? this.people.filter(hit).slice(0, 14) : [];
    el.style.display = results.length ? "block" : "none";
    el.replaceChildren.apply(el, results.map((p) => this._searchItem(p)));
  },
  ctlSearch(full) {
    const input = h("input", Object.assign({ "data-fkey": "search", value: this.state.query, placeholder: "নাম খুঁজুন…", style: { width: full ? "100%" : "170px", padding: "8px 12px", "font-size": "14px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fbf6ea", color: "#3b2f21", outline: "none" } }, this._ime((v) => { this.state.query = v; this._fillSearch(); }, true)));
    const drop = h("div", { ref: (el) => (this._searchDrop = el), class: "kz-scroll", style: { position: "absolute", top: "42px", left: "0", width: full ? "100%" : "280px", "max-height": "300px", overflow: "auto", background: "#fbf6ea", border: "1px solid #d4c096", "border-radius": "10px", "box-shadow": "0 12px 30px rgba(70,48,18,.22)", "z-index": "40", padding: "5px", display: "none" } });
    const wrap = h("div", { style: { position: "relative", flex: full ? "1" : "none" } }, input, drop);
    this._fillSearch();
    return wrap;
  },
  ctlLayout(full) {
    const a = this.state.accent, l = this.state.layout;
    const go = (v) => {
      if (v === "explorer" && this.state.colSel == null) this.state.colSel = this.state.selectedId != null ? this.state.selectedId : this.rootId;
      this.setState({ layout: v, menu: false });
      if (v !== "explorer") requestAnimationFrame(() => this.focusRoot());
    };
    return this._grp([["tree", "নকশা"], ["outline", "তালিকা"], ["explorer", "স্তর"]].map(([v, t]) => this._seg(t, l === v, () => go(v), a, full)));
  },
  // Medallions & Ledger are hidden from public view for now (node() still supports
  // them — re-add the entries below to bring them back).
  // four-arrow icon: dir "out" = arrows to corners (expand), "in" = arrows to centre (collapse)
  _arrows(dir) {
    const ln = (x1, y1, x2, y2) => h("line", { x1, y1, x2, y2, stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round" });
    const hd = (pts) => h("polyline", { points: pts, fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round", "stroke-linejoin": "round" });
    const kids = dir === "out"
      ? [ln(11, 11, 4, 4), hd("4,9 4,4 9,4"), ln(13, 11, 20, 4), hd("15,4 20,4 20,9"), ln(11, 13, 4, 20), hd("4,15 4,20 9,20"), ln(13, 13, 20, 20), hd("15,20 20,20 20,15")]
      : [ln(4, 4, 10, 10), hd("10,5 10,10 5,10"), ln(20, 4, 14, 10), hd("14,5 14,10 19,10"), ln(4, 20, 10, 14), hd("5,14 10,14 10,19"), ln(20, 20, 14, 14), hd("19,14 14,14 14,19")];
    return h("svg", { viewBox: "0 0 24 24", width: "19", height: "19", style: { display: "block" } }, kids);
  },
  _searchIcon() {
    return h("svg", { viewBox: "0 0 24 24", width: "18", height: "18", style: { display: "block" } },
      [h("circle", { cx: "10", cy: "10", r: "6.5", fill: "none", stroke: "currentColor", "stroke-width": "2" }),
       h("line", { x1: "14.8", y1: "14.8", x2: "20", y2: "20", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round" })]);
  },
  _menuIcon() {
    return h("svg", { viewBox: "0 0 24 24", width: "20", height: "20", style: { display: "block" } },
      [5, 12, 19].map((y) => h("line", { x1: "4", y1: y, x2: "20", y2: y, stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round" })));
  },
  _expandBtn(dir, onClick, title) { const b = h("button", { onClick, title, style: { display: "flex", "align-items": "center", "justify-content": "center", width: "38px", height: "38px", border: "1px solid #cdb988", "border-radius": "9px", background: "#fbf6ea", color: "#5c4a2c", cursor: "pointer" } }, this._arrows(dir)); hover(b, "background:#f1e6cb"); return b; },
  ctlExpand() { return h("div", { style: { display: "flex", gap: "6px" } }, this._expandBtn("out", () => this.expandAll(), "সব খুলুন"), this._expandBtn("in", () => this.collapseAll(), "সব বন্ধ")); },
  ctlSwatches() { const accent = this.state.accent; return h("div", { style: { display: "flex", gap: "6px", "align-items": "center" } }, this.ACCENTS.map((c) => h("button", { title: "accent", onClick: () => this.setState({ accent: c }), style: { width: "22px", height: "22px", "border-radius": "50%", background: c, border: c === accent ? "2px solid #3b2f21" : "2px solid #fbf5e7", cursor: "pointer", "box-shadow": "0 1px 3px rgba(80,55,20,.3)" } }))); },
  logo(small) {
    return h("div", { style: { display: "flex", "align-items": "center", gap: small ? "9px" : "13px", flex: "none" } },
      h("div", { style: { width: small ? "34px" : "40px", height: small ? "34px" : "40px", "border-radius": "50%", border: "2px solid #9c4326", display: "flex", "align-items": "center", "justify-content": "center", color: "#9c4326", "font-size": small ? "18px" : "21px", "font-weight": "600", background: "#fbf5e7", "flex-shrink": "0" } }, "ক"),
      h("div", null,
        h("div", { style: { "font-size": small ? "17px" : "21px", "font-weight": "600", "letter-spacing": ".3px", "line-height": "1" } }, "Kazi Ancestry"),
        small ? null : h("div", { style: { "font-size": "12px", color: "#8a7146", "margin-top": "4px", "letter-spacing": ".2px" } }, "কাজী বংশলতিকা · " + this.people.length + " জন · " + (this.maxDepth + 1) + " প্রজন্ম")));
  },

  topbar() {
    const barStyle = { position: "absolute", top: "0", left: "0", right: "0", display: "flex", "align-items": "center", "justify-content": "space-between", "z-index": "20", background: "linear-gradient(180deg, rgba(244,236,214,.97), rgba(244,236,214,.85))", "backdrop-filter": "blur(6px)", "border-bottom": "1px solid #d4c096" };
    if (this.isMobile()) {
      const searchBtn = h("button", { title: "খুঁজুন", onClick: () => { const open = !this.state.search; this.setState({ search: open, menu: false }); if (open) requestAnimationFrame(() => { const el = this.root.querySelector('[data-fkey="search"]'); if (el) el.focus(); }); }, style: { display: "flex", "align-items": "center", "justify-content": "center", flex: "none", width: "40px", height: "40px", border: "1px solid #cdb988", "border-radius": "9px", background: this.state.search ? "#f1e6cb" : "#fbf6ea", color: "#5c4a2c", cursor: "pointer" } }, this._searchIcon());
      // Secondary account actions (name, যাচাই, আমার প্রস্তাব, log in/out) collapse
      // into this ⋯ menu so the bar fits 320px. Mode toggle lives in the bottom bar
      // (always tappable); a dot flags an admin's pending suggestions while collapsed.
      const pending = this.state.user && this.isAdmin() ? this.state.suggestions.filter((s) => s.status === "pending").length : 0;
      const menuBtn = h("button", { title: "মেনু", onClick: () => this.setState({ menu: !this.state.menu, search: false }), style: { position: "relative", display: "flex", "align-items": "center", "justify-content": "center", flex: "none", width: "40px", height: "40px", border: "1px solid #cdb988", "border-radius": "9px", background: this.state.menu ? "#f1e6cb" : "#fbf6ea", color: "#5c4a2c", cursor: "pointer" } },
        this._menuIcon(), pending ? h("span", { style: { position: "absolute", top: "-4px", right: "-4px", "min-width": "16px", height: "16px", padding: "0 4px", "border-radius": "8px", background: "#9c4326", color: "#fbf5e7", "font-size": "10.5px", "font-weight": "700", "line-height": "16px", "text-align": "center" } }, pending) : null);
      return h("div", { ref: (el) => (this.topbarEl = el), style: Object.assign({}, barStyle, { gap: "10px", padding: "10px 14px" }) }, this.logo(true),
        h("div", { style: { display: "flex", "align-items": "center", gap: "8px", flex: "none" } }, searchBtn, menuBtn));
    }
    return h("div", { ref: (el) => (this.topbarEl = el), style: Object.assign({}, barStyle, { gap: "18px", padding: "12px 20px", "flex-wrap": "wrap" }) },
      this.logo(false),
      h("div", { style: { display: "flex", "align-items": "center", gap: "11px", "flex-wrap": "wrap", "justify-content": "flex-end" } },
        this.ctlSearch(false), this.ctlLayout(),
        h("div", { style: { width: "1px", height: "26px", background: "#d4c096" } }),
        this.account()));
  },

  searchSheet() {
    if (!this.isMobile() || !this.state.search) return null;
    const card = h("div", { onMouseDown: (e) => e.stopPropagation(), style: { position: "absolute", top: (this._topbarH + 8) + "px", left: "12px", right: "12px", background: "#fbf6ea", border: "1px solid #d4c096", "border-radius": "14px", "box-shadow": "0 16px 40px rgba(70,48,18,.24)", "z-index": "46", padding: "14px", animation: "kzpop .16s ease" } },
      this.ctlSearch(true));
    return h("div", { onMouseDown: () => this.setState({ search: false }), style: { position: "absolute", inset: "0", "z-index": "44" } }, card);
  },

  // mobile overflow menu (⋯): the account actions that don't fit the narrow bar.
  menuSheet() {
    if (!this.isMobile() || !this.state.menu) return null;
    const accent = this.state.accent, user = this.state.user;
    const close = () => (this.state.menu = false);
    const item = (label, onClick, opts) => {
      opts = opts || {};
      const b = h("button", { onClick: () => { close(); onClick(); }, style: { display: "flex", "align-items": "center", "justify-content": "space-between", gap: "10px", width: "100%", "text-align": "left", padding: "11px 13px", "font-size": "14.5px", border: "1px solid " + (opts.accent ? "#9c4326" : "#e2d2a8"), "border-radius": "9px", background: opts.accent ? accent : "#fdf9ee", color: opts.accent ? "#fbf5e7" : "#5c4a2c", cursor: "pointer", "font-weight": opts.accent ? "600" : "500" } },
        label, opts.badge != null ? h("span", { style: { background: "#fbf5e7", color: "#9c4326", "font-size": "11.5px", "font-weight": "700", "border-radius": "9px", padding: "1px 7px" } }, opts.badge) : null);
      hover(b, opts.accent ? "" : "background:#f1e6cb"); return b;
    };
    const rows = [];
    if (!user) {
      rows.push(item("লগ ইন", () => this.login(), { accent: true }));
    } else {
      const role = this.role();
      const badge = { admin: ["কর্তৃপক্ষ", "#9c4326"], viewer: ["দর্শক", "#8a6d4a"] }[role];
      rows.push(h("div", { style: { display: "flex", "align-items": "center", gap: "8px", padding: "4px 2px 10px" } },
        h("span", { style: { "font-size": "15px", "font-weight": "600", color: "#3b2f21" } }, user.name),
        badge ? h("span", { style: { "font-size": "10.5px", "letter-spacing": ".5px", color: "#fbf5e7", background: badge[1], "border-radius": "10px", padding: "2px 7px" } }, badge[0]) : null));
      if (this.isAdmin()) rows.push(item("যাচাই", () => this.toggleInbox(), { badge: this.state.suggestions.filter((s) => s.status === "pending").length || null }));
      if (this.canSuggest() && !this.isAdmin()) rows.push(item("আমার প্রস্তাব", () => this.toggleMine()));
      rows.push(item("লগ আউট", () => this.signOut()));
    }
    const card = h("div", { onMouseDown: (e) => e.stopPropagation(), style: { position: "absolute", top: (this._topbarH + 8) + "px", right: "12px", width: "min(260px, calc(100vw - 24px))", display: "flex", "flex-direction": "column", gap: "8px", background: "#fbf6ea", border: "1px solid #d4c096", "border-radius": "14px", "box-shadow": "0 16px 40px rgba(70,48,18,.24)", "z-index": "47", padding: "12px", animation: "kzpop .16s ease" } }, rows);
    return h("div", { onMouseDown: () => this.setState({ menu: false }), style: { position: "absolute", inset: "0", "z-index": "44" } }, card);
  },

  account() {
    const accent = this.state.accent, user = this.state.user;
    if (!user) {
      const b = h("button", { onClick: () => this.login(), style: { padding: "8px 14px", "font-size": "13.5px", border: "1px solid #9c4326", "border-radius": "8px", background: accent, color: "#fbf5e7", cursor: "pointer", "font-weight": "500" } }, "লগ ইন");
      return h("div", { style: { display: "flex", "align-items": "center", gap: "10px", "flex-wrap": "wrap" } }, b);
    }
    const role = this.role(), isAdmin = role === "admin";
    // Contributors get no role chip — only কর্তৃপক্ষ (admin) and দর্শক (viewer) are badged.
    const badge = { admin: ["কর্তৃপক্ষ", "#9c4326"], viewer: ["দর্শক", "#8a6d4a"] }[role];
    const chip = h("div", { style: { display: "flex", "align-items": "center", gap: "7px", padding: "5px 11px", border: "1px solid #d4c096", "border-radius": "8px", background: "#fbf6ea" } },
      h("span", { style: { "font-size": "13.5px", "font-weight": "600", color: "#3b2f21" } }, user.name),
      badge ? h("span", { style: { "font-size": "10.5px", "letter-spacing": ".5px", color: "#fbf5e7", background: badge[1], "border-radius": "10px", padding: "2px 7px" } }, badge[0]) : null);
    const out = h("button", { onClick: () => this.signOut(), style: { padding: "8px 12px", "font-size": "13px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fbf6ea", color: "#5c4a2c", cursor: "pointer" } }, "লগ আউট");
    hover(out, "background:#f1e6cb");
    let review = null;
    if (isAdmin) {
      const pending = this.state.suggestions.filter((s) => s.status === "pending");
      review = h("button", { onClick: () => this.toggleInbox(), style: { display: "flex", "align-items": "center", gap: "5px", padding: "8px 13px", "font-size": "13.5px", border: "1px solid " + (pending.length ? "#9c4326" : "#cdb988"), "border-radius": "8px", background: pending.length ? accent : "#fbf6ea", color: pending.length ? "#fbf5e7" : "#5c4a2c", cursor: "pointer", "font-weight": "500" } },
        "যাচাই", pending.length ? h("span", { style: { background: "#fbf5e7", color: "#9c4326", "font-size": "11.5px", "font-weight": "700", "border-radius": "9px", padding: "1px 7px", "margin-left": "2px" } }, pending.length) : null);
    }
    let mine = null;
    if (this.canSuggest() && !isAdmin) {
      mine = h("button", { onClick: () => this.toggleMine(), style: { padding: "8px 13px", "font-size": "13.5px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fbf6ea", color: "#5c4a2c", cursor: "pointer", "font-weight": "500" } }, "আমার প্রস্তাব");
      hover(mine, "background:#f1e6cb");
    }
    return h("div", { style: { display: "flex", "align-items": "center", gap: "10px", "flex-wrap": "wrap" } }, chip, review, mine, out);
  },

  bottomLeft() {
    const canvas = this.state.layout === "tree" || this.state.layout === "outline", mobile = this.isMobile();
    const zbtn = (label, onClick, fs, title) => { const b = h("button", { onClick, title, style: { display: "flex", "align-items": "center", "justify-content": "center", width: "38px", height: "38px", border: "1px solid #cdb988", "border-radius": "9px", background: "rgba(251,246,234,.92)", color: "#5c4a2c", "font-size": fs, cursor: "pointer", "line-height": "1", "backdrop-filter": "blur(4px)" } }, label); hover(b, "background:#f1e6cb"); return b; };
    const zoom = canvas ? h("div", { style: { display: "flex", "flex-direction": "column", gap: "6px" } }, zbtn("+", () => this.zoomIn(), "20px"), zbtn("−", () => this.zoomOut(), "20px"), zbtn("◎", () => this.fitView(), "18px", "মাঝে আনুন")) : null;

    if (mobile) {
      const showExp = this.state.layout !== "explorer";
      const exp = showExp ? h("div", { style: { display: "flex", "flex-direction": "column", gap: "6px" } },
        zbtn(this._arrows("out"), () => this.expandAll(), "", "সব খুলুন"), zbtn(this._arrows("in"), () => this.collapseAll(), "", "সব বন্ধ")) : null;
      return h("div", { style: { position: "absolute", inset: "0", "pointer-events": "none", "z-index": "20" } },
        zoom ? h("div", { style: { position: "absolute", left: "14px", bottom: "16px", "pointer-events": "auto" } }, zoom) : null,
        h("div", { style: { position: "absolute", left: "50%", bottom: "16px", transform: "translateX(-50%)", "pointer-events": "auto", "border-radius": "9px", "box-shadow": "0 6px 18px rgba(70,48,18,.22)" } }, this.ctlLayout()),
        exp ? h("div", { style: { position: "absolute", right: "14px", bottom: "16px", "pointer-events": "auto" } }, exp) : null);
    }
    const showExp = this.state.layout !== "explorer";
    const exp = showExp ? h("div", { style: { display: "flex", "flex-direction": "column", gap: "6px" } },
      zbtn(this._arrows("out"), () => this.expandAll(), "", "সব খুলুন"), zbtn(this._arrows("in"), () => this.collapseAll(), "", "সব বন্ধ")) : null;
    return h("div", { style: { position: "absolute", inset: "0", "pointer-events": "none", "z-index": "20" } },
      h("div", { style: { position: "absolute", left: "18px", bottom: "18px", display: "flex", "align-items": "flex-end", gap: "13px", "pointer-events": "auto", "flex-wrap": "wrap" } },
        zoom,
        h("div", { style: { background: "rgba(251,246,234,.85)", border: "1px solid #ddcba0", "border-radius": "10px", padding: "9px 13px", "font-size": "11.5px", color: "#7d6740", "line-height": "1.7", "backdrop-filter": "blur(4px)" } },
          h("div", null, h("em", { style: { color: "#9c4326" } }, "বাঁকা লেখা"), " — এলাকা"),
          canvas ? h("div", null, h("b", null, "+n"), " — n সন্তান দেখুন") : null,
          h("div", { style: { "margin-top": "6px", "padding-top": "6px", "border-top": "1px solid #e6d8b8" } }, this.ctlSwatches()))),
      exp ? h("div", { style: { position: "absolute", right: "18px", bottom: "18px", "pointer-events": "auto" } }, exp) : null);
  },

  panel() {
    const id = this.state.selectedId; if (id == null || !this.byId[id]) return null;
    const s = this.byId[id], accent = this.state.accent, admin = this.isAdmin(), canAct = this.canAct();
    const life = [s.birth, s.death].some((x) => x) ? (s.birth || "?") + " – " + (s.death || "?") : "";
    const noteParts = []; if (s.note) noteParts.push(s.note);
    const kidIds = this.childrenOf[id] || [];
    const parentCell = s.parentId != null && this.byId[s.parentId] ? this._personPill(s.parentId) : "মূল পুরুষ";

    const row = (label, val) => h("div", { style: { display: "flex", "justify-content": "space-between", "align-items": "center", gap: "14px", padding: "11px 0", "border-bottom": "1px solid #e6d8b8" } }, h("span", { style: { "font-size": "13px", color: "#9c8456", "letter-spacing": ".3px" } }, label), h("span", { style: { "font-size": "15px", "text-align": "right" } }, val));
    // viewers see only filled fields; editors keep empty rows (shown as —) so they know what to fill
    const viewer = !canAct, orow = (label, val) => (viewer && (val == null || val === "")) ? null : row(label, val || "—");
    const actBtn = (label, onClick, kind) => {
      const styles = { primary: { background: accent, color: "#fbf5e7", border: "1px solid #9c4326", "font-weight": "500" }, plain: { background: "#fdf9ee", color: "#5c4a2c", border: "1px solid #cdb988" }, danger: { background: "transparent", color: "#a8442a", border: "1px solid #c98b73" } }[kind];
      const b = h("button", { onClick, style: Object.assign({ padding: "8px 14px", "font-size": "13.5px", "border-radius": "8px", cursor: "pointer" }, styles) }, label);
      if (kind === "plain") hover(b, "background:#f1e6cb"); if (kind === "danger") hover(b, "background:#f3ddd3"); return b;
    };

    let actions = null;
    if (canAct) {
      // sibling reorder is a staged draft (see startReorder): ←/→ rearrange the
      // live order (the tree previews it); admin commits with সম্পন্ন, a
      // contributor sends one proposal — nothing saved per keystroke.
      const sib = s.parentId != null ? (this.childrenOf[s.parentId] || []) : [];
      const inReorder = this.state.reorder && this.state.reorder.parentId === s.parentId;
      let reorder = null;
      if (sib.length > 1 && !inReorder) {
        reorder = h("div", { style: { "margin-top": "18px" } }, actBtn("ভাইবোনদের ক্রম বদলান", () => this.startReorder(id), "plain"));
      } else if (inReorder) {
        const focus = this.state.reorder.focusId, fi = sib.indexOf(focus);
        const mvBtn = (label, dir, dis) => { const b = h("button", { onClick: dis ? null : () => this.moveSibling(dir), disabled: dis, title: dir < 0 ? "আগে সরান" : "পরে সরান", style: { padding: "7px 16px", "font-size": "17px", "line-height": "1", border: "1px solid #cdb988", "border-radius": "8px", background: dis ? "#f1ead7" : "#fdf9ee", color: dis ? "#bcae87" : "#5c4a2c", cursor: dis ? "default" : "pointer", opacity: dis ? ".55" : "1" } }, label); if (!dis) hover(b, "background:#f1e6cb"); return b; };
        const list = h("div", { style: { display: "flex", "flex-direction": "column", gap: "4px", margin: "12px 0" } }, sib.map((cid, idx) => { const on = cid === focus; return h("button", { onClick: () => this.reorderFocus(cid), style: { display: "flex", "align-items": "center", gap: "9px", width: "100%", "text-align": "left", border: "1px solid " + (on ? accent : "#e2d2a8"), background: on ? this.hexA(accent, 0.1) : "#fdf9ee", "border-radius": "8px", padding: "7px 10px", cursor: "pointer", "font-size": "14px", color: "#3b2f21" } }, h("span", { style: { "font-size": "12px", color: "#9c8456", width: "15px", "flex-shrink": "0" } }, idx + 1), this.byId[cid].name); }));
        const changed = sib.some((cid, i) => cid !== this.state.reorder.orig[i]);
        reorder = h("div", { style: { "margin-top": "18px", "padding-top": "16px", "border-top": "1px dashed #d4c096" } },
          h("div", { style: { "font-size": "12px", "letter-spacing": ".4px", color: "#a8854a", "margin-bottom": "10px" } }, "ক্রম সাজান — “" + this.byId[focus].name + "” সরান"),
          h("div", { style: { display: "flex", gap: "8px", "align-items": "center" } }, mvBtn("←", -1, fi <= 0), mvBtn("→", 1, fi >= sib.length - 1), h("span", { style: { "font-size": "12.5px", color: "#9c8456" } }, (fi + 1) + " / " + sib.length)),
          list,
          h("div", { style: { display: "flex", gap: "8px" } }, actBtn(admin ? "সম্পন্ন" : "প্রস্তাব পাঠান", () => this.reorderDone(), changed ? "primary" : "plain"), actBtn("বাতিল", () => this.reorderCancel(), "plain")));
      }
      // while staging a reorder, show only the reorder controls so an edit/add
      // (which would commit + re-sort) can't silently drop the draft
      actions = h("div", { style: { "margin-top": "24px", "padding-top": "18px", "border-top": "1px solid #e6d8b8" } },
        inReorder ? null : h("div", { style: { "font-size": "12px", "letter-spacing": ".4px", color: "#a8854a", "margin-bottom": "11px" } }, admin ? "তথ্য সংশোধন" : "সংশোধনের প্রস্তাব"),
        inReorder ? null : h("div", { style: { display: "flex", "flex-wrap": "wrap", gap: "8px" } },
          actBtn(admin ? "সংশোধন" : "সংশোধনের প্রস্তাব", () => this.onEdit(), "primary"),
          actBtn(admin ? "সন্তান যোগ" : "সন্তানের প্রস্তাব", () => this.onAddChild(), "plain"),
          s.parentId != null ? actBtn(admin ? "ভাইবোন যোগ" : "ভাইবোনের প্রস্তাব", () => this.onAddSibling(), "plain") : null,
          admin && kidIds.length === 0 ? actBtn("মুছুন", () => this.onDelete(), "danger") : null),
        reorder);
    } else if (!this.state.user) {
      // Guest: invite login to contribute (read-only otherwise).
      actions = h("div", { style: { "margin-top": "24px", "padding-top": "18px", "border-top": "1px solid #e6d8b8" } },
        h("button", { onClick: () => this.login(), style: { padding: "9px 15px", "font-size": "13.5px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fbf6ea", color: "#5c4a2c", cursor: "pointer" } }, "প্রস্তাব দিতে লগ ইন করুন"));
    }

    const mobile = this.isMobile();
    const shell = mobile
      ? { position: "absolute", left: "0", right: "0", bottom: "0", height: this.state.panelH + "px", "max-height": "92%", background: "linear-gradient(180deg,#fbf6ea,#f6efde)", "border-top": "1px solid #d4c096", "border-radius": "18px 18px 0 0", "box-shadow": "0 -16px 44px rgba(70,48,18,.2)", "z-index": "50", overflow: "auto", padding: "8px 20px 28px", animation: "kzpop .18s ease" }
      : { position: "absolute", top: (this._topbarH || 64) + "px", right: "0", bottom: "0", width: "420px", background: "linear-gradient(180deg,#fbf6ea,#f6efde)", "border-left": "1px solid #d4c096", "box-shadow": "-16px 0 44px rgba(70,48,18,.18)", "z-index": "30", overflow: "auto", padding: "22px 22px 30px" };
    const handle = mobile ? h("div", { onMouseDown: (e) => this.onPanelResizeStart(e), ref: (el) => el && el.addEventListener("touchstart", (ev) => this.onPanelResizeStart(ev), { passive: false }), style: { "touch-action": "none", cursor: "ns-resize", padding: "6px 0 12px", margin: "-2px -20px 2px", display: "flex", "justify-content": "center" } }, h("div", { style: { width: "46px", height: "5px", "border-radius": "3px", background: "#cbb88f" } })) : null;
    return h("div", { class: "kz-scroll", ref: (el) => (this.panelEl = el), style: shell }, handle,
      h("div", { style: { display: "flex", "justify-content": "space-between", "align-items": "flex-start", gap: "10px" } }, h("div", { style: { "font-size": "12px", "letter-spacing": ".4px", color: "#a8854a" } }, "প্রজন্ম " + (this.depthOf[id] + 1)), h("button", { onClick: () => this.closePanel(), style: { border: "none", background: "transparent", color: "#9c8456", "font-size": "22px", cursor: "pointer", "line-height": "1", padding: "0" } }, "×")),
      h("div", { style: { "font-size": "29px", "font-weight": "600", margin: "6px 0 2px", "line-height": "1.15" } }, s.name),
      life ? h("div", { style: { "font-size": "14px", color: "#9c8456", "font-style": "italic" } }, life) : null,
      h("div", { style: { height: "2px", width: "46px", background: accent, margin: "14px 0 16px" } }),
      h("div", { style: { display: "flex", "flex-direction": "column" } }, orow("এলাকা", s.origin), orow("ডাকনাম", s.alias), orow("স্বামী/স্ত্রী", s.spouse), row("বাবা/মা", parentCell), row("ছেলেমেয়ে", kidIds.length), row("মোট বংশধর", this.descOf[id])),
      noteParts.length ? h("div", { style: { "margin-top": "16px", background: "#f3e8cc", border: "1px solid #e2d2a8", "border-radius": "9px", padding: "11px 13px", "font-size": "13.5px", color: "#7d6740", "line-height": "1.5" } }, noteParts.join(" ")) : null,
      kidIds.length ? h("div", { style: { "margin-top": "20px" } }, h("div", { style: { "font-size": "12px", "letter-spacing": ".4px", color: "#a8854a", "margin-bottom": "11px" } }, "ছেলেমেয়ে"), h("div", { style: { display: "flex", "flex-wrap": "wrap", gap: "8px" } }, kidIds.map((c) => this._personPill(c)))) : null,
      this._tagChips(s),
      actions);
  },

  // _statusBadge renders a colored pill for a resolved/pending suggestion.
  _statusBadge(status) {
    const m = { approved: ["অনুমোদিত", "#5c6b4a", "#eef0e6"], rejected: ["প্রত্যাখ্যাত", "#a8442a", "#f6e7e1"], pending: ["অপেক্ষমাণ", "#a8854a", "#f3ebd6"] }[status] || ["—", "#9c8456", "#f1e6cb"];
    return h("span", { style: { display: "inline-block", "font-size": "11.5px", "font-weight": "600", color: m[1], background: m[2], border: "1px solid " + m[1] + "55", "border-radius": "20px", padding: "3px 11px" } }, m[0]);
  },

  // _sugCard renders one suggestion. opts.actions adds approve/reject (admin pending);
  // otherwise a status badge is shown. opts.showBy=false hides the submitter line.
  _sugCard(x, opts) {
    opts = opts || {};
    const accent = this.state.accent;
    let rows = [], title = "", typeLabel = "";
    if (x.type === "edit") { typeLabel = "সংশোধন"; title = x.targetName || "—"; rows = Object.keys(x.changes || {}).map((k) => ({ field: this.fieldLabel(k), old: this.fmtVal(k, x.before ? x.before[k] : ""), nw: this.fmtVal(k, x.changes[k]) })); }
    else if (x.type === "reorder") { typeLabel = "ক্রম পরিবর্তন"; title = (x.parentName || "") + "-এর সন্তানদের ক্রম"; const nm = (ids) => (ids || []).map((cid) => (this.byId[cid] && this.byId[cid].name) || cid).join(", "); rows = [{ field: "ক্রম", old: nm(x.before), nw: nm(x.order) }]; }
    else if (x.type === "add" || x.fields) { const fields = x.fields || {}; typeLabel = "নতুন ব্যক্তি"; title = (fields.name || "—") + "  ·  " + (x.parentName || "") + "-এর অধীনে"; rows = ["origin", "alias", "spouse", "birth", "death", "note"].filter((k) => fields[k]).map((k) => ({ field: this.fieldLabel(k), old: "—", nw: this.fmtVal(k, fields[k]) })); if (rows.length === 0) rows = [{ field: "নাম", old: "—", nw: fields.name || "—" }]; }
    // unknown/legacy/empty-payload suggestion — render a minimal card instead of throwing
    else { typeLabel = "প্রস্তাব"; title = x.targetName || "—"; rows = []; }
    let footer;
    if (opts.actions) {
      const approve = h("button", { onClick: () => this.approve(x), style: { flex: "1", padding: "8px 0", "font-size": "13.5px", border: "none", "border-radius": "8px", background: "#5c6b4a", color: "#fbf5e7", cursor: "pointer", "font-weight": "500" } }, "অনুমোদন"); hover(approve, "background:#4d5a3e");
      const reject = h("button", { onClick: () => this.reject(x), style: { flex: "1", padding: "8px 0", "font-size": "13.5px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fdf9ee", color: "#8a6a52", cursor: "pointer" } }, "প্রত্যাখ্যান"); hover(reject, "background:#f1e6cb");
      footer = h("div", { style: { display: "flex", gap: "8px", "margin-top": "14px" } }, approve, reject);
    } else {
      footer = h("div", { style: { "margin-top": "13px" } }, this._statusBadge(x.status));
    }
    return h("div", { style: { background: "#fdf9ee", border: "1px solid #e2d2a8", "border-radius": "12px", padding: "15px 16px", "box-shadow": "0 2px 6px rgba(80,55,20,.07)" } },
      h("div", { style: { display: "flex", "justify-content": "space-between", "align-items": "baseline", gap: "10px" } }, h("span", { style: { "font-size": "11px", "letter-spacing": ".8px", "text-transform": "uppercase", "font-weight": "600", color: accent } }, typeLabel), h("span", { style: { "font-size": "11.5px", color: "#a89468" } }, this.relTime(x.at))),
      h("div", { style: { "font-size": "17px", "font-weight": "600", margin: "5px 0 2px" } }, title),
      opts.showBy === false ? null : h("div", { style: { "font-size": "12.5px", color: "#9c8456", "margin-bottom": "11px" } }, "প্রস্তাব দিয়েছেন: " + x.by),
      h("div", { style: { display: "flex", "flex-direction": "column", gap: "7px" } }, rows.map((r) => h("div", { style: { display: "flex", "align-items": "flex-start", gap: "8px", "font-size": "13.5px", "line-height": "1.45" } }, h("span", { style: { flex: "none", width: "62px", color: "#9c8456", "font-size": "12px", "padding-top": "2px" } }, r.field), h("span", { style: { flex: "1" } }, h("span", { style: { color: "#b07a6a", "text-decoration": "line-through", opacity: ".8" } }, r.old), " ", h("span", { style: { color: "#a89468" } }, "→"), " ", h("span", { style: { color: "#3b2f21", "font-weight": "500" } }, r.nw))))),
      footer);
  },

  // shared drawer shell + header for the inbox and my-suggestions panels.
  // The detail panel reuses these exact dimensions (see panel()) so both modals
  // are the same box — they're mutually exclusive, so whichever opens looks alike.
  _drawerShell() {
    return this.isMobile()
      ? { position: "absolute", left: "0", right: "0", bottom: "0", "max-height": "86%", background: "linear-gradient(180deg,#fbf6ea,#f4ecd9)", "border-top": "1px solid #d4c096", "border-radius": "18px 18px 0 0", "box-shadow": "0 -16px 44px rgba(70,48,18,.2)", "z-index": "55", overflow: "auto", padding: "16px 18px 26px", animation: "kzpop .18s ease" }
      : { position: "absolute", top: (this._topbarH || 64) + "px", right: "0", bottom: "0", width: "420px", background: "linear-gradient(180deg,#fbf6ea,#f4ecd9)", "border-left": "1px solid #d4c096", "box-shadow": "-16px 0 44px rgba(70,48,18,.18)", "z-index": "45", overflow: "auto", padding: "22px 22px 30px" };
  },
  _drawerHeader(title, onClose) {
    return h("div", { style: { display: "flex", "justify-content": "space-between", "align-items": "center", gap: "10px", "margin-bottom": "16px" } },
      h("div", { style: { "font-size": "20px", "font-weight": "600" } }, title),
      h("button", { onClick: onClose, style: { border: "none", background: "transparent", color: "#9c8456", "font-size": "22px", cursor: "pointer", "line-height": "1", padding: "0" } }, "×"));
  },

  inbox() {
    if (!(this.state.showInbox && this.isAdmin())) return null;
    const accent = this.state.accent, tab = this.state.inboxTab;
    const pending = this.state.suggestions.filter((x) => x.status === "pending");
    const resolved = this.state.suggestions.filter((x) => x.status !== "pending").sort((a, b) => (b.at || 0) - (a.at || 0));
    const list = tab === "pending" ? pending : resolved;
    const tabBtn = (key, label, n) => { const on = tab === key; return h("button", { onClick: () => this.setState({ inboxTab: key }), style: { flex: "1", padding: "8px 0", "font-size": "13.5px", border: "1px solid " + (on ? "#9c4326" : "#cdb988"), "border-radius": "8px", background: on ? accent : "#fbf6ea", color: on ? "#fbf5e7" : "#5c4a2c", cursor: "pointer", "font-weight": on ? "600" : "400" } }, label + " (" + n + ")"); };
    const empty = h("div", { style: { "text-align": "center", padding: "50px 20px", color: "#a89468" } }, h("div", { style: { "font-size": "15px" } }, tab === "pending" ? "কোনো অপেক্ষমাণ প্রস্তাব নেই" : "কোনো সম্পন্ন প্রস্তাব নেই"));
    return h("div", { class: "kz-scroll", style: this._drawerShell() },
      this._drawerHeader("প্রস্তাব যাচাই", () => this.toggleInbox()),
      h("div", { style: { display: "flex", gap: "8px", "margin-bottom": "18px" } }, tabBtn("pending", "অপেক্ষমাণ", pending.length), tabBtn("resolved", "সম্পন্ন", resolved.length)),
      list.length ? h("div", { style: { display: "flex", "flex-direction": "column", gap: "14px" } }, list.map((x) => this._sugCard(x, { actions: tab === "pending" }))) : empty);
  },

  // mySuggestions: a contributor's own submissions with their statuses (read-only).
  mySuggestions() {
    if (!(this.state.showMine && this.canSuggest())) return null;
    const list = (this.state.mine || []).slice().sort((a, b) => (b.at || 0) - (a.at || 0));
    const body = this.state.mine == null
      ? h("div", { style: { "text-align": "center", padding: "50px 20px", color: "#a89468" } }, "লোড হচ্ছে…")
      : (list.length ? h("div", { style: { display: "flex", "flex-direction": "column", gap: "14px" } }, list.map((x) => this._sugCard(x, { actions: false, showBy: false }))) : h("div", { style: { "text-align": "center", padding: "50px 20px", color: "#a89468" } }, "আপনি এখনো কোনো প্রস্তাব দেননি।"));
    return h("div", { class: "kz-scroll", style: this._drawerShell() },
      this._drawerHeader("আমার প্রস্তাব", () => this.toggleMine()),
      body);
  },

  modal() {
    const m = this.state.modal; if (!m) return null;
    const accent = this.state.accent, f = this.state.form, isSuggest = !!m.asSuggestion;
    let title = "", subtitle = "", cta = "";
    if (m.kind === "edit") { const nm = this.byId[m.target] ? this.byId[m.target].name : ""; title = isSuggest ? "সংশোধনের প্রস্তাব" : "তথ্য সংশোধন"; subtitle = isSuggest ? nm + "-এর সংশোধনের প্রস্তাব" : "সংশোধন করছেন: " + nm; cta = isSuggest ? "প্রস্তাব পাঠান" : "সংরক্ষণ করুন"; }
    else { title = isSuggest ? "নতুন ব্যক্তির প্রস্তাব" : "নতুন ব্যক্তি যোগ"; subtitle = (this.byId[m.parentId] ? this.byId[m.parentId].name : "") + "-এর সন্তান"; cta = isSuggest ? "প্রস্তাব পাঠান" : "যোগ করুন"; }
    const label = (t) => h("label", { style: { "font-size": "12px", "letter-spacing": ".4px", color: "#9c8456", "margin-bottom": "5px", display: "block" } }, t);
    const inp = (key, ph, fs) => h("input", { value: f[key] || "", placeholder: ph || "", onInput: (e) => this.setForm(key, e.target.value), style: { width: "100%", padding: "9px 11px", "font-size": fs || "14px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fdf9ee", color: "#3b2f21", outline: "none", "margin-bottom": "14px" } });
    const half = (l, key, ph) => h("div", { style: { flex: "1" } }, label(l), inp(key, ph));
    const save = h("button", { onClick: () => this.onModalSave(), style: { padding: "10px 20px", "font-size": "14px", border: "none", "border-radius": "9px", background: accent, color: "#fbf5e7", cursor: "pointer", "font-weight": "600" } }, cta);
    const cancel = h("button", { onClick: () => this.onModalCancel(), style: { padding: "10px 18px", "font-size": "14px", border: "1px solid #cdb988", "border-radius": "9px", background: "transparent", color: "#8a6a52", cursor: "pointer" } }, "বাতিল"); hover(cancel, "background:#efe2c2");
    const dialog = h("div", { onMouseDown: (e) => e.stopPropagation(), class: "kz-scroll", style: { width: "460px", "max-width": "100%", "max-height": "90%", overflow: "auto", background: "linear-gradient(180deg,#fbf6ea,#f6efde)", border: "1px solid #d4c096", "border-radius": "16px", "box-shadow": "0 24px 70px rgba(50,32,12,.4)", padding: "26px 28px", animation: "kzpop .18s ease" } },
      h("div", { style: { "font-size": "22px", "font-weight": "600" } }, title),
      h("div", { style: { "font-size": "13.5px", color: "#9c8456", "margin-top": "4px", "margin-bottom": "20px" } }, subtitle),
      label("নাম"), inp("name", "পূর্ণ নাম", "15px"),
      h("div", { style: { display: "flex", gap: "12px" } }, half("এলাকা", "origin"), half("ডাকনাম", "alias")),
      label("স্বামী/স্ত্রী"), inp("spouse"),
      h("div", { style: { display: "flex", gap: "12px" } }, half("জন্ম", "birth", "যেমন ১৯৪৮"), half("মৃত্যু", "death")),
      label("মন্তব্য"),
      h("textarea", { rows: "3", onInput: (e) => this.setForm("note", e.target.value), style: { width: "100%", padding: "9px 11px", "font-size": "14px", border: "1px solid #cdb988", "border-radius": "8px", background: "#fdf9ee", color: "#3b2f21", outline: "none", resize: "vertical", "margin-bottom": "12px" } }, f.note || ""),
      h("label", { style: { display: "flex", "align-items": "center", gap: "9px", "font-size": "14px", color: "#5c4a2c", cursor: "pointer", "margin-bottom": "6px" } }, h("input", { type: "checkbox", checked: this.hasTag(f, "died_young"), onChange: (e) => this._setFormTag("died_young", e.target.checked), style: { width: "16px", height: "16px", "accent-color": "#9c4326" } }), "অল্প বয়সে মৃত্যু (✻)"),
      isSuggest ? h("div", { style: { "margin-top": "10px", background: "#f1e7cc", border: "1px solid #e2d2a8", "border-radius": "8px", padding: "10px 12px", "font-size": "12.5px", color: "#8a6d3f", "line-height": "1.5" } }, "এটি প্রস্তাব হিসেবে পাঠানো হবে — কর্তৃপক্ষ অনুমোদন করলে তবেই গাছে যুক্ত হবে।") : null,
      h("div", { style: { display: "flex", "justify-content": "flex-end", gap: "10px", "margin-top": "22px" } }, cancel, save));
    return h("div", { onMouseDown: () => this.onModalCancel(), style: { position: "absolute", inset: "0", background: "rgba(58,40,20,.34)", "backdrop-filter": "blur(2px)", "z-index": "60", display: "flex", "align-items": "center", "justify-content": "center", padding: "24px" } }, dialog);
  },

  toastEl() { if (!this.state.toast) return null; return h("div", { style: { position: "absolute", bottom: "26px", left: "50%", transform: "translateX(-50%)", background: "#3b2f21", color: "#fbf5e7", padding: "11px 20px", "border-radius": "24px", "font-size": "14px", "box-shadow": "0 8px 24px rgba(40,26,8,.35)", "z-index": "70", animation: "kztoast .2s ease" } }, this.state.toast); },

  layoutLayer() {
    const layout = this.state.layout, bg = "radial-gradient(125% 120% at 50% 16%, #f5edd7 0%, #ece0c2 55%, #ddcca6 100%)";
    // tree & branch share one pan/zoom canvas (drag to pan, wheel/pinch to zoom)
    if (layout === "tree" || layout === "outline") {
      const inner = layout === "tree" ? this.node(this.rootId) : this.outlineNode(this.rootId);
      this._fitEl = inner;
      const content = layout === "tree"
        ? h("div", { style: { padding: "118px 160px 200px" } }, inner)
        : h("div", { style: { padding: "120px 120px 140px 60px", display: "inline-block" } }, inner);
      const stage = h("div", { ref: (el) => (this.stage = el), style: { transform: "translate(" + this.state.tx + "px," + this.state.ty + "px) scale(" + this.state.scale + ")", "transform-origin": "0 0", "will-change": "transform" } }, content);
      const vp = h("div", { onMouseDown: (e) => this.onPanStart(e), onClick: () => this._canvasTap(), style: { position: "absolute", inset: "0", overflow: "hidden", cursor: "grab", background: bg, "touch-action": "none" } }, stage);
      this.vp = vp;
      vp.addEventListener("wheel", (e) => this._onWheel(e), { passive: false });
      vp.addEventListener("touchstart", (e) => this._onTouchStart(e), { passive: false });
      vp.addEventListener("touchmove", (e) => this._onTouchMove(e), { passive: false });
      vp.addEventListener("touchend", (e) => this._onTouchEnd(e));
      return vp;
    }
    this.vp = null;
    // explorer / columns. Desktop: reserve the detail-panel width on the right so
    // the last columns stay reachable. Mobile: master/detail above the sheet.
    const rightInset = this.state.selectedId != null && !this.isMobile() ? 420 : 0;
    const bottomInset = this.isMobile() && this.state.selectedId != null ? this.state.panelH + "px" : "0";
    return h("div", { onClick: () => this._canvasTap(), style: { position: "absolute", inset: "0", background: bg } }, h("div", { onClick: (e) => e.stopPropagation(), class: "kz-scroll", ref: (el) => (this.columnsEl = el), style: { position: "absolute", top: this._topbarH + "px", left: "0", right: rightInset + "px", bottom: bottomInset, "overflow-x": "auto", "overflow-y": "hidden" } }, this.columnsView()));
  },

  measureChrome() {
    const tb = this.topbarEl; if (!tb) return;
    const hh = tb.offsetHeight || 64;
    if (hh === this._topbarH) return;
    this._topbarH = hh;
    if (this.columnsEl) this.columnsEl.style.top = hh + "px";
  },

  render() {
    this.columnsEl = null; this.panelEl = null;
    // Guest viewing: the tree always renders; logged-out visitors browse read-only.
    const frame = h("div", { style: { position: "fixed", inset: "0", background: "#e9ddc2", color: "#3b2f21", overflow: "hidden" } },
      this.layoutLayer(), this.topbar(), this.searchSheet(), this.menuSheet(), this.bottomLeft(), this.panel(), this.inbox(), this.mySuggestions(), this.modal(), this.toastEl());

    const a = document.activeElement; let fk = null, ss, se;
    if (a && a.dataset && a.dataset.fkey) { fk = a.dataset.fkey; ss = a.selectionStart; se = a.selectionEnd; }
    this.root.replaceChildren(frame);
    if (fk) { const ne = this.root.querySelector('[data-fkey="' + fk + '"]'); if (ne) { ne.focus(); try { ne.setSelectionRange(ss, se); } catch (e) {} } }
    this.measureChrome();
    // columns: reveal the active (rightmost) column
    if (this.state.layout === "explorer" && this.columnsEl) this.columnsEl.scrollLeft = this.columnsEl.scrollWidth;
  },
};

window.App = App; // exposed for debugging / future backend wiring
window.addEventListener("DOMContentLoaded", () => App.init());
