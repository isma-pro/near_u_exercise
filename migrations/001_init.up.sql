CREATE TABLE funds (
    id TEXT PRIMARY KEY,
    currency TEXT NOT NULL,
    timezone TEXT NOT NULL,
    cutoff_hour INT NOT NULL,
    cutoff_minute INT NOT NULL
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    cash BIGINT NOT NULL,
    reserved_cash BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE positions (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    fund_id TEXT NOT NULL REFERENCES funds(id) ON DELETE CASCADE,
    units BIGINT NOT NULL,
    reserved_units BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, fund_id)
);

CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id),
    fund_id TEXT NOT NULL REFERENCES funds(id),
    side TEXT NOT NULL CHECK (side IN ('SUBSCRIPTION', 'REDEMPTION')),
    amount BIGINT NOT NULL DEFAULT 0,
    units BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('RECEIVED', 'PRICED', 'CANCELLED', 'REJECTED')),
    trade_date DATE NOT NULL,
    nav_used BIGINT NOT NULL DEFAULT 0,
    priced_units BIGINT NOT NULL DEFAULT 0,
    priced_amount BIGINT NOT NULL DEFAULT 0,
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_account_status ON orders(account_id, status, created_at, id);
CREATE INDEX idx_orders_fund_status_trade_date ON orders(fund_id, status, trade_date);

CREATE TABLE order_events (
    id BIGSERIAL PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('RECEIVED', 'CANCELLED', 'PRICED', 'REJECTED')),
    status TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    nav_used BIGINT NOT NULL DEFAULT 0,
    priced_units BIGINT NOT NULL DEFAULT 0,
    priced_amount BIGINT NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_order_events_order_id ON order_events(order_id, timestamp, id);

CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE nav_prices (
    fund_id TEXT NOT NULL REFERENCES funds(id) ON DELETE CASCADE,
    trade_date DATE NOT NULL,
    nav BIGINT NOT NULL,
    PRIMARY KEY (fund_id, trade_date)
);
