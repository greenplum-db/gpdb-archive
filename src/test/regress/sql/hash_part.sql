--
-- Hash partitioning.
--

-- Use hand-rolled hash functions and operator classes to get predictable
-- result on different machines.  See the definitions of
-- part_part_test_int4_ops and part_test_text_ops in insert.sql.

CREATE TABLE mchash (a int, b text, c jsonb)
  PARTITION BY HASH (a part_test_int4_ops, b part_test_text_ops);
CREATE TABLE mchash1
  PARTITION OF mchash FOR VALUES WITH (MODULUS 4, REMAINDER 0);

-- invalid OID, no such table
SELECT satisfies_hash_partition(0, 4, 0, NULL);

-- not partitioned
SELECT satisfies_hash_partition('tenk1'::regclass, 4, 0, NULL);

-- partition rather than the parent
SELECT satisfies_hash_partition('mchash1'::regclass, 4, 0, NULL);

-- invalid modulus
SELECT satisfies_hash_partition('mchash'::regclass, 0, 0, NULL);

-- remainder too small
SELECT satisfies_hash_partition('mchash'::regclass, 1, -1, NULL);

-- remainder too large
SELECT satisfies_hash_partition('mchash'::regclass, 1, 1, NULL);

-- modulus is null
SELECT satisfies_hash_partition('mchash'::regclass, NULL, 0, NULL);

-- remainder is null
SELECT satisfies_hash_partition('mchash'::regclass, 4, NULL, NULL);

-- too many arguments
SELECT satisfies_hash_partition('mchash'::regclass, 4, 0, NULL::int, NULL::text, NULL::json);

-- too few arguments
SELECT satisfies_hash_partition('mchash'::regclass, 3, 1, NULL::int);

-- wrong argument type
SELECT satisfies_hash_partition('mchash'::regclass, 2, 1, NULL::int, NULL::int);

-- ok, should be false
SELECT satisfies_hash_partition('mchash'::regclass, 4, 0, 0, ''::text);

-- ok, should be true
SELECT satisfies_hash_partition('mchash'::regclass, 4, 0, 2, ''::text);

-- argument via variadic syntax, should fail because not all partitioning
-- columns are of the correct type
SELECT satisfies_hash_partition('mchash'::regclass, 2, 1,
								variadic array[1,2]::int[]);

-- multiple partitioning columns of the same type
CREATE TABLE mcinthash (a int, b int, c jsonb)
  PARTITION BY HASH (a part_test_int4_ops, b part_test_int4_ops);

-- now variadic should work, should be false
SELECT satisfies_hash_partition('mcinthash'::regclass, 4, 0,
								variadic array[0, 0]);

-- should be true
SELECT satisfies_hash_partition('mcinthash'::regclass, 4, 0,
								variadic array[0, 1]);

-- wrong length
SELECT satisfies_hash_partition('mcinthash'::regclass, 4, 0,
								variadic array[]::int[]);

-- wrong type
SELECT satisfies_hash_partition('mcinthash'::regclass, 4, 0,
								variadic array[now(), now()]);

-- check satisfies_hash_partition passes correct collation
create table text_hashp (a text) partition by hash (a);
create table text_hashp0 partition of text_hashp for values with (modulus 2, remainder 0);
create table text_hashp1 partition of text_hashp for values with (modulus 2, remainder 1);
-- The result here should always be true, because 'xxx' must belong to
-- one of the two defined partitions
select satisfies_hash_partition('text_hashp'::regclass, 2, 0, 'xxx'::text) OR
	   satisfies_hash_partition('text_hashp'::regclass, 2, 1, 'xxx'::text) AS satisfies;

-- cleanup
DROP TABLE mchash;
DROP TABLE mcinthash;
DROP TABLE text_hashp;

-- Test case for AO Hash partitioning table
-- https://github.com/greenplum-db/gpdb/pull/17280
CREATE TABLE tbl_17280(c0 int) PARTITION BY HASH(c0) WITH (appendonly=true);
CREATE INDEX idx_17280 ON tbl_17280 USING HASH(c0) WHERE (c0!=0);
-- should not panic
SELECT count(*) FROM tbl_17280;

create table tbl_17280_p1 partition of tbl_17280 for values with (modulus 2, remainder 0) WITH (appendonly=true);
create table tbl_17280_p2 partition of tbl_17280 for values with (modulus 2, remainder 1) WITH (appendonly=true);
insert into tbl_17280 select generate_series(1,1000);
set optimizer to off;
set enable_seqscan to off;
analyze tbl_17280_p1;
-- should use bitmap index scan
explain select c0 from tbl_17280_p1 where c0=12;

DROP TABLE tbl_17280;
