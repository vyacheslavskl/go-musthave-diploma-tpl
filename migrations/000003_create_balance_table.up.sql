CREATE TABLE IF NOT EXISTS balance_transactions (
    transaction_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(order_id),
    -- +accrual, -withdrawal
    amount NUMERIC NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bt_user_id ON balance_transactions(user_id);

CREATE INDEX IF NOT EXISTS idx_bt_order_id ON balance_transactions(order_id);