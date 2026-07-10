# ADR-008 — Upgrade Next.js 15 → 16 (`apps/web`)

Status: Proposed (belum dieksekusi — dokumen desain untuk frontend engineer)
Tanggal: 2026-07-10
Penulis: System Architect (Zosmed)
Scope: Menaikkan `apps/web` dari **Next.js 15.1.x** ke **Next.js 16.x** beserta tooling terkait (ESLint config, React, `@types/*`, konvensi `middleware` → `proxy`, Turbopack default). **Murni perubahan frontend + tooling** — tidak menyentuh backend Go, tidak menambah/mengubah permukaan API Instagram (§4), tidak mengubah engine/Kit (§8). Satu risiko regresi utama yang dijaga: **proxy rewrites auth (ADR-003)** tidak boleh rusak.
Referensi: CLAUDE.md §5a (monorepo Bun+Turbo, boundary `apps` ↔ `packages`), §11 (design token dark+lime), §12a (prinsip coding), §14 (DoD). ADR-003 (auth cookie first-party lewat rewrites `/api/*` & `/connect/*`). ADR-004/005 (workflow builder + `@xyflow/react`).

> **Guardrail §4 (tidak ada yang berubah).** Upgrade ini **tidak** menambah/menghapus kapabilitas IG apa pun. Semua integrasi IG tetap di backend Go (`apps/api` → `graph.instagram.com`, §4.0). `apps/web` hanya UI; upgrade framework tidak menyentuh DO-NOT list §4b maupun safety layer §10. Yang wajib diverifikasi tetap jalan: **proxy rewrites first-party cookie `zsid`** (ADR-003), karena itulah satu-satunya jalur yang menyentuh sesi/keamanan.

---

## 0. Ringkasan Keputusan

Naik ke **Next.js `16.2.10`** (rilis stabil terbaru per Mei 2026; verifikasi ulang `next@latest` saat eksekusi). Perubahan yang **wajib** untuk repo ini hanya **dua**: (1) bump versi dependency, (2) rename `middleware.ts` → `proxy.ts` + rename fungsi. Sisanya sebagian besar **sudah kompatibel** karena repo ini sudah mengikuti pola Next 15 modern (async `params`/`searchParams`/`cookies()`, ESLint flat config, script `eslint .` bukan `next lint`, tidak pakai `next/image`, tidak pakai parallel routes, tidak pakai PPR/`revalidateTag`/AMP/`runtimeConfig`).

**Keputusan per area (ringkas):**

| Area | Status di repo saat ini | Keputusan | Alasan 1-kalimat |
|---|---|---|---|
| Versi `next`/React | `next ^15.1.6`, `react ^19.0.0` | Bump `next@^16.2.10`; bump `react`/`react-dom` ke `^19.2` (latest) | Peer Next 16 = `react ^19.0.0` (React 19 sudah terpasang → risiko rendah); ikut 19.2 untuk selaras runtime yang di-bundle Next 16. |
| Turbopack default | script sudah `next dev`/`next build` polos, **tanpa** custom webpack config | **Tidak perlu ubah script**; Turbopack aktif otomatis | Tidak ada `webpack()` di `next.config.ts` → build tak akan gagal; tak perlu flag `--turbopack` lagi. |
| `middleware.ts` | ada, coarse route guard cookie `zsid` | **Rename → `proxy.ts`**, fungsi `middleware` → `proxy`, `config.matcher` tetap | Konvensi baru Next 16; `middleware.ts` deprecated. Guard kita tak pakai fitur edge → aman di runtime `nodejs` proxy. |
| `next.config.ts` rewrites | `rewrites()` proxy `/api/*` & `/connect/*` | **Tidak perlu diubah** | `rewrites()` masih didukung penuh di Next 16; `transpilePackages` juga. |
| ESLint | flat config (`FlatCompat` + `compat.extends`), script `eslint .` | Bump `eslint-config-next`/`@next/eslint-plugin-next@^16`; config **tidak perlu diubah** | `next lint` sudah tidak dipakai; flat config sudah dipakai; `FlatCompat` masih berfungsi di v16. |
| async `params`/`cookies()` | sudah `Promise<...>` + `await` di semua route dinamis & server util | **Tidak perlu diubah** | Repo sudah pola Next 16; sinkron-akses (yang dihapus di 16) tidak dipakai. |
| `next/image` | **tidak dipakai** (hanya referensi tipe di `next-env.d.ts`) | **Tidak perlu diubah** | Semua breaking change `images.*` (qualities/localPatterns/minimumCacheTTL/dll) tidak relevan. |
| Node runtime | lokal v23.10, belum ada `engines`/`.nvmrc` | Tambah `engines.node >= 20.9.0` + `.nvmrc` | Menegakkan minimum Next 16 (Node 18 tidak didukung); mencegah drift versi antar mesin/CI. |

