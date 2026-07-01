
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS warehouses (
    id          TEXT PRIMARY KEY,       
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('central_admin', 'hub_user')),
    disabled        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_warehouses (
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    warehouse_id  TEXT NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, warehouse_id)
);

CREATE TABLE IF NOT EXISTS invoices (
    invoice_id    TEXT PRIMARY KEY,
    vendor_name   TEXT NOT NULL,
    warehouse_id  TEXT NOT NULL REFERENCES warehouses(id),
    status        TEXT NOT NULL DEFAULT 'Pending' CHECK (status IN ('Pending', 'Receiving', 'Completed')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invoice_lines (
    id                  BIGSERIAL PRIMARY KEY,
    invoice_id          TEXT NOT NULL REFERENCES invoices(invoice_id) ON DELETE CASCADE,
    item_sku            TEXT NOT NULL,
    item_name           TEXT NOT NULL,
    expected_quantity   INTEGER NOT NULL CHECK (expected_quantity > 0),
    UNIQUE (invoice_id, item_sku)
);

CREATE TABLE IF NOT EXISTS receiving_ledger (
    invoice_id       TEXT NOT NULL REFERENCES invoices(invoice_id) ON DELETE CASCADE,
    item_sku         TEXT NOT NULL,
    received_quantity INTEGER NOT NULL DEFAULT 0,
    last_updated     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (invoice_id, item_sku)
);

CREATE TABLE IF NOT EXISTS audit_log (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL DEFAULT now(),
    invoice_id   TEXT NOT NULL,
    item_sku     TEXT NOT NULL,
    warehouse_id  TEXT NOT NULL,
    user_id      UUID NOT NULL REFERENCES users(id),
    action_type  TEXT NOT NULL, 
    old_quantity INTEGER NOT NULL,
    new_quantity INTEGER NOT NULL,
    delta        INTEGER NOT NULL,
    reason       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_invoices_warehouse ON invoices(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_invoice_lines_invoice ON invoice_lines(invoice_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_invoice ON audit_log(invoice_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_ts ON audit_log(ts);
