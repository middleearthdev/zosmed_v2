# Zosmed Frontend — Implementation Plan

> Derived from the hi-fi design master `Vendor.html` (see `SCREEN-INVENTORY.md`),
> the design tokens (`tokens.css`), the shared primitives (`primitives.jsx`), and
> the product spec (`CLAUDE.md`). This is a **plan**, not production code.
> Scope: **frontend only** — port ~23 artboard screens to Next.js + TS + Tailwind
> with mock/stub data. No Go backend is built here.

---

## 1. Scope & Assumptions

### In scope
- Bootstrap the monorepo skeleton per `CLAUDE.md` §5a (JS/TS side only; `go.work`
  + `apps/api` + `apps/worker` left for the backend track but their dirs reserved).
- `apps/web`: Next.js (App Router) + TypeScript strict + Tailwind.
- `packages/ui`: design-system port of `primitives.jsx` + `tokens.css` + shared
  app chrome (AppShell / Sidebar / Topbar) inferred from the screen set.
- `packages/types`: TS contract types aligned with domain model §6.
- `packages/config`: shared tsconfig / eslint / tailwind preset.
- `packages/kits`: Seller / Creator / Booking UI presets (Seller = MVP real,
  Creator/Booking = shells in this phase per roadmap §13).
- Port all ~23 screens to routes, wired to **mock data** living in the repo.
- Honor design tokens §11, IG constraints §4, BI olshop copy, DoD §14.

### Out of scope (explicitly)
- No Go/API/worker implementation, no real Instagram OAuth, no real webhooks.
- No real DB; all data is typed mock fixtures.
- No payment/QRIS, no ongkir API (fase lanjutan per §13/§12).
- No design-tool chrome (`DesignCanvas`, `DCSection`, `TweaksPanel`) — those are
  not app features (SCREEN-INVENTORY lines 6–10). The "tweaks" (accent/density/
  nodeStyle/grid) ship only as their **defaults**: lime / compact / soft.
- No real-time / IG-Live anything (§4b). "● LIVE" pill = workflow-running only.

### Working assumptions
- Each artboard's exact JSX is **pulled just-in-time** via DesignSync MCP
  (`get_file`, projectId `019dc928-acd7-7e0b-82e3-1b8f17387464`) when that screen
  is built. We plan from the inventory + tokens + primitives, not from on-disk JSX.
- Several artboard files export **multiple** screens (e.g. `workflow-dark.jsx` →
  Builder + Inspector + Runs; `kits-dark.jsx` → Creator + Booking). Confirm exports
  at pull time (see Risks §9).
- Artboards use **inline styles + hardcoded hex/oklch**; porting replaces these
  with tokens + `packages/ui` primitives (no inline-duplicated primitives).

### Token source-of-truth note (resolve before Phase 0)
`tokens.css` / `primitives.jsx` define `--zz-lime = oklch(0.9 0.2 130)` (a green),
while `CLAUDE.md` §11 lists the brand lime as `oklch(0.85 0.16 75)` (which equals
`--zz-warn` in `tokens.css`). The **design-reference files are the visual master**
for the port, so we adopt `oklch(0.9 0.2 130)` as `--zz-lime`. Flagged in §9 for a
one-line product confirmation; the Tailwind preset centralizes it so a change is
one edit.

---

## 2. Tech Stack & Tooling Decisions