---

## 1. Verifikasi Breaking Changes (sumber resmi)

Diverifikasi dari blog & upgrade guide resmi Next.js (bukan memori):
- Blog rilis: <https://nextjs.org/blog/next-16>
- Upgrade guide 15→16: <https://nextjs.org/docs/app/guides/upgrading/version-16>
- `next@latest` = `16.2.10`, peer `react`/`react-dom` `^19.0.0`, `engines.node >=20.9.0` (npm registry).
- `eslint-config-next@latest` = `16.2.10`, peer `eslint >=9.0.0`, `typescript` optional `>=3.3.1`.

Pemetaan tiap breaking change ke repo ini:

| Breaking change Next 16 | Sumber | Dampak ke `apps/web` |
|---|---|---|
| **Turbopack jadi bundler default** `dev`+`build`; build **gagal** jika ada custom `webpack()` | blog §Turbopack (stable) | **Aman.** `next.config.ts` tak punya `webpack()`. Tak perlu flag. Uji build sekali (lihat §5). |
| **Node.js min 20.9.0** (Node 18 dihapus), **TS min 5.1** | upgrade guide §Version Requirements | Lokal v23.10 & `typescript ^5.7.3` sudah lolos. Tambah `engines`/`.nvmrc` agar CI konsisten. |
| **`middleware.ts` → `proxy.ts`** (deprecated, runtime `nodejs`, edge tak didukung di proxy) | upgrade guide §middleware to proxy | **Wajib rename** (lihat §3). Guard kita tak pakai fitur edge → aman. |
| **Sync `params`/`searchParams`/`cookies()`/`headers()`/`draftMode()` dihapus** — wajib async | upgrade guide §Async Request APIs | **Aman.** Sudah async semua (grep terbukti — §3 checklist). |
| **`next lint` dihapus**; `next build` tak lagi lint | upgrade guide §next lint | **Aman.** Script sudah `eslint .`. Tidak ada opsi `eslint` di `next.config.ts`. |
| **`@next/eslint-plugin-next` default ke Flat Config** | blog §Behavior Changes | **Aman.** Sudah flat config. Cukup bump versi. |
| **`next/image` defaults** (`qualities [75]`, `localPatterns` untuk query, `minimumCacheTTL` 4 jam, hapus `16` di `imageSizes`, blok local IP, max 3 redirect, `images.domains` deprecated) | upgrade guide §next/image | **Tidak relevan.** `next/image` tidak dipakai di kode app. |
| **Parallel routes wajib `default.js`** | upgrade guide §Parallel Routes | **Tidak relevan.** Tidak ada slot `@folder`. |
| **`revalidateTag(tag)` wajib argumen kedua** (`cacheLife` profile) | blog §Caching APIs | **Tidak relevan.** `revalidateTag` tidak dipakai (grep kosong). |
| **`experimental.ppr`/`experimental_ppr`/`dynamicIO`/`useCache` dihapus/ganti `cacheComponents`** | upgrade guide §PPR / removals | **Tidak relevan.** Tak ada flag eksperimental di `next.config.ts`. |
| **AMP dihapus** (`useAmp`, `config.amp`) | upgrade guide §AMP | **Tidak relevan.** Tidak dipakai. |
| **`serverRuntimeConfig`/`publicRuntimeConfig` dihapus** | upgrade guide §Runtime Configuration | **Tidak relevan.** Env dibaca via `lib/env` / `NEXT_PUBLIC_`. |
| **`experimental.turbopack` → top-level `turbopack`** | upgrade guide §Turbopack config location | **Tidak relevan.** Tak ada blok `turbopack`/`experimental` di config. |
| **`next dev` output ke `.next/dev`** (dev & build dir terpisah) + lockfile anti-instance-ganda | upgrade guide §Concurrent dev and build | Minor: pertimbangkan exclude `.next/dev/**` di `turbo.json` (lihat §3, opsional). |
| **`scroll-behavior: smooth` tak lagi di-override** default | upgrade guide §Scroll Behavior | **Tidak relevan** kecuali ada CSS `scroll-behavior: smooth` global — verifikasi cepat, tambah `data-scroll-behavior="smooth"` di `<html>` bila perlu. |

