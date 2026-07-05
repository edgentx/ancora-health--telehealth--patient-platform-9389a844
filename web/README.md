# Ancora Health — Web (Next.js App Router)

Multi-surface, role-based frontend for the Ancora Health telehealth platform.
The Go backend (`../cmd/server`) serves the API/WS; this app renders the four
role surfaces and gates UI by the role resolved from **trusted edge headers**.

## Role model

Authentication happens at the edge. The proxy injects the resolved identity as
trusted headers; this app **never parses a JWT or runs its own RBAC**. It reads
the headers server-side (`src/lib/identity.ts`) and routes/gates the UI by role.

| Header (configurable) | Purpose                                         | Default         |
| --------------------- | ----------------------------------------------- | --------------- |
| `ANCORA_ROLE_HEADER`  | resolved role: patient/provider/scheduler/admin | `x-ancora-role` |
| `ANCORA_USER_HEADER`  | resolved user display name / id                 | `x-ancora-user` |

Landing routes (one per surface, exercised by the smoke test):

- patient → `/dashboard`
- provider → `/provider/dashboard`
- scheduler → `/scheduler/dashboard`
- admin → `/admin/dashboard`

## Structure

```
src/
  app/
    layout.tsx            root: fonts, providers, identity resolution
    page.tsx              public marketing home (/)
    (patient)/            patient surface (route group) + landing
    (provider)/           provider surface + landing
    (scheduler)/          front-desk/scheduler surface + landing
    (admin)/              clinic admin surface + landing
  components/
    providers.tsx         TanStack Query client + Zustand hydration (root)
    app-shell.tsx         shared shell + role-scoped navigation + role gate
    role-badge.tsx        client consumer of the hydrated store
  lib/
    env.ts                API/WS base URLs + trusted-header names from env
    roles.ts              role model + navigation map (mirrors design-system/navigation.md)
    identity.ts           resolveIdentity() from trusted headers (server-only)
  store/
    ui-store.ts           Zustand store (role mirror + UI state)
```

## Commands

```bash
npm install
npm run dev          # local dev server
npm run type-check   # tsc --noEmit (strict mode)
npm run lint         # next lint (ESLint + prettier config)
npm run format       # prettier --write
npm run build        # optimized production build
npm run smoke        # boot `next start` and render every role landing route
```

## Environment

Copy `.env.example` to `.env.local` and adjust as needed. API/WS base URLs are
`NEXT_PUBLIC_*` (browser-visible); the trusted-header names are server-only.
