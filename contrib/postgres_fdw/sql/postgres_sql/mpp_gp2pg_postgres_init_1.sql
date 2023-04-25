-- This sql file is used by mpp_gp2pg_postgres_fdw test, and it runs in
-- postgres server.

-- ===================================================================
-- create objects used through FDW pgserver server
-- ===================================================================
SET timezone = 'PST8PDT';

CREATE SCHEMA "MPP_S 1";
CREATE TABLE "MPP_S 1"."T 1" (
	c1 int,
	c2 int
);

-- Disable autovacuum for these tables to avoid unexpected effects of that
ALTER TABLE "MPP_S 1"."T 1" SET (autovacuum_enabled = 'false');

INSERT INTO "MPP_S 1"."T 1"
	SELECT id,
	       id % 2
	FROM generate_series(1, 5) id;

ANALYZE "MPP_S 1"."T 1";