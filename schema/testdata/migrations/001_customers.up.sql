BEGIN TRANSACTION;

CREATE TABLE customers (
    customer_id  UUID  NOT NULL,
    email        TEXT  NOT NULL,
    CONSTRAINT pk_customers PRIMARY KEY (customer_id)
);

GRANT SELECT, INSERT, UPDATE ON customers TO :"app";

COMMIT TRANSACTION;
