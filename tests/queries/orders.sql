-- Ecommerce read/write queries, loaded by the lexicon package.
--
-- House style on display here: every statement uses pgx *named* parameters,
-- named in snake_case to mirror the column (@customer_id, never positional $1/?
-- and never @customerId); joins use explicit JOIN ... ON; and primary-key UUIDs
-- are generated in Go (google/uuid) and bound as parameters, so the inserts need
-- no gen_random_uuid() or RETURNING and stay portable.

-- name: insert-customer
INSERT INTO customers (customer_id, email, full_name)
VALUES (@customer_id, @email, @full_name);

-- name: insert-order
INSERT INTO customer_orders
    (customer_order_id, customer_id, currency, amount, amount_minor_units, order_status)
VALUES
    (@customer_order_id, @customer_id, @currency, @amount, @amount_minor_units, @order_status);

-- name: order-with-customer
SELECT o.customer_order_id AS order_id,
       o.customer_id       AS customer_id,
       c.full_name         AS customer_name,
       c.email             AS customer_email,
       o.currency          AS currency,
       cc.name             AS currency_name,
       o.amount            AS amount,
       o.order_status      AS order_status
FROM   customer_orders o
JOIN
       customers       c
    ON o.customer_id = c.customer_id
JOIN
       currency_codes  cc
    ON o.currency = cc.currency
WHERE  o.customer_order_id = @customer_order_id;
