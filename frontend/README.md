# Meta Super App starter

Nuxt 4 + Vuetify starter with a server-side API proxy and HttpOnly-cookie authentication.

The application shell is responsive: fixed sidebar on desktop, compact rail on tablet, and bottom navigation with safe-area spacing on phones. Feature pages render inside `layouts/default.vue`; authentication pages use `layouts/auth.vue`.

## UI conventions

- Use semantic tokens from `assets/tokens.css`; do not place raw brand colors in feature components.
- Use Vuetify theme names (`primary`, `secondary`, `error`) in Vuetify props.
- Put reusable primitives in `components/ui`, application shell components in `components/app`, and feature-specific components in `components/<feature>`.
- Keep navigation metadata and required claims together in `config/navigation.ts`.
- Pages coordinate data and components; reusable UI and business logic belong in components and composables.

## Start

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Set `NUXT_API_BASE_URL` to the backend origin. The login endpoint returns per-account claims:

```json
{
  "access_token": "token",
  "user": {
    "id": 1,
    "name": "Name",
    "email": "user@example.com",
    "accounts": [{
      "account": { "id": "acc_1", "name": "My Company", "slug": "my-company" },
      "role": "admin",
      "claims": ["dashboard:read", "orders:read", "orders:update"]
    }]
  }
}
```

Browser code should call `/api/proxy/<backend-path>` (or `useApi('/proxy/...')`). The Nuxt server attaches the token; the token never enters client-side JavaScript.

Every proxied request includes a server-verified `X-Account-ID`. The server checks account membership and required claims before forwarding. The backend must repeat both tenant and permission checks; UI visibility is never an authorization boundary.

## Security notes

- Put secrets only in server runtime config; never use `NUXT_PUBLIC_` for secrets.
- Use HTTPS in production. The session cookie becomes `Secure` outside development.
- Restrict CORS on the backend and validate authorization for every resource there.
- Replace the in-memory rate limiter with Redis or an edge/WAF limit when horizontally scaling.
- Adapt upstream login/user response types and token refresh behavior to your backend contract.
