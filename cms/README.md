# Omni Channel CMS

NextJS administration CMS for the Omni Channel platform. The first version covers login, dashboard, users, roles, permissions, teams, channel accounts, and audit logs.

## Run Locally

```bash
npm install
npm run dev
```

Default API host:

```txt
NEXT_PUBLIC_API_BASE_URL=http://localhost:18080
```

Copy `.env.example` to `.env.local` when the API runs on a different host.

## Design Structure Rules

- Use App Router pages under `app/`; admin surfaces live in `/dashboard` and `/admin/*`.
- Keep shared UI in `components/`, typed API access in `lib/api.ts`, auth/session behavior in `lib/auth.tsx`, theme state in `lib/theme.tsx`, sidebar state in `lib/sidebar.tsx`, and backend contracts in `lib/types.ts`.
- Prefer dense, work-focused screens inspired by Chatwoot: fixed sidebar, compact top bar, bordered panels, scannable tables, and clear status pills.
- Keep colors token-based through `globals.css` CSS variables. All admin components should use `--app-*` tokens so light, dark, and system themes work consistently.
- Use 6px radius or less for panels/buttons by default. Do not nest decorative cards inside cards; reserve panels for repeated data groups, modals, and framed tools.
- Sidebar open/closed state and theme preference are persisted in localStorage. Preserve mobile overlay behavior and desktop collapsed icon-only behavior.
- Add motion through small transition/animation utilities such as `page-motion` and `menu-motion`; avoid large decorative motion in operational screens.
- Tables must support horizontal overflow, fixed row spacing, concise cells, and an explicit empty state.
- Forms use `FormField`, native inputs/selects, and modal submit flows. Destructive actions must use backend soft-disable endpoints where available.
- Render RBAC-aware content from `/api/auth/profile`; do not trust frontend checks as authorization. Backend permission checks remain the source of truth.
- API calls must go through `lib/api.ts`, include Bearer tokens only when needed, and never expose password hashes or raw channel credentials.
- Channel account screens must use `/api/channel-admin/*`, hide the old channel registry table, and support shared teams/users for message visibility.
- Keep page copy short and operational. Avoid marketing text, oversized hero sections, and decorative illustrations in admin workflows.

## Implemented Routes

- `/login`
- `/dashboard`
- `/admin/users`
- `/admin/roles`
- `/admin/teams`
- `/admin/channels`
- `/admin/audit`
