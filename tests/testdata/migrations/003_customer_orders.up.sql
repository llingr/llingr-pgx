BEGIN TRANSACTION;


CREATE TABLE customer_orders (
    customer_order_id   UUID        NOT NULL,
    customer_id         UUID        NOT NULL,
    currency            TEXT        NOT NULL,  -- e.g. USD
    amount              BIGINT      NOT NULL,  -- e.g. 1399 for USD 13.99
    amount_minor_units  INTEGER     NOT NULL,  -- e.g. 2 for USD 13.99
    order_status        TEXT        NOT NULL,
    created_ts          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_customer_orders PRIMARY KEY (customer_order_id)
);

ALTER TABLE customer_orders
    ADD CONSTRAINT fk_customer_orders_customer_id
    FOREIGN KEY (customer_id) REFERENCES customers (customer_id);

ALTER TABLE customer_orders
    ADD CONSTRAINT fk_customer_orders_currency
    FOREIGN KEY (currency) REFERENCES currency_codes (currency);


CREATE INDEX idx_customer_orders_customer_id
    ON customer_orders (customer_id);


ALTER TABLE customer_orders
    ADD CHECK (amount >= 0);

ALTER TABLE customer_orders
    ADD CHECK (amount_minor_units >= 0 AND amount_minor_units <= 100);


GRANT SELECT, INSERT, UPDATE
    ON customer_orders TO :"app";

GRANT SELECT
    ON customer_orders TO :"readonly";


COMMIT TRANSACTION;
