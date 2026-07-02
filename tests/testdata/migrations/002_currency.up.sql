BEGIN TRANSACTION;


CREATE TABLE currency_codes (
    currency       TEXT        NOT NULL,
    name           TEXT        NOT NULL,
    minor_units    INTEGER     NULL,  -- e.g. 2 for USD 13.99
    CONSTRAINT pk_currency_codes PRIMARY KEY (currency)
);

CREATE UNIQUE INDEX currency_codes_name
    ON currency_codes (name);

ALTER TABLE currency_codes
    ADD CONSTRAINT chk_currency_codes_minor_units
    CHECK (minor_units >= 0 AND minor_units <= 100);


INSERT INTO currency_codes (currency, name, minor_units)
    VALUES ('AUD', 'Australian dollar', 2),
           ('CAD', 'Canadian dollar', 2),
           ('CHF', 'Swiss franc', 2),
           ('CNY', 'Yuan renminbi', 2),
           ('EUR', 'Euro', 2),
           ('GBP', 'Pound sterling', 2),
           ('HKD', 'Hong Kong dollar', 2),
           ('INR', 'Indian rupee', 2),
           ('JPY', 'Yen', 0),
           ('KRW', 'Won', 0),
           ('NOK', 'Norwegian krone', 2),
           ('NZD', 'New Zealand dollar', 2),
           ('SEK', 'Swedish krona', 2),
           ('SGD', 'Singapore dollar', 2),
           ('USD', 'United States dollar', 2),
           ('XAU', 'Gold', NULL),
           ('XPD', 'Palladium', NULL);


GRANT SELECT, INSERT
    ON currency_codes TO :"app";

GRANT SELECT
    ON currency_codes TO :"readonly";


COMMIT TRANSACTION;