Fitur **opt-in** Next 16 (Cache Components/`"use cache"`, React Compiler, `updateTag`/`refresh`, FS cache Turbopack) **di luar scope** ADR ini — tidak diaktifkan. Upgrade ini murni "lift" ke 16 tanpa mengadopsi paradigma caching baru.

---

## 2. Versi Target Konkret

`apps/web/package.json`:

| Paket | Sekarang | Target | Catatan |
|---|---|---|---|
| `next` | `^15.1.6` | `^16.2.10` | Verifikasi `next@latest` saat eksekusi; pin minor terbaru. |
| `react` | `^19.0.0` | `^19.2.0` | Peer Next 16 minimal `^19.0.0`; ikut 19.2 (latest) untuk selaras. |
| `react-dom` | `^19.0.0` | `^19.2.0` | Ikut `react`. |
| `eslint-config-next` | `^15.1.6` | `^16.2.10` | Samakan mayor dengan `next`. |
| `@next/eslint-plugin-next` | `^15.1.6` | `^16.2.10` | Samakan mayor dengan `next`. |
| `@types/react` | `^19.0.7` | `^19.2.x` (latest 19) | Selaraskan dengan React 19.2. |
| `@types/react-dom` | `^19.0.3` | `^19.2.x` (latest 19) | Idem. |
| `@types/node` | `^22.10.7` | `^22` (tetap) atau bump `^24` | `^22` cukup (Node 20.9+). Boleh naik bila selaras Node CI. |
| `typescript` | `^5.7.3` | tetap (≥5.1 OK) | Tidak wajib naik. |
| `eslint` | `^9.18.0` | tetap (peer `>=9`) | Tidak wajib naik. |
| `tailwindcss` / `@tailwindcss/postcss` | `^4.0.0` | tetap | Tailwind v4 kompatibel dengan Turbopack; tak ada perubahan. |

**Root `package.json`:** `packageManager bun@1.3.13`, `turbo ^2.3.4` — **tidak berubah**. (Turbo 2.3.4 kompatibel; tak ada persyaratan turbo baru dari Next 16.)

React/Tailwind **tidak wajib naik mayor** — React tetap di 19 (bukan lompatan mayor), Tailwind tetap v4.

Cara eksekusi (opsi):
- Otomatis: `bunx @next/codemod@canary upgrade latest` (jalankan dari `apps/web`) — bisa bump versi, migrasi `middleware`→`proxy`, dan cek async API. **Wajib review diff manual** setelahnya karena repo pakai Bun workspaces + config bersama.
- Manual: edit `package.json` sesuai tabel → `bun install` di root → lakukan rename `proxy.ts` sendiri.

---

## 3. Peta Perubahan File (checklist konkret)

### WAJIB diubah

- [ ] **`apps/web/package.json`** — bump versi sesuai §2. Tambah blok:
  ```json
  "engines": { "node": ">=20.9.0" }
  ```
  Script `dev`/`build`/`start`/`lint`/`typecheck` **tetap** (tak ada `--turbopack`, `eslint .` dipertahankan).
