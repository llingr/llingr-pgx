BEGIN TRANSACTION;

CREATE USER ecommerce_schema_owner
       WITH PASSWORD 'tester';

CREATE USER ecommerce_app_user
       WITH PASSWORD 'tester';

CREATE USER ecommerce_readonly_user
       WITH PASSWORD 'tester';

ALTER SCHEMA public
      OWNER TO ecommerce_schema_owner;

COMMIT TRANSACTION;
