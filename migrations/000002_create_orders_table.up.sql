CREATE TABLE IF NOT EXISTS orders (
    order_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'NEW',
    accrual NUMERIC,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
    UNIQUE (user_id, number)
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);