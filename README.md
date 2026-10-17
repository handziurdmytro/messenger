# Messenger

A web messenger with authentication, user profiles, contacts, chats, and file attachments.

The frontend is AI-generated.

## Architecture

Nginx serves the React frontend and routes API requests to four Go services:

- **auth-service** — authentication and JWT tokens.
- **user-service** — profiles and contacts.
- **chat-service** — chats and participants.
- **message-service** — messages and attachments.

Services communicate through NATS JetStream. Auth, users, and chats each have a separate PostgreSQL database; messages use ScyllaDB, and attachments use S3-compatible RustFS storage. Shared Go packages live in `pkg/`.

## Patterns

- **Hexagonal architecture** — domain logic and use cases connect to HTTP, messaging, and storage through ports and adapters.
- **Repository pattern** — storage interfaces separate use cases from database implementations.
- **Event-driven communication** — services publish and consume events through NATS JetStream.
- **Transactional outbox** — registration saves the user and event in one transaction; a worker publishes the event to NATS.
- **Database per service** — each service owns its data.

## Stack

- **Backend:** Go.
- **Frontend:** React, TypeScript, Vite.
- **Databases:** PostgreSQL, ScyllaDB.
- **Messaging:** NATS JetStream.
- **File storage:** RustFS (S3-compatible).
- **Infrastructure:** Nginx, Docker Compose.
- **Logging:** Grafana, Loki, Alloy.

## Run

Requires Docker with Compose, Task, and OpenSSL. From the repository root:

```sh
task keys
docker compose up -d --build
```

Open [localhost](http://localhost) for the app or [localhost:3000](http://localhost:3000) for Grafana. Database migrations run automatically during startup.
