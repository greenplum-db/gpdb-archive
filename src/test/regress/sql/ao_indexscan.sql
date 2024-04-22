-- Test index scans on append-optimized tables. We mainly test the plans being
-- generated, in addition to smoke testing the output if Index Scan is exercised.

-- Turn off ORCA as it doesn't yet support Index Scans on AO/CO tables.
set optimizer to off;

-- Create an uncompressed test ao_row table
create table aoindexscantab_uncomp (i int4, j int4) using ao_row;
insert into aoindexscantab_uncomp select g, g % 10000 from generate_series(1, 10000000) g;
create index on aoindexscantab_uncomp(j);
create index on aoindexscantab_uncomp(i);
analyze aoindexscantab_uncomp;

-- A simple key-value lookup query. Should use an Index scan.
explain (costs off) select i, j from aoindexscantab_uncomp where i = 90;
select i, j from aoindexscantab_uncomp where i = 90;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select i from aoindexscantab_uncomp where i = 90;

-- BitmapScan should still be preferred when selectivity is higher.
explain (costs off) select * from aoindexscantab_uncomp where i < 1000000;

-- Should use an Index Scan as an ordering operator when limit is specified.
explain (costs off) select * from aoindexscantab_uncomp order by i limit 5;
select * from aoindexscantab_uncomp order by i limit 5;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select j from aoindexscantab_uncomp order by j limit 15;

-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan. But
-- IndexOnly Scans continue to be picked.
set gp_enable_ao_indexscan to off;
explain (costs off) select i, j from aoindexscantab_uncomp where i = 90;
explain (costs off) select i from aoindexscantab_uncomp where i = 90;
reset gp_enable_ao_indexscan;

-- Create a compressed test ao_row table
create table aoindexscantab (i int4, j int4) with (appendonly=true, compresstype=zstd);
insert into aoindexscantab select g, g % 10000 from generate_series(1, 10000000) g;
create index on aoindexscantab(j);
create index on aoindexscantab(i);
analyze aoindexscantab;

-- A simple key-value lookup query. Should use an Index scan.
explain (costs off) select i, j from aoindexscantab where i = 90;
select i, j from aoindexscantab where i = 90;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select i from aoindexscantab where i = 90;

-- BitmapScan should still be preferred when selectivity is higher (we are more
-- pessimistic towards IndexScans when the table is compressed, which is why the
-- predicate is 10x lower here as compared to the uncompressed case).
explain (costs off) select * from aoindexscantab where i < 100000;

-- Should use an Index Scan as an ordering operator when limit is specified.
explain (costs off) select * from aoindexscantab order by i limit 5;
select * from aoindexscantab order by i limit 5;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select j from aoindexscantab order by j limit 15;

-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan. But
-- IndexOnly Scans continue to be picked.
set gp_enable_ao_indexscan to off;
explain (costs off) select i, j from aoindexscantab where i = 90;
explain (costs off) select i from aoindexscantab where i = 90;
reset gp_enable_ao_indexscan;

-- Create an uncompressed test ao_column table
create table aocsindexscantab_uncomp (i int4, j int4) using ao_column;
insert into aocsindexscantab_uncomp select g, g % 10000 from generate_series(1, 10000000) g;
create index on aocsindexscantab_uncomp(j);
create index on aocsindexscantab_uncomp(i);
analyze aocsindexscantab_uncomp;

-- A simple key-value lookup query. Should use an Index scan.
explain (costs off) select i, j from aocsindexscantab_uncomp where i = 90;
select i, j from aocsindexscantab_uncomp where i = 90;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select i from aocsindexscantab_uncomp where i = 90;

-- BitmapScan should still be preferred when selectivity is higher.
explain (costs off) select * from aocsindexscantab_uncomp where i < 1000000;

-- Should use an Index Scan as an ordering operator when limit is specified.
explain (costs off) select * from aocsindexscantab_uncomp order by i limit 5;
select * from aocsindexscantab_uncomp order by i limit 5;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select j from aocsindexscantab_uncomp order by j limit 15;

-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan.
-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan. But
-- IndexOnly Scans continue to be picked.
set gp_enable_ao_indexscan to off;
explain (costs off) select i, j from aocsindexscantab_uncomp where i = 90;
explain (costs off) select i from aocsindexscantab_uncomp where i = 90;
reset gp_enable_ao_indexscan;

-- Create a compressed test ao_column table
create table aocsindexscantab (i int4, j int4) with (appendonly=true, orientation=column, compresstype=zstd);
insert into aocsindexscantab select g, g % 10000 from generate_series(1, 10000000) g;
create index on aocsindexscantab(j);
create index on aocsindexscantab(i);
analyze aocsindexscantab;

-- A simple key-value lookup query. Should use an Index scan.
explain (costs off) select i, j from aocsindexscantab where i = 90;
select i, j from aocsindexscantab where i = 90;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select i from aocsindexscantab where i = 90;

-- Should use an Index Scan as an ordering operator when limit is specified.
explain (costs off) select * from aocsindexscantab order by i limit 5;
select * from aocsindexscantab order by i limit 5;

-- IndexOnlyScan should still be preferred when only the index key is involved.
explain (costs off) select j from aocsindexscantab order by j limit 15;

-- BitmapScan should still be preferred when selectivity is higher (we are more
-- pessimistic towards IndexScans when the table is compressed, which is why the
-- predicate is 10x lower here as compared to the uncompressed case).
explain (costs off) select * from aocsindexscantab where i < 100000;

-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan.
-- When gp_enable_ao_indexscan is off, we should not pick an Index Scan. But
-- IndexOnly Scans continue to be picked.
set gp_enable_ao_indexscan to off;
explain (costs off) select i, j from aocsindexscantab where i = 90;
explain (costs off) select i from aocsindexscantab where i = 90;
reset gp_enable_ao_indexscan;