- [ ] **`apps/web/middleware.ts` → rename ke `apps/web/proxy.ts`**:
  - Rename export `export function middleware(request: NextRequest)` → `export function proxy(request: NextRequest)`.
  - `export const config = { matcher: [...] }` **tetap sama persis** (matcher literal array tetap dianalisis statis).
  - Update komentar header: hapus klaim *"Runs on the edge"* — `proxy` berjalan di runtime **`nodejs`** (bukan edge). Logika guard (cek `zsid`, hanya klaim negatif, redirect ke `/login`) **tidak berubah** — tetap konsisten dengan ADR-003 §5.3 (authoritative check tetap di `(app)/layout.tsx` via `getMe()`).
  - Impor `NextResponse`/`NextRequest` dari `next/server` **tetap**.
- [ ] **`bun install`** di root (regenerasi `bun.lock`) — commit lockfile.
- [ ] **`.nvmrc`** (baru, di root atau `apps/web`) — mis. isi `22` (LTS ≥20.9) agar CI/kontributor seragam.

### Verifikasi / opsional

- [ ] **`apps/web/next.config.ts`** — **tidak perlu diubah**. `rewrites()` (proxy `/api/*` & `/connect/*`, ADR-003) dan `transpilePackages` (`@zosmed/{ui,types,kits}`) tetap didukung Next 16. Jangan tambah `turbopack {}` kecuali muncul kebutuhan `resolveAlias` (tidak diantisipasi).
- [ ] **`apps/web/eslint.config.mjs`** — **tidak perlu diubah**. `FlatCompat` + `compat.extends('next/core-web-vitals','next/typescript')` masih valid di `eslint-config-next@16`. (Opsional masa depan: migrasi ke import flat native `eslint-config-next/flat` bila `FlatCompat`/`@eslint/eslintrc` di-deprecate — **bukan** bagian upgrade ini.)
- [ ] **`apps/web/tsconfig.json`** — **tidak perlu diubah**. Plugin `next` & `include` `.next/types/**/*.ts` tetap. (Catatan: dengan `.next/dev`, tipe dev bisa berpindah; jika typecheck kehilangan tipe, jalankan `bunx next typegen` untuk regen — lihat §5.)
- [ ] **`turbo.json`** — **umumnya tidak perlu**. Task `build` outputs `.next/**` (exclude `.next/cache/**`) tetap valid; `dev` `cache:false` tak ber-output. *Opsional:* tambah `"!.next/dev/**"` ke outputs `build` agar artefak dev tak ikut ke-cache turbo.
- [ ] **Route dinamis & server util** — **tidak perlu diubah** (sudah async). Terverifikasi via grep:
  - `app/(app)/analytics/[workflowId]/page.tsx`, `app/(app)/contacts/[id]/page.tsx`, `app/(app)/workflows/[id]/page.tsx` → `params: Promise<...>` + `await params`.
  - `app/onboarding/page.tsx`, `app/(app)/settings/page.tsx` → `searchParams: Promise<...>`.
  - `lib/get-me.ts`, `lib/api/workflows.server.ts` → `await cookies()`.
- [ ] **`next/image`** — tak dipakai; abaikan seluruh breaking change `images.*`.
- [ ] **`<html>` di `app/layout.tsx`** — verifikasi cepat apakah ada CSS global `scroll-behavior: smooth`; bila ada dan diinginkan perilaku lama, tambah `data-scroll-behavior="smooth"`.

### TIDAK relevan (jangan buang waktu)

Parallel routes `default.js`, `revalidateTag`/`updateTag`/`cacheComponents`, PPR, AMP, `serverRuntimeConfig`/`publicRuntimeConfig`, `next lint`, `next/legacy/image`, `images.domains` — **tidak ada di repo** (grep kosong).

---

## 4. Diagram Alur — Dampak Upgrade (ASCII)

