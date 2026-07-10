-- +migrate Down
DROP TABLE IF EXISTS delivery_order_items CASCADE;
DROP TABLE IF EXISTS delivery_orders CASCADE;
