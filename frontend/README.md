# Messenger frontend

React 19 + TypeScript + Vite. Responsive dark interface using the requested green accent. All runtime data comes from the backend; fixtures exist only in browser tests.

## Run

Production, with all services:

```sh
docker compose up --build
```

Open http://localhost. The Nginx image builds and serves the frontend bundle itself, so no local Node.js process is needed after the image is built.

For frontend-only development:

```sh
cd frontend
npm ci
npm run dev
```

Open http://localhost:5173. The development server proxies `/api` and `/ws` to the existing gateway at `http://localhost:80`. Set `BACKEND_URL` in `.env.local` to change the gateway. See `.env.example` for optional deployed HTTP/WebSocket URLs. Never point the authenticated API directly at services that trust `X-User-ID`.

```sh
npm run typecheck
npm run lint
npm run build
PLAYWRIGHT_BROWSERS_PATH=.cache/browsers npx playwright install chromium --only-shell
npm test
```

The production bundle is built inside the Compose Nginx image. Nginx serves the React application from `/usr/share/nginx/html` and falls back to `index.html` for client-side routes.

## Features

- Email registration and login; serialized token refresh and expired-session handling.
- Profile name and avatar URL editing, user search, contact creation and listing.
- Direct conversations, history pagination, live text messages, persistence confirmations, and manual reconnect.
- Server-backed conversations, refreshed every 10 seconds, and per-contact local drafts; desktop and mobile layouts.
- Accessible labels, native modal focus management, keyboard sending, reduced-motion support, and actionable error/empty states.

Tokens and drafts use localStorage; sign-out clears tokens. Drafts remain on that browser for the same account. Conversations and messages are fetched from the backend. There is no logout/revocation endpoint, so sign-out is local.

## Integration

1. **WebSocket access tokens use a subprotocol header.** The gateway validates the browser-provided JWT from `Sec-WebSocket-Protocol` and passes only the resulting trusted user ID to the chat service.
2. **History authorization.** The message service checks membership through the chat service before reading history or an individual message. Authorization failures deny access; service failures return 503.
3. **Server inbox.** `GET /api/chat/` lists the caller's conversations; `/api/users/profiles/{id}` supplies participant details. The underlying service ports are internal to the Compose network because they trust gateway identity headers.
4. **Only text messages are wired into the UI.** The backend supports file-oriented message payloads, but the current user experience intentionally exposes only text messages.

Chat creation returns the existing private chat for the same participants. Search is debounced and cancels obsolete requests. Client frames are capped at 60 KiB, below the backend's 64 KiB limit. Message IDs reconcile persistence events without conflating identical text.

Compose uses the `messenger_messages` Scylla keyspace with tablets disabled, as required by the conditional writes on Scylla 6.0. The original `messenger` keyspace is retained. Both original message tables were verified empty before this deployment switched keyspaces. Other deployments with existing data must migrate it before changing `SCYLLA_KEYSPACE`; the script does not copy existing data. Current schema creation statements are repeatable; future schema changes require new migrations.

## Validation

Browser tests mock the exact inspected API contracts; they do not prove live backend integration. They cover registration errors, sign-in/out, profile changes, contact search/addition, history casing/pagination, incoming live messages, confirmations, persisted drafts, refresh serialization/expiry, and mobile disconnect behavior. The visual-review test writes desktop and mobile screenshots into `test-results/`.

Run the live integration check against the local Compose stack with:

```sh
PLAYWRIGHT_BROWSERS_PATH=.cache/browsers node tests/live-smoke.mjs
```

It creates three test accounts and a conversation, verifies browser delivery and history after reload, and verifies an outsider receives 403. Test records remain in the local database. Changes and validation are recorded in `../AGENTS/FRONTEND_CHANGES.md`.