```
                 Browser (same-origin ke apps/web)
                          │
                          ▼
   ┌─────────────────── apps/web (Next.js 16, Turbopack default) ───────────────────┐
   │                                                                                 │
   │  proxy.ts  ← (dulu middleware.ts; runtime nodejs, BUKAN edge)                    │
   │    └─ cek cookie `zsid` → redirect /login (klaim NEGATIF saja, ADR-003)          │
   │                                                                                 │
   │  next.config.ts rewrites()  ── TIDAK BERUBAH (jalur kritis auth) ──┐            │
   │    /api/:path*     ─────────────────────────────────────────────┐  │            │
   │    /connect/:path* ─────────────────────────────────────────────┼──┼──► API_BASE│
   │                                                                  │  │  (apps/api)│
   │  (app)/layout.tsx → getMe()  [authoritative, server-side]        │  │            │
   └──────────────────────────────────────────────────────────────────┘  │            │
                          │                                               │
                          └───────── cookie `zsid` tetap FIRST-PARTY ─────┘
                                     (SameSite=Lax cukup — TIDAK boleh regresi)

   Yang berubah   : nama file middleware→proxy, versi paket, bundler (webpack→Turbopack)
   Yang TETAP     : rewrites proxy, transpilePackages, cookie flow, route guard logic,
                    async params/cookies, eslint flat config, Tailwind v4 PostCSS
```

---

## 5. Risiko & Titik Uji (langkah verifikasi)

| # | Risiko | Kenapa | Langkah verifikasi |
|---|---|---|---|
| R1 | **Proxy rewrites auth regresi** (ADR-003) — cookie `zsid` tak lagi first-party / OAuth connect putus | Jalur paling sensitif; upgrade bundler/proxy bisa mengubah handling header | Smoke: (a) login → cookie `zsid` ter-set same-origin; (b) hit `/api/*` dari browser → 200 lewat proxy, cookie terkirim; (c) alur `/connect/*` OAuth IG balik dengan sesi utuh; (d) tanpa cookie di route protected → redirect `/login`. |
| R2 | **`middleware`→`proxy` runtime nodejs** mengubah perilaku guard | Proxy jalan di `nodejs`, bukan edge | Uji redirect guard tetap jalan; pastikan tak ada dependensi edge-only. Matcher tetap meng-cover semua path §9. |
| R3 | **`@xyflow/react` (workflow builder) + Turbopack** | Bundler baus; xyflow punya CSS + banyak modul | Buka `/workflows/[id]` → canvas render, drag node, edge, inspector jalan (ADR-004/005). Cek console error bundling. |
| R4 | **`transpilePackages` workspace + Turbopack** | `@zosmed/{ui,types,kits}` ship TS/TSX source | `turbo build` sukses; komponen dari `packages/ui` (design token §11) ter-render; tipe `@zosmed/types` resolve. |
| R5 | **Tailwind v4 PostCSS + Turbopack** | Pipeline CSS di bundler baru | Verifikasi tema dark + aksen lime (§11) muncul benar di beberapa halaman kunci. |
| R6 | **Typecheck kehilangan tipe route** (`.next/types` pindah ke `.next/dev`) | Dev output dir berubah | Jika `tsc --noEmit` error tipe route, jalankan `bunx next typegen` lalu ulangi. |
| R7 | **CI Node version** | Belum ada `engines`/`.nvmrc` | Pastikan CI pakai Node ≥20.9; `engines` + `.nvmrc` menegakkan. |

### Urutan verifikasi (jalankan dari root, via Turbo/Bun)

1. `bun install` — resolusi dependency baru sukses, lockfile ter-update.
2. `bun run typecheck` (`turbo run typecheck`) — 0 error (jalankan `bunx next typegen` bila R6).
3. `bun run lint` (`turbo run lint` → `eslint .`) — 0 error/parse-failure (konfirmasi `eslint-config-next@16` termuat).
4. `bun run build` (`turbo run build` → `next build`, kini Turbopack) — build sukses tanpa fallback webpack error.
5. `bun run dev` → smoke test halaman kunci:
   - `/login` (guard R1/R2), `/dashboard` (redirect tanpa cookie), `/workflows/[id]` (xyflow R3), satu halaman ber-komponen `packages/ui` (R4/R5).
   - Uji end-to-end auth: login → session → `/api/*` & `/connect/*` lewat proxy (R1).

---

## 6. Acceptance Criteria (DoD §14)

