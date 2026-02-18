CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_name TEXT NOT NULL,
    amount INT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at DESC);