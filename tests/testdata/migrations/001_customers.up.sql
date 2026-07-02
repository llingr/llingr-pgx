BEGIN TRANSACTION;


CREATE TABLE customers (
    customer_id  UUID        NOT NULL,
    email        TEXT        NOT NULL,
    full_name    TEXT        NOT NULL,
    created_ts   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_customers_customer_id PRIMARY KEY (customer_id)
);

CREATE UNIQUE INDEX uidx_customers_email
    ON customers (email);

CREATE INDEX idx_customers_full_name
    ON customers (full_name);


GRANT SELECT, INSERT, UPDATE, DELETE
    ON customers TO :"app";

GRANT SELECT
    ON customers TO :"readonly";


COMMIT TRANSACTION;