1. `apps/web/package.json` memakai `next@^16`, `react`/`react-dom@^19.2`, `eslint-config-next@^16`, `@next/eslint-plugin-next@^16`, `@types/react(-dom)@^19.2`, plus `engines.node >=20.9.0`; `bun install` sukses & lockfile ter-commit.
2. `middleware.ts` di-rename ke `proxy.ts` dengan fungsi `proxy`; `config.matcher` identik; logika guard & referensi ADR-003 tak berubah; komentar "edge" dikoreksi ke runtime `nodejs`.
3. `next.config.ts` **tidak diubah**: `rewrites()` (`/api/*`, `/connect/*`) dan `transpilePackages` tetap; **AC kritis:** alur auth first-party cookie `zsid` (ADR-003) terbukti tidak regresi lewat smoke test R1.
4. `bun run typecheck`, `bun run lint`, `bun run build` (Turbopack default) semua **hijau**; `next build` tidak jatuh ke fallback webpack.
5. Halaman kunci lolos smoke test: guard `/login`↔protected, workflow builder `@xyflow/react`, komponen `packages/ui`, tema dark+lime §11.
6. Tidak ada fitur Next 16 opt-in (Cache Components, React Compiler, `revalidateTag` baru) yang diaktifkan — scope tetap "lift ke 16".
7. Guardrail §4 tak tersentuh: tak ada perubahan permukaan IG, tak ada item §4b, backend Go tak diubah. Copy default Bahasa Indonesia tetap. Prinsip §12a terjaga (tak ada duplikasi config baru; hanya rename + bump).

## 7. Non-Scope (ditunda)

- **Adopsi Cache Components / `"use cache"` / PPR** — paradigma caching baru; evaluasi terpisah pasca-upgrade.
- **React Compiler** (`reactCompiler: true`) — butuh Babel, memperlambat build; ukur dulu manfaatnya.
- **Turbopack FS cache dev** (`experimental.turbopackFileSystemCacheForDev`) — opsional performa, bukan bagian lift.
- **Migrasi ESLint ke flat-config native** (`eslint-config-next/flat`) tanpa `FlatCompat` — dilakukan bila `@eslint/eslintrc` di-deprecate (ESLint v10).
- **Upgrade Tailwind/React mayor lebih jauh** — di luar kebutuhan Next 16.
- **Next DevTools MCP** — tooling AI opsional, bukan syarat upgrade.

---

## 8. Catatan Implementasi (hasil eksekusi)

Versi final terpasang & verifikasi **hijau** (`typecheck`, `lint`, `build` Turbopack; smoke test guard `/login`↔protected 307 tidak regresi): `next@16.2.10`, `react`/`react-dom@19.2.0`, `@types/react@19.2.17`, `@types/react-dom@19.2.3`, `eslint-config-next`/`@next/eslint-plugin-next@16.2.10`, `engines.node >=20.9.0`, `.nvmrc` = `20.9.0`. Build memetakan guard sebagai `ƒ Proxy (Middleware)` → `proxy.ts` dikenali benar.

**Dua penyimpangan dari rencana (§3/§7), keduanya sudah dieksekusi:**

1. **`eslint.config.mjs` WAJIB diubah** (rencana §6.2 keliru menyebut "tidak perlu diubah"; §7 keliru menunda ke ESLint v10). `eslint-config-next@16` kini mengekspor **flat config array native** per sub-path (`eslint-config-next/core-web-vitals`, `.../typescript`), bukan lagi config eslintrc-shaped. `FlatCompat.extends(...)` lama membuat lint crash (`Converting circular structure to JSON`). Fix: import langsung kedua sub-path dan spread ke array config. Konsekuensi: `@eslint/eslintrc` jadi tak terpakai → dihapus dari `devDependencies`.
2. **Rule baru `react-hooks/set-state-in-effect`** (dari `eslint-plugin-react-hooks@7`, transitif `eslint-config-next@16`) menandai 2 baris di `Sidebar.tsx` (restore preferensi `localStorage` pasca-hydration + collapse route-driven). Keduanya pola legit → di-suppress dengan `eslint-disable-next-line` bertarget baris `setState` + komentar alasan, **tanpa perubahan perilaku UI**.

Dokumentasi terdampak diperbarui: `docs/how-to-run.md` (prasyarat Node ≥20.9 + catatan Next 16) dan `docs/specs/auth-login-onboarding.md` (referensi `middleware.ts` → `proxy.ts`).
