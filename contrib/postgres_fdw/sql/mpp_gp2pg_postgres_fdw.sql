-- This file is used to test the feature that there are multiple remote postgres servers.

-- ===================================================================
-- create FDW objects
-- ===================================================================
SET timezone = 'PST8PDT';

-- Clean
-- start_ignore
DROP EXTENSION IF EXISTS postgres_fdw CASCADE;
-- end_ignore

CREATE EXTENSION postgres_fdw;

CREATE SERVER pgserver FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (host 'dummy', port '0',
           dbname 'contrib_regression', multi_hosts 'localhost  localhost',
           multi_ports '5432  5555', num_segments '2', mpp_execute 'all segments');

CREATE USER MAPPING FOR CURRENT_USER SERVER pgserver;

-- ===================================================================
-- create objects used through FDW pgserver server
-- ===================================================================
-- remote postgres server 1 -- listening port 5432
\! env PGOPTIONS='' psql -p 5432 contrib_regression -f sql/postgres_sql/mpp_gp2pg_postgres_init_1.sql
-- remote postgres server 2 -- listening port 5555
\! env PGOPTIONS='' psql -p 5555 contrib_regression -f sql/postgres_sql/mpp_gp2pg_postgres_init_2.sql

-- ===================================================================
-- create foreign tables
-- ===================================================================
CREATE FOREIGN TABLE mpp_ft1 (
	c1 int,
	c2 int
) SERVER pgserver OPTIONS (schema_name 'MPP_S 1', table_name 'T 1');

-- ===================================================================
-- tests for validator
-- ===================================================================
-- Error when the length of option multi_hosts and multi_ports is NOT same.
CREATE SERVER testserver FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (dbname 'contrib_regression', multi_hosts 'localhost localhost',
           multi_ports '5432', num_segments '2', mpp_execute 'all segments');
-- Error when specifying option multi_hosts and multi_ports but option mpp_execute is NOT 'all segments'.
CREATE FOREIGN TABLE mpp_test (
	c1 int,
	c2 int
) SERVER pgserver OPTIONS (schema_name 'MPP_S 1', table_name 'T 1', mpp_execute 'coordinator');
SELECT * FROM mpp_test;
ALTER FOREIGN TABLE mpp_test OPTIONS (drop mpp_execute);
-- Error when the value of option num_segments is NOT same as the length of option multi_hosts and multi_ports.
ALTER SERVER pgserver OPTIONS (set num_segments '1');
SELECT * FROM mpp_test;
ALTER SERVER pgserver OPTIONS (set num_segments '2');
-- ===================================================================
-- Simple queries
-- ===================================================================
EXPLAIN VERBOSE SELECT * FROM mpp_ft1 ORDER BY c1;
SELECT * FROM mpp_ft1 ORDER BY c1;

ALTER FOREIGN TABLE mpp_ft1 OPTIONS (add use_remote_estimate 'true');
EXPLAIN VERBOSE SELECT * FROM mpp_ft1 ORDER BY c1;
SELECT * FROM mpp_ft1 ORDER BY c1;
ALTER FOREIGN TABLE mpp_ft1 OPTIONS (drop use_remote_estimate);

-- ===================================================================
-- When there are multiple remote servers, we don't support IMPORT FOREIGN SCHEMA
-- ===================================================================
CREATE SCHEMA mpp_import_dest;
IMPORT FOREIGN SCHEMA import_source FROM SERVER pgserver INTO mpp_import_dest;

-- ===================================================================
-- When there are multiple remote servers, we don't support INSERT/UPDATE/DELETE
-- ===================================================================
INSERT INTO mpp_ft1 VALUES (1, 1);

UPDATE mpp_ft1 SET c1 = c1 + 1;

DELETE FROM mpp_ft1;
