# FreshTrack

FreshTrack is a Go, PostgreSQL, Redis, JWT, Docker, HTML/CSS/vanilla JavaScript warehouse receiving system.

## Features

- Central Admin: login/logout, dashboard, user CRUD/disable, warehouse CRUD, warehouse assignment, CSV/XLSX invoice upload, invoice visibility, reconciliation reports, CSV/Excel downloads, audit logs.
- Hub User: login/logout, assigned warehouse selection, assigned invoices, barcode scanning, manual increment/decrement, finish receiving, receiving history.
- Server-side warehouse isolation for hub users.
- JWT auth with role middleware and expiry.
- Redis duplicate scan protection and Redis stream backed scan processing.
- Invoice statuses: `Pending`, `Receiving`, `Completed`.
- Server-Sent Events for live progress updates.
- Dark, responsive vanilla frontend with sidebar navigation, tables, search, validation, loading states, toasts, and scan-focused controls.

## Requirements

- Go 1.22+
- Docker and Docker Compose
- PostgreSQL 16
- Redis 7

## Installation

```bash
cp .env.example .env
go mod tidy
```

For local Go execution, `.env.example` uses Postgres on host port `5433` because Docker maps `5433:5432`.

## Running With Docker

```bash
docker compose up --build
```

The app is served at `http://localhost:8080`.

Postgres and Redis are also exposed for local development:

- Postgres: `localhost:5433`
- Redis: `localhost:6379`

If you already have a `pgdata` volume from the old schema, recreate it before relying on the new migration:

```bash
docker compose down -v
docker compose up --build
```

## Running Locally

Start dependencies:

```bash
docker compose up postgres redis
```

Run the server:

```bash
go run ./cmd/server
```

Open `http://localhost:8080`.

## API

Auth:

- `POST /login` or `POST /api/auth/login`
- `POST /logout`
- `GET /me`

Admin:

- `GET /api/dashboard`
- `GET /api/users`
- `POST /api/users`
- `PUT /api/users/{id}`
- `DELETE /api/users/{id}`
- `GET /api/warehouses`
- `POST /api/warehouses`
- `PUT /api/warehouses/{id}`
- `DELETE /api/warehouses/{id}`
- `POST /api/upload`
- `GET /api/invoices`
- `GET /api/invoice/{id}`
- `GET /api/audit`
- `GET /api/reports`
- `GET /api/admin/reports/reconciliation?format=csv`
- `GET /api/admin/reports/reconciliation?format=xlsx`
- `POST /api/override`

Hub:

- `GET /api/hub-dashboard`
- `GET /api/hub/warehouses`
- `GET /api/hub/invoices?warehouse_id=WH-001`
- `POST /api/scan`
- `POST /api/manual-increment`
- `POST /api/manual-decrement`
- `POST /api/finish-receiving`
- `GET /api/progress?invoice_id=INV-001`
- `GET /api/hub/progress/events?invoice_id=INV-001&access_token=JWT`
- `GET /api/history`

## Invoice Upload Format

CSV and `.xlsx` files must contain these columns in the first sheet:

```text
Invoice_ID,Vendor_Name,Target_Warehouse_ID,Item_SKU,Item_Name,Expected_Quantity
```

Validation rejects the whole upload with row numbers when:

- warehouse does not exist
- invoice id already exists
- quantity is not greater than 0
- SKU is empty
- one invoice appears under multiple warehouses

## Frontend Pages

- `/login.html` and `/index.html`
- `/admin/dashboard.html`
- `/admin/users.html`
- `/admin/warehouses.html`
- `/admin/upload.html`
- `/admin/reports.html`
- `/admin/audit.html`
- `/hub/dashboard.html`
- `/hub/invoices.html`
- `/hub/scan.html`
- `/hub/history.html`

## Folder Structure

```text
cmd/server              application entrypoint
internal/auth           password hashing and JWT
internal/config         environment config
internal/db             pgx pool and migrations
internal/handlers       HTTP handlers
internal/middleware     JWT and role middleware
internal/models         shared API models
internal/queue          Redis stream scan worker
internal/redisclient    Redis setup
internal/router         route registration
web                     vanilla frontend
```

## Testing

```bash
go test ./...
go build ./cmd/server
docker compose config
```

## Screenshots

Run the app and open these pages:

- Admin dashboard: `http://localhost:8080/admin/dashboard.html`
- Scan screen: `http://localhost:8080/hub/scan.html`
# Inventory
