# Messenger frontend

React 19 + TypeScript + Vite. Minimal neutral interface with automatic light and dark themes (`prefers-color-scheme`). All runtime data comes from the backend; fixtures exist only in browser tests.

## Run

Production, with all services:

```sh
docker compose up --build
```

Open http://localhost. The Nginx image builds and serves the frontend bundle itself.

Frontend-only development:

```sh
cd frontend
npm ci
npm run dev
```

Open http://localhost:5173. The dev server proxies `/api` and `/ws` to the gateway at `http://localhost:80` (`BACKEND_URL` in `.env.local` overrides it).

## Checks

```sh
npm run typecheck
npm run lint
npm run build
PLAYWRIGHT_BROWSERS_PATH=.cache/browsers npx playwright install chromium --only-shell
npm test
```

`npm test` runs browser tests against an in-memory fake backend (`tests/fake-backend.ts`) that intercepts `/api` and the WebSocket.

`node tests/live-smoke.mjs` drives three real accounts against a running stack (`MESSENGER_URL`, default `http://localhost`): private chat, live delivery, unread badge, read receipts, edit/delete, group chat and blocking. Test accounts (`smoke-<timestamp>-*`) are kept.

## Structure

- `src/api` — typed clients per service; token refresh is serialized in `client.ts`.
- `src/realtime/socket.ts` — one WebSocket per session with exponential reconnect; the JWT travels as the WebSocket subprotocol.
- `src/hooks/useMessenger.ts` — workspace state: chats, profiles cache, previews, unread counts and live event handling.
- `src/components` — screens and dialogs.

## Features

- Registration, login, logout with refresh-token revocation, sign out on all devices, password change, account deletion.
- Profile editing, people search, contacts (add, remove), blocking and the blocked list; deleted accounts are shown as “Deleted account”.
- Private and group chats, cursor-paginated chat list with previews and unread badges, group rename and member management, leaving and deleting chats.
- Live messages, edits, deletions and read receipts over the shared WebSocket; history pagination; editing and deleting own messages; per-chat drafts.