| Concern | Decision | Rationale |
|---|---|---|
| Framework | **Next.js (App Router)**, React 19 | Per §5a; App Router for nested layouts (shared app shell) + route groups (marketing vs app). |
| Language | **TypeScript strict** everywhere | Hard requirement; `strict: true` + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes`. |
| Styling | **Tailwind CSS v4** | Native `oklch()` support, CSS-first `@theme` that maps directly onto `tokens.css` variables, no JS config drift. Tokens are already oklch → v4 is the natural fit. |
| Package manager / runtime | **Bun** workspaces | Per §5a (`bun install`, `bun run`). |
| Monorepo orchestration | **Turborepo** (`bunx turbo run build|lint|test`) | Per §5a. |
| Fonts | **Geist Sans + Geist Mono** via `next/font` | `--font-sans` / `--font-mono` per §11; `geist` package or `next/font/google`. |
| Lint/format | ESLint (flat config) + Prettier, shared from `packages/config` | DRY config §12a. |
| Data fetching | **Mock only**: typed fixtures + thin async stubs (so swapping to real fetch later is local). | Frontend-only scope. |
| Icons | Ported `I` icon set → typed `packages/ui` icon components | Reuse, no extra icon lib. |
| Charts (analytics) | Lightweight SVG (port artboard markup) or `recharts` if artboard implies real charts | Decide at Analytics pull; prefer porting static SVG to avoid premature dep. |

### How tokens map to Tailwind (v4)
1. Ship `tokens.css` (CSS variables) as the single source — copied into
   `packages/ui/src/styles/tokens.css`, imported once at app root.
2. In the global stylesheet, an `@theme inline { ... }` block re-exposes each
   `--zz-*` variable as a Tailwind theme token, e.g.
   `--color-bg: var(--zz-bg)`, `--color-lime: var(--zz-lime)`,
   `--color-text: var(--zz-text)`, `--font-mono: var(--font-mono)`. This yields
   utilities `bg-bg`, `text-text-2`, `border-line`, `text-lime`, `font-mono`, etc.
3. Utility classes from `tokens.css` (`.mono`, `.tnum`, `.tracked`, `.btn-lime`,
   `.btn-ghost`, `.zz-placeholder`, `.zz-scroll`) are kept as `@layer components`
   OR re-expressed as `packages/ui` components (`<Button variant="lime">`,
   `<Placeholder/>`). Prefer components; keep `.mono`/`.tnum`/`.tracked` as
   utilities since they are pure typography.
4. No hardcoded hex/oklch in screens — only token utilities or `var(--zz-*)`.

---

## 3. Monorepo Bootstrap Steps (ordered, concrete)

> Do not touch / create `go.work`, `apps/api`, `apps/worker` contents — only
> reserve the dirs. JS/TS workspace lives alongside.

**Step 0 — root files**
- `/package.json` — root, private, `"workspaces": ["apps/web","packages/*"]`,
  scripts delegating to turbo (`dev`, `build`, `lint`, `typecheck`).
- `/turbo.json` — pipeline: `build` (dependsOn `^build`), `lint`, `typecheck`,
  `dev` (persistent, no cache).
- `/.gitignore`, `/.npmrc` or `bunfig.toml` if needed.
- `/tsconfig.json` — root references base.
- Leave `/go.work` untouched (created by backend track later).

**Step 1 — `packages/config`** (shared, no deps on others)
- `packages/config/package.json` (name `@zosmed/config`).
- `packages/config/tsconfig.base.json` — strict flags listed in §2.
- `packages/config/eslint.config.mjs` — flat config (next, ts, react-hooks).
- `packages/config/tailwind-preset.ts` — Tailwind v4 preset re-exporting the
  `@theme` token mapping (or a shared `tokens.css` import path).
- `packages/config/prettier.config.mjs`.

**Step 2 — `packages/ui`** (depends on config)
- `packages/ui/package.json` (name `@zosmed/ui`, exports `./*`).
- `packages/ui/src/styles/tokens.css` (copy of design-reference tokens.css).
- `packages/ui/src/styles/globals.css` (imports tokens + `@theme inline` mapping
  + `@layer components` for `.btn-*`, `.zz-*`).
- `packages/ui/tsconfig.json` (extends base).
- Primitive + chrome components (see §4).

**Step 3 — `packages/types`** (depends on config only)
- `packages/types/package.json` (name `@zosmed/types`).
- `packages/types/src/*` domain contracts (see §7).

**Step 4 — `packages/kits`** (depends on ui, types)
- `packages/kits/package.json` (name `@zosmed/kits`).
- `packages/kits/src/seller|creator|booking/` preset descriptors + UI.

**Step 5 — `apps/web`** (depends on ui, types, kits, config)
- `apps/web/package.json` (name `@zosmed/web`, next/react deps + workspace deps).
- `apps/web/next.config.ts`, `apps/web/tsconfig.json` (extends base + paths).
- `apps/web/postcss.config.mjs` (Tailwind v4 plugin) + `app/globals.css`
  importing `@zosmed/ui/styles/globals.css`.
- `apps/web/app/layout.tsx` — root layout: html/body, Geist fonts via `next/font`,
  `dark` class, global css.
- `apps/web/app/page.tsx` — temporary redirect to `/dashboard` (or landing).
- Mock data dir `apps/web/lib/mock/` (see §7).

**Step 6 — verify** `bun install`, `bunx turbo run typecheck lint`, `bun run dev`
renders an empty shell at `/dashboard`. Gate for Phase 1.

---

## 4. `packages/ui` — Design-System Port

### 4a. Primitives (1:1 from `primitives.jsx`, typed, tokenized)
Each becomes a `.tsx` with a typed props interface; inline hex/oklch → tokens.

| Primitive | Port notes / props |
|---|---|
| `Logo` | `{ size?: number; theme?: 'dark'\|'light'; showWord?: boolean }`. SVG kept; `ZZ_LIME` literal → `var(--zz-lime)`. |
| `Pill` | `{ tone?: 'lime'\|'neutral'\|'warn'\|'pink'\|'blue'; children; className? }`. Tone map → token-based classes, not inline objects. Used for status incl. "● LIVE" (workflow-running). |
| `Dot` | `{ color?: string; size?: number }` → token default `var(--zz-lime)`. |
| `I` (icon set) | Export as typed record `Icon: Record<IconName, (p?: SVGProps)=>JSX>` OR individual components `<ArrowIcon/>`. Keep all 24 glyphs (arrow, bolt, chat, filter, send, ai, chart, workflow, inbox, user, cog, plus, search, heart, check, sparkle, shield, bell, users, whatsapp, calendar, tag, box, live). `IconName` union type. |
| `Placeholder` | `{ label?: string; height?: number; className? }`. Striped bg → token border/colors. |
| `Avatar` | `{ name?: string; color?: string; size?: number }`. Initials, mono. |
| `ZZ_LIME` const | Replaced by token; export `LIME = 'var(--zz-lime)'` only if a JS literal is unavoidable. |

### 4b. Shared atoms to extract (appear across many screens → DRY)
Inferred from tokens.css utility classes + recurring screen patterns:
- `Button` — `variant: 'lime'|'ghost'`, `size`, `icon?`, replaces `.btn-lime`/`.btn-ghost`.
- `Card` — rounded panel (`bg-2`/`bg-3`, `border-line`, radius) — the dominant container.
- `Stat` / `StatCard` — metric + label + delta (dashboard, analytics, safety).
- `Gauge` / `Meter` — quota bars ("200/200 dm·hr") for Safety/Dashboard (§10 UI).
- `Tag` / `Chip` — contact tags, keyword chips.
- `SearchInput`, `Field`/`Input`, `Toggle`, `Select` — forms (settings, builder).
- `SectionHeader` — title + actions row (used atop most app screens).
- `EmptyState` — icon + copy + CTA (states screen + reused everywhere).
- `Table` / `DataRow` — contacts, runs, billing, team.

> Apply rule-of-three (§12a): extract an atom only once ≥2 screens need it. Start
> with the clearly-shared set above; defer speculative atoms.

### 4c. App chrome (inferred from the app screen set — all share a shell)
- `AppShell` — grid: fixed Sidebar + Topbar + scrollable content (`zz-scroll`).
  Wraps every authenticated screen via App Router layout.
- `Sidebar` — nav in §9 order: dashboard · workflows · inbox · ai · contacts ·
  analytics · safety · templates · settings · team · notifications. Logo at top,
  active-route highlight (lime), icons from `I`. Account/Kit switcher at bottom.
- `Topbar` — page title, global search, notifications bell, account menu,
  optional Kit indicator + global **kill-switch** affordance (§10).
- `KitProvider` (context) — current segment (seller/creator/booking) → drives
  Kit-specific nav items/presets. Lives in `apps/web` but consumes `packages/kits`.

### 4d. Token CSS
- `packages/ui/src/styles/tokens.css` (verbatim copy) + `globals.css` with the
  `@theme inline` mapping and component layers (§2). Single import point.

---

## 5. Routing Map (App Router)

Two route groups under `apps/web/app`:
- `(marketing)` — public, no app shell (landing).
- `(app)` — authenticated app, wrapped by `AppShell` layout.
- `onboarding` — standalone flow (no full shell; minimal chrome).

| # | Screen | Route | Key components |
|---|---|---|---|
| 1 | Landing | `/(marketing)/` → `/` | `LandingDark` sections, `Logo`, `Button`, `Pill`, marketing-only blocks |
| 2 | Onboarding (4 steps) | `/onboarding` | `OnboardingStepper`, segment picker (jualan/edukasi/jasa → loads Kit §8), IG-connect stub |
| 3 | Dashboard | `/(app)/dashboard` | `AppShell`, `StatCard`, `Gauge`, activity feed, `Pill` LIVE |
| 4 | Inbox (threaded chat) | `/(app)/inbox` | thread list, `ChatThread`, message composer, WA-handoff CTA, `Avatar` |
| 5 | Workflows — Builder | `/(app)/workflows` (or `/workflows/[id]`) | node canvas, node palette (§7 catalog), `WorkflowNode` |
| 6 | Workflows — Inspector | `/(app)/workflows/[id]` (inspector panel) | node inspector side-panel, `Field`, `Toggle` |
| 7 | Workflows — Runs | `/(app)/workflows/[id]/runs` | `Table`/run log rows, `Pill` status, RunLog mock |
| 8 | Comment-to-Order | `/(app)/workflows/comment-to-order` | reservation board, countdown, status pills (`reserved`→`waiting-pay`→`closed-wa`→`expired-released`) |
| 9 | AI Studio | `/(app)/ai` | persona editor, prompt/few-shot, test console |
| 10 | Contacts | `/(app)/contacts` | `Table`, tags/chips, search, window-24h status |
| 11 | Contact — Profile | `/(app)/contacts/[id]` | profile header, timeline, tags, conversations |
| 12 | Analytics | `/(app)/analytics` | charts, `StatCard`, period selector |
| 13 | Analytics — Drilldown | `/(app)/analytics/[workflowId]` | per-workflow funnel/attribution |
| 14 | Templates | `/(app)/templates` | template grid/cards, category filter |
| 15 | Safety center | `/(app)/safety` | quota `Gauge`s (§10), auto-pause log, kill switch |
| 16 | Notifications | `/(app)/notifications` | notification list, read/unread |
| 17 | Team | `/(app)/team` | member `Table`, roles, invite |
| 18 | Settings | `/(app)/settings` | settings sections, account/IG connection |
| 19 | Settings — Billing | `/(app)/settings/billing` | plan card, usage, invoices (no real payment) |
| 20 | Seller Kit | `/(app)/kits/seller` | Kit Center: keep/C, trust-kit, commerce calendar presets |
| 21 | Creator Kit | `/(app)/kits/creator` | lead-magnet, link-in-DM, waitlist presets (shell) |
| 22 | Booking Kit | `/(app)/kits/booking` | comment-to-booking, calendar handoff presets (shell) |
| 23 | Empty & error states | `/(app)/states` (showcase) + reused inline | `EmptyState`, error/404/500 variants |

Notes:
- A **Kit Center** index `/(app)/kits` lists active Kits (§9). Individual kit pages
  20–22 are its children.
- `not-found.tsx` / `error.tsx` at the `(app)` level reuse the `EmptyState` atom.

---

## 6. Screen Build Order / Phasing

> Each phase ends with `typecheck + lint` green and screens visually matching the
> artboard at lime/compact/soft defaults. Maps to roadmap §13 (MVP = Seller Kit).

**Phase 0 — Foundation** (§3 + §4d + §7 types)
- Monorepo bootstrap, Tailwind/token wiring, fonts, empty `AppShell` at `/dashboard`.
- `packages/types` domain contracts + `apps/web/lib/mock` skeleton.
- Dep gate for everything else.

**Phase 1 — Shell + Dashboard** (screens 3)
- `AppShell`, `Sidebar`, `Topbar`, `KitProvider`.
- Extract first shared atoms (`Card`, `StatCard`, `Gauge`, `Button`, `SectionHeader`).
- Dashboard screen (#3) — the canonical consumer that validates the shell + atoms.

**Phase 2 — Workflows cluster** (screens 5, 6, 7, 8) — depends on Phase 1
- Builder (#5) → Inspector (#6) → Runs (#7) → Comment-to-Order (#8).
- Highest complexity; node canvas + reservation state machine UI (mock).
- Pull `workflow-dark.jsx` once; it likely exports 5/6/7 (confirm).

**Phase 3 — Inbox** (screen 4) — depends on Phase 1
- Threaded chat, composer, WA-handoff CTA (renders `wa.me` deep-link, no API).
- Source `menu-details-dark.jsx` (confirm).

**Phase 4 — Contacts & Analytics** (screens 10, 11, 12, 13) — depends on Phase 1
- Contacts list + profile; Analytics + drilldown. Extract `Table`, chart atoms.

**Phase 5 — System screens** (screens 9, 14, 15, 16, 17, 18, 19)
- AI Studio, Templates, Safety (#15 surfaces §10 gauges/log/kill-switch),
  Notifications, Team, Settings, Billing.

**Phase 6 — Kits** (screens 20, 21, 22) — depends on `packages/kits` + Phase 5
- Seller Kit (#20) = MVP real content (§13). Creator (#21) + Booking (#22) =
  functional **shells** with preset descriptors (phase-2 features, UI present).

**Phase 7 — States, Landing, Onboarding** (screens 1, 2, 23)
- `EmptyStates` (#23) atom showcase (build early-ish if reused, finalize here),
  Landing (#1, marketing group), Onboarding (#2, segment picker → Kit load).

Dependency summary: Phase 0 → 1 → {2,3,4,5} (parallelizable after 1) → 6 → 7.

---

## 7. Shared Data / Types Strategy

### `packages/types` — contracts aligned to domain model §6
One file per aggregate, all exported from `packages/types/src/index.ts`:
- `Account` — IG business/creator connection: `{ id, handle, displayName, status, connectedAt, kit: Segment }`.
- `Workflow` — `{ id, name, status: 'draft'|'live'|'paused', nodes: WorkflowNode[], edges, updatedAt }`.
- `WorkflowNode` — discriminated union by §7 catalog: trigger/filter/action kinds (`type`, `config`). `Segment = 'seller'|'creator'|'booking'`.
- `Contact` — `{ id, igUserId, name, tags: string[], windowState: 'open'|'closed', lastSeen, source }`.
- `Conversation` — `{ id, contactId, source: 'comment'|'dm'|'story', windowState, messages: Message[] }`.
- `Reservation` — `{ id, code, product, status: 'reserved'|'waiting-pay'|'closed-wa'|'expired-released', expiresAt }`.
- `OptIn` — `{ id, contactId, topic, optedInAt }`.
- `RunLog` / `Event` — `{ id, workflowId, nodeId, status, message, at }`.
- `TrustAsset` — `{ id, kind: 'testimoni'|'real-pict'|'resi', url, label }`.
- Supporting: `SafetyQuota` (§10 gauges: commentReplies/hr, dm/hr, dm/day,
  commentsPerPost/5min, aiTokens/day), `Notification`, `TeamMember`, `Template`,
  `AnalyticsMetric`.
- Constants module: rate-limit caps (§4c), reservation statuses, Kit keywords —
  defined **once** here and imported everywhere (DRY §12a).

### Mock data
- Location: `apps/web/lib/mock/<aggregate>.ts`, each typed against `@zosmed/types`.
- Thin async accessors `apps/web/lib/mock/api.ts` (`getDashboard()`, `listContacts()`)
  returning typed promises → so screens use server/client components as they will
  with a real API; swap-in later is local.
- BI olshop copy in fixtures (names, products, "keep"/"C1", testimoni text).
- No IG-Live-derived fields anywhere (§4b): no follower-count-trigger, no live
  viewer count, no follow-status. Lint/PR check enforced via §8 checklist.

---

## 8. Per-Screen Porting Checklist (repeatable)

For each screen, in order:
1. **Pull source** — DesignSync MCP `get_file` on the artboard file (projectId in
   SCREEN-INVENTORY). Confirm which exported component is this screen (multi-export
   files). Strip design-tool wrappers (`DCArtboard`, etc.).
2. **Create route** — add the `app/.../page.tsx` per §5 map; wrap in `(app)` shell
   unless marketing/onboarding.
3. **Replace inline styles** — every hex/oklch → token utility (`bg-bg-2`,
   `text-text-2`, `border-line`, `text-lime`) or `var(--zz-*)`. No literals.
4. **Swap primitives** — replace inline Logo/Pill/Dot/icons/Placeholder/Avatar with
   `@zosmed/ui` imports. Never re-inline a primitive (DRY §12a).
5. **Extract sub-components** — repeated blocks → `packages/ui` atom (if ≥2 screens)
   or a local `_components/` (if screen-local). Rule-of-three; no premature abstraction.
6. **Type props** — strict TS interfaces; data props typed against `@zosmed/types`.
7. **Wire mock data** — pull from `apps/web/lib/mock`; no hardcoded data in JSX.
8. **Copy pass** — default copy Bahasa Indonesia, olshop tone (§2/§12).
9. **Constraint audit (§4)** — verify no DO-NOT capability (§4b) is implied: no IG
   Live data, no follower-trigger, no follow-status, no mass-DM blast. Confirm any
   "LIVE" pill = workflow-running (§9). Outbound IG affordances framed via safety
   layer / window-24h where shown (§10).
10. **DoD check (§14 + §11)** — tokens consistent, responsive at artboard width,
    `typecheck + lint` green, SoC respected (presentational vs data hook vs types).

---

## 9. Risks & Open Questions

1. **Token/lime discrepancy** (§1): `--zz-lime` is `oklch(0.9 0.2 130)` in the
   design master vs `oklch(0.85 0.16 75)` in CLAUDE.md §11. → Confirm brand lime;
   plan adopts the design-master value (centralized, one-edit fix).
2. **oklch + Tailwind v4 + browser support** — v4 emits oklch natively; ensure a
   fallback strategy isn't needed for target browsers (modern only — acceptable).
   If v4 friction appears, fallback is Tailwind v3 + `tailwind.config` mapping the
   same CSS vars (lose native oklch ergonomics but works).
3. **Multi-export artboard files** — `workflow-dark.jsx`, `kits-dark.jsx`,
   `menu-*-dark.jsx`, `analytics-dark.jsx` each hold several screens; exact
   component→screen mapping confirmed only at pull time (SCREEN-INVENTORY `*` rows).
4. **Unknown source files** — rows marked `artboards/*?*` (AI Studio, Contacts,
   Templates, Safety, Notifications, Team, Settings, Billing, States) need file
   discovery via DesignSync before their phase.
5. **Charts** — Analytics may need a charting lib; decide port-static-SVG vs
   `recharts` at pull (avoid premature dep).
6. **Node canvas** — Workflow Builder (#5) may need pan/zoom/drag; for frontend
   mock, decide static-canvas vs interactive (likely static-first, interactivity
   later). Highest-effort screen — budget accordingly.
7. **MVP-critical set** — per §13, prioritize Dashboard, Workflows+Comment-to-Order,
   Inbox, Safety, AI Studio, Seller Kit, Onboarding. Creator/Booking Kits are
   shells; Templates/Team/Billing can trail if time-boxed.
8. **Responsive scope** — artboards are fixed-width (1440/1280). Confirm whether MVP
   targets desktop-only or needs responsive down to mobile (affects layout effort).
```
