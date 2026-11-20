CREATE TABLE IF NOT EXISTS orders (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id        TEXT        NOT NULL,
    restaurant_id      TEXT        NOT NULL,
    status             TEXT        NOT NULL DEFAULT 'PLACED',
    total_amount_paise BIGINT      NOT NULL CHECK (total_amount_paise >= 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
    id               BIGSERIAL PRIMARY KEY,
    order_id         UUID    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    item_id          TEXT    NOT NULL,
    name             TEXT    NOT NULL,
    quantity         INT     NOT NULL CHECK (quantity > 0),
    unit_price_paise BIGINT  NOT NULL CHECK (unit_price_paise >= 0)
);

CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);